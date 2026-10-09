package transactions

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/budgets"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
	"wallet-api/internal/notify"
)

type transferRequest struct {
	FromAccountID int   `json:"from_account_id"`
	ToAccountID   int   `json:"to_account_id"`
	Amount        int64 `json:"amount"`
	// ToAmount farklı para birimli hesaplar arasında giren tutar (kur uygulanmış).
	ToAmount    *int64     `json:"to_amount"`
	Description *string    `json:"description"`
	OccurredAt  *time.Time `json:"occurred_at"`
}

// CreateTransfer kendi iki hesabı arasında para aktarır: kaynak hesapta gider,
// hedef hesapta gelir oluşur ve birbirine bağlanır. Transferler gelir/gider
// toplamlarına ve bütçelere sayılmaz. Yanıt: kaynak bacak.
func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	var req transferRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.FromAccountID == req.ToAccountID {
		httpx.WriteError(w, http.StatusBadRequest, "kaynak ve hedef hesap ayni olamaz")
		return
	}
	ctx := r.Context()
	userID := auth.UserID(ctx)

	var fromCur, toCur, fromName, toName string
	err := h.DB.QueryRow(ctx, `
SELECT f.currency, t.currency, f.name, t.name FROM accounts f, accounts t
WHERE f.id = $1 AND t.id = $2 AND f.user_id = $3 AND t.user_id = $3`,
		req.FromAccountID, req.ToAccountID, userID).Scan(&fromCur, &toCur, &fromName, &toName)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}
	toAmount := req.Amount
	if fromCur != toCur {
		if req.ToAmount == nil {
			httpx.WriteError(w, http.StatusBadRequest, "farkli para birimleri icin to_amount gerekli")
			return
		}
		toAmount = *req.ToAmount
	}
	desc := req.Description
	if desc == nil || strings.TrimSpace(*desc) == "" {
		d := fromName + " → " + toName
		desc = &d
	}
	at := time.Now()
	if req.OccurredAt != nil {
		at = *req.OccurredAt
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "transfer begin", err)
		return
	}
	defer tx.Rollback(ctx)
	// Kilit sırası sabit (küçük id önce): iki ters yönlü eşzamanlı transfer kilitlenmesin.
	for _, id := range []int{min(req.FromAccountID, req.ToAccountID), max(req.FromAccountID, req.ToAccountID)} {
		if _, err := tx.Exec(ctx, "SELECT 1 FROM accounts WHERE id = $1 FOR UPDATE", id); err != nil {
			httpx.ServerError(w, "transfer lock", err)
			return
		}
	}
	outID, err := ledger.Create(ctx, tx, userID, ledger.Input{
		AccountID: req.FromAccountID, Type: "expense", Amount: req.Amount, Description: desc, OccurredAt: at,
	})
	if err != nil {
		httpx.Fail(w, "transfer out", err)
		return
	}
	inID, err := ledger.Create(ctx, tx, userID, ledger.Input{
		AccountID: req.ToAccountID, Type: "income", Amount: toAmount, Description: desc, OccurredAt: at,
	})
	if err != nil {
		httpx.Fail(w, "transfer in", err)
		return
	}
	_, err = tx.Exec(ctx, `
UPDATE transactions SET transfer_peer_id = CASE id WHEN $1 THEN $2 ELSE $1 END WHERE id IN ($1, $2)`, outID, inID)
	if err != nil {
		httpx.ServerError(w, "transfer link", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "transfer commit", err)
		return
	}
	h.writeOne(w, r, outID, http.StatusCreated)
}

// afterCreate yeni işlem için bildirimleri gönderir: bütçe uyarısı ve grup
// harcamasında diğer üyelere haber. İstek bittikten sonra arka planda çalışır.
func (h *Handler) afterCreate(_ context.Context, txID int) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		budgets.Check(ctx, h.DB, h.Notify, h.Loc, txID)
		NotifyGroupExpense(ctx, h.DB, h.Notify, txID)
	}()
}

// NotifyGroupExpense grup işlemini ödeyen dışındaki üyelere, kendi paylarıyla bildirir.
// Tekrarlayan kuraldan gelen işlemlerde de kullanılır.
func NotifyGroupExpense(ctx context.Context, db *pgxpool.Pool, n *notify.Notifier, txID int) {
	var groupID, payerID int
	var payer, group, currency string
	var desc, category *string
	var amount int64
	err := db.QueryRow(ctx, `
SELECT t.group_id, a.user_id, u.username, g.name, a.currency, t.description, t.category, t.amount
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN users u ON u.id = a.user_id
JOIN groups g ON g.id = t.group_id
WHERE t.id = $1 AND t.type = 'expense'`, txID).Scan(&groupID, &payerID, &payer, &group, &currency, &desc, &category, &amount)
	if err != nil {
		return // grup gideri değil
	}
	rows, err := db.Query(ctx, `
SELECT gm.user_id, COALESCE(s.amount, 0)
FROM group_members gm
LEFT JOIN transaction_splits s ON s.transaction_id = $1 AND s.user_id = gm.user_id
WHERE gm.group_id = $2 AND gm.user_id <> $3`, txID, groupID, payerID)
	if err != nil {
		log.Println("group expense notify:", err)
		return
	}
	type share struct {
		user  int
		share int64
	}
	var shares []share
	for rows.Next() {
		var s share
		if rows.Scan(&s.user, &s.share) == nil {
			shares = append(shares, s)
		}
	}
	rows.Close()
	what := "Harcama"
	if desc != nil && *desc != "" {
		what = *desc
	} else if category != nil {
		what = budgets.CategoryLabel(*category)
	}
	sym := map[string]string{"TRY": "₺", "USD": "$", "EUR": "€", "GBP": "£"}[currency]
	for _, s := range shares {
		body := fmt.Sprintf("%s ödedi: %s %s", payer, ledger.FormatMoney(amount), sym)
		if s.share > 0 {
			body += fmt.Sprintf(" · senin payın %s %s", ledger.FormatMoney(s.share), sym)
		}
		n.Send(ctx, notify.Notice{
			UserID: s.user, Kind: "group_expense", RefID: &txID,
			Title: group + ": " + what, Body: body, URL: "/groups/" + strconv.Itoa(groupID),
		})
	}
}

// searchTerms aramayı LIKE kalıbına çevirir; metin bir tutarsa kuruş olarak da döner.
func searchTerms(q string) (*string, *int64) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if len([]rune(q)) > 64 {
		q = string([]rune(q)[:64])
	}
	esc := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(q))
	like := "%" + esc + "%"
	var amount *int64
	norm := strings.ReplaceAll(strings.ReplaceAll(q, ".", ""), ",", ".")
	if f, err := strconv.ParseFloat(norm, 64); err == nil && f > 0 && f < 1e10 {
		k := int64(f*100 + 0.5)
		amount = &k
	}
	return &like, amount
}
