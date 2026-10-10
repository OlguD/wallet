package groups

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
	"wallet-api/internal/notify"
)

var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

type Settlement struct {
	ID            int       `json:"id"`
	GroupID       int       `json:"group_id"`
	GroupName     string    `json:"group_name"`
	FromUserID    int       `json:"from_user_id"`
	FromUsername  string    `json:"from_username"`
	ToUserID      int       `json:"to_user_id"`
	ToUsername    string    `json:"to_username"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	TransactionID *int      `json:"transaction_id"`
	Note          *string   `json:"note"`
	CreatedBy     int       `json:"created_by"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
	// CounterStatus karşı tarafın (kaydı girmeyen) durumu: none, pending, booked, dismissed.
	CounterStatus string `json:"counter_status"`
}

const selectSettlement = `
SELECT s.id, s.group_id, g.name, s.from_user_id, fu.username, s.to_user_id, tu.username,
       s.amount, s.currency, s.transaction_id, s.note, s.created_by, s.occurred_at, s.created_at,
       s.counter_status
FROM settlements s
JOIN groups g ON g.id = s.group_id
JOIN users fu ON fu.id = s.from_user_id
JOIN users tu ON tu.id = s.to_user_id
`

func (h *Handler) ListSettlements(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(r)
	rows, err := h.DB.Query(r.Context(),
		selectSettlement+"WHERE s.group_id = $1 ORDER BY s.occurred_at DESC, s.id DESC LIMIT $2 OFFSET $3",
		id, limit, offset,
	)
	if err != nil {
		httpx.ServerError(w, "settlements list", err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Settlement])
	if err != nil {
		httpx.ServerError(w, "settlements scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

type settlementRequest struct {
	FromUserID int    `json:"from_user_id"`
	ToUserID   int    `json:"to_user_id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	AccountID  *int   `json:"account_id"`
	// AccountAmount hesabın para birimi ödemeninkinden farklıysa hesaba
	// işlenecek tutar (kur uygulanmış, hesabın para biriminde).
	AccountAmount *int64     `json:"account_amount"`
	Note          *string    `json:"note"`
	OccurredAt    *time.Time `json:"occurred_at"`
}

// CreateSettlement bir üyenin diğerine yaptığı ödemeyi kaydeder. Kaydı ödeyen
// ya da alan taraf girebilir. account_id verilirse ödeme, kaydı girenin o
// hesabına da işlenir (ödeyense gider, alansa gelir).
func (h *Handler) CreateSettlement(w http.ResponseWriter, r *http.Request) {
	groupID, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req settlementRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	userID := auth.UserID(ctx)

	if req.FromUserID != userID && req.ToUserID != userID {
		httpx.WriteError(w, http.StatusBadRequest, "odemenin tarafi siz olmalisiniz")
		return
	}
	if req.FromUserID == req.ToUserID {
		httpx.WriteError(w, http.StatusBadRequest, "odeyen ve alan ayni kisi olamaz")
		return
	}
	if err := ledger.ValidateAmount(req.Amount); err != nil {
		httpx.Fail(w, "settlement", err)
		return
	}
	note, err := ledger.NormalizeDescription(req.Note)
	if err != nil {
		httpx.Fail(w, "settlement", err)
		return
	}
	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "settlement begin", err)
		return
	}
	defer tx.Rollback(ctx)

	// İki tarafın üyelik satırları kilitlenir; işlem sürerken gruptan ayrılamazlar.
	rows, err := tx.Query(ctx,
		"SELECT user_id FROM group_members WHERE group_id = $1 AND user_id = ANY($2) FOR SHARE",
		groupID, []int{req.FromUserID, req.ToUserID},
	)
	if err != nil {
		httpx.ServerError(w, "settlement members", err)
		return
	}
	found, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		httpx.ServerError(w, "settlement members scan", err)
		return
	}
	if !slices.Contains(found, userID) {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}
	if len(found) < 2 {
		httpx.WriteError(w, http.StatusBadRequest, "iki taraf da grubun uyesi olmali")
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	var txID *int
	if req.AccountID != nil {
		var accCurrency string
		err = tx.QueryRow(ctx,
			"SELECT currency FROM accounts WHERE id = $1 AND user_id = $2",
			*req.AccountID, userID,
		).Scan(&accCurrency)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
			return
		}
		if err != nil {
			httpx.ServerError(w, "settlement account", err)
			return
		}
		if currency == "" {
			currency = accCurrency
		}
		bookAmount, err := accountAmount(req.Amount, currency, accCurrency, req.AccountAmount)
		if err != nil {
			httpx.Fail(w, "settlement", err)
			return
		}

		var groupName, fromName, toName string
		err = tx.QueryRow(ctx, `
SELECT (SELECT name FROM groups WHERE id = $1),
       (SELECT username FROM users WHERE id = $2),
       (SELECT username FROM users WHERE id = $3)`,
			groupID, req.FromUserID, req.ToUserID,
		).Scan(&groupName, &fromName, &toName)
		if err != nil {
			httpx.ServerError(w, "settlement names", err)
			return
		}

		typ := "income"
		if req.FromUserID == userID {
			typ = "expense"
		}
		desc := settlementDesc(groupName, fromName, toName, req.Amount, currency, accCurrency)
		id, err := ledger.Create(ctx, tx, userID, ledger.Input{
			AccountID:   *req.AccountID,
			Type:        typ,
			Amount:      bookAmount,
			Description: &desc,
			OccurredAt:  occurredAt,
		})
		if err != nil {
			httpx.Fail(w, "settlement transaction", err)
			return
		}
		txID = &id
	}
	if currency == "" {
		currency = "TRY"
	}
	if !currencyRe.MatchString(currency) {
		httpx.WriteError(w, http.StatusBadRequest, "para birimi 3 harfli ISO 4217 kodu olmali (orn. TRY)")
		return
	}

	var id int
	err = tx.QueryRow(ctx, `
INSERT INTO settlements (group_id, from_user_id, to_user_id, amount, currency, transaction_id, note, created_by, occurred_at, counter_status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending') RETURNING id`,
		groupID, req.FromUserID, req.ToUserID, req.Amount, currency, txID, note, userID, occurredAt,
	).Scan(&id)
	if err != nil {
		httpx.ServerError(w, "settlement insert", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "settlement commit", err)
		return
	}

	rows, err = h.DB.Query(ctx, selectSettlement+"WHERE s.id = $1", id)
	if err != nil {
		httpx.ServerError(w, "settlement get", err)
		return
	}
	s, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[Settlement])
	if err != nil {
		httpx.ServerError(w, "settlement get scan", err)
		return
	}
	h.notifyCounterparty(ctx, s)
	httpx.WriteJSON(w, http.StatusCreated, s)
}

// notifyCounterparty kaydı girmeyen tarafa ödemeyi haber verir; gelen
// kutusundan kendi hesabına işleyebilir.
func (h *Handler) notifyCounterparty(ctx context.Context, s Settlement) {
	amount := ledger.FormatMoney(s.Amount) + " " + s.Currency
	n := notify.Notice{Kind: "settlement", RefID: &s.ID, URL: "/inbox"}
	if s.CreatedBy == s.FromUserID {
		n.UserID = s.ToUserID
		n.Title = s.FromUsername + " sana " + amount + " gönderdi"
		n.Body = "\"" + s.GroupName + "\" grubu. Gelen kutusundan hangi hesabına geldiğini seç."
	} else {
		n.UserID = s.FromUserID
		n.Title = s.ToUsername + " senden " + amount + " aldığını kaydetti"
		n.Body = "\"" + s.GroupName + "\" grubu. Gelen kutusundan hangi hesabından çıktığını seç."
	}
	h.Notify.Send(ctx, n)
}

// PendingSettlements kullanıcının hesabına işlemesi beklenen ödemeler: başka
// bir üyenin girdiği ve kullanıcının taraf olduğu hesaplaşmalar.
func (h *Handler) PendingSettlements(w http.ResponseWriter, r *http.Request) {
	me := auth.UserID(r.Context())
	rows, err := h.DB.Query(r.Context(), selectSettlement+`
WHERE s.counter_status = 'pending' AND $1 IN (s.from_user_id, s.to_user_id) AND s.created_by <> $1
ORDER BY s.occurred_at DESC, s.id DESC`, me)
	if err != nil {
		httpx.ServerError(w, "pending settlements", err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Settlement])
	if err != nil {
		httpx.ServerError(w, "pending settlements scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

type bookRequest struct {
	AccountID int `json:"account_id"`
	// AccountAmount hesabın para birimi farklıysa hesaba işlenecek tutar.
	AccountAmount *int64 `json:"account_amount"`
}

// accountAmount ödemenin hesaba işlenecek tutarını verir: para birimleri
// aynıysa ödeme tutarı, farklıysa istemcinin kurla hesapladığı tutar.
func accountAmount(amount int64, currency, accCurrency string, given *int64) (int64, error) {
	if currency == accCurrency {
		return amount, nil
	}
	if given == nil {
		return 0, httpx.BadRequest("farkli para birimli hesap icin account_amount gerekli")
	}
	if err := ledger.ValidateAmount(*given); err != nil {
		return 0, err
	}
	return *given, nil
}

// settlementDesc hesaba yazılan açıklama; dövizli ödemede asıl tutar da yazılır.
func settlementDesc(group, from, to string, amount int64, currency, accCurrency string) string {
	d := fmt.Sprintf("Hesaplasma (%s): %s -> %s", group, from, to)
	if currency != accCurrency {
		d += " · " + ledger.FormatMoney(amount) + " " + currency
	}
	return d
}

// BookSettlement karşı taraf ödemeyi kendi hesabına işler: alansa gelir,
// ödeyense gider. Hesabın para birimi ödemeninkiyle aynı olmalı.
func (h *Handler) BookSettlement(w http.ResponseWriter, r *http.Request) {
	h.resolveCounter(w, r, true)
}

// DismissSettlement ödemeyi hesaba işlemeden kapatır (ör. zaten elle girildi).
// Grup borcu etkilenmez; hesaplaşma kaydı durur.
func (h *Handler) DismissSettlement(w http.ResponseWriter, r *http.Request) {
	h.resolveCounter(w, r, false)
}

func (h *Handler) resolveCounter(w http.ResponseWriter, r *http.Request, book bool) {
	sid, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req bookRequest
	if book && !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	me := auth.UserID(ctx)

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "settlement counter begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var s Settlement
	err = tx.QueryRow(ctx, `
SELECT s.id, s.from_user_id, s.to_user_id, s.amount, s.currency, s.occurred_at,
       g.name, fu.username, tu.username
FROM settlements s
JOIN groups g ON g.id = s.group_id
JOIN users fu ON fu.id = s.from_user_id
JOIN users tu ON tu.id = s.to_user_id
WHERE s.id = $1 AND s.counter_status = 'pending'
  AND $2 IN (s.from_user_id, s.to_user_id) AND s.created_by <> $2
FOR UPDATE OF s`, sid, me).Scan(&s.ID, &s.FromUserID, &s.ToUserID, &s.Amount, &s.Currency, &s.OccurredAt,
		&s.GroupName, &s.FromUsername, &s.ToUsername)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "bekleyen odeme bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "settlement counter lock", err)
		return
	}

	if !book {
		_, err = tx.Exec(ctx, "UPDATE settlements SET counter_status = 'dismissed' WHERE id = $1", sid)
	} else {
		var accCurrency string
		err = tx.QueryRow(ctx, "SELECT currency FROM accounts WHERE id = $1 AND user_id = $2",
			req.AccountID, me).Scan(&accCurrency)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
			return
		}
		if err != nil {
			httpx.ServerError(w, "settlement counter account", err)
			return
		}
		bookAmount, aerr := accountAmount(s.Amount, s.Currency, accCurrency, req.AccountAmount)
		if aerr != nil {
			httpx.Fail(w, "settlement counter", aerr)
			return
		}
		typ := "expense"
		if me == s.ToUserID {
			typ = "income"
		}
		desc := settlementDesc(s.GroupName, s.FromUsername, s.ToUsername, s.Amount, s.Currency, accCurrency)
		var txID int
		txID, err = ledger.Create(ctx, tx, me, ledger.Input{
			AccountID: req.AccountID, Type: typ, Amount: bookAmount, Description: &desc, OccurredAt: s.OccurredAt,
		})
		if err != nil {
			httpx.Fail(w, "settlement counter transaction", err)
			return
		}
		_, err = tx.Exec(ctx,
			"UPDATE settlements SET counter_status = 'booked', counter_transaction_id = $2 WHERE id = $1", sid, txID)
	}
	if err != nil {
		httpx.ServerError(w, "settlement counter update", err)
		return
	}
	// Bu ödemenin bildirimi okundu sayılır.
	if _, err := tx.Exec(ctx,
		"UPDATE notifications SET read_at = now() WHERE user_id = $1 AND kind = 'settlement' AND ref_id = $2 AND read_at IS NULL",
		me, sid); err != nil {
		httpx.ServerError(w, "settlement counter notification", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "settlement counter commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSettlement: ödemenin tarafları veya kaydı giren silebilir.
// İki tarafın hesabına işlenmiş ödeme işlemleri de silinir.
func (h *Handler) DeleteSettlement(w http.ResponseWriter, r *http.Request) {
	groupID, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	sid, ok := httpx.PathID(w, r, "sid")
	if !ok {
		return
	}
	ctx := r.Context()
	userID := auth.UserID(ctx)

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "settlement delete begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var txID, counterID *int
	err = tx.QueryRow(ctx, `
DELETE FROM settlements s
WHERE s.id = $1 AND s.group_id = $2
  AND $3 IN (s.from_user_id, s.to_user_id, s.created_by)
  AND EXISTS (SELECT 1 FROM group_members WHERE group_id = $2 AND user_id = $3)
RETURNING s.transaction_id, s.counter_transaction_id`,
		sid, groupID, userID,
	).Scan(&txID, &counterID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "hesaplasma bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "settlement delete", err)
		return
	}
	for _, id := range []*int{txID, counterID} {
		if id == nil {
			continue
		}
		if err := ledger.Remove(ctx, tx, *id); err != nil {
			httpx.ServerError(w, "settlement delete transaction", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "settlement delete commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
