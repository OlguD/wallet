// Package budgets aylık kategori bütçelerini yönetir. Harcanan tutar, o ay
// kullanıcının hesaplarından çıkan (transfer olmayan) giderlerin toplamıdır.
package budgets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
	"wallet-api/internal/notify"
)

type Handler struct {
	DB     *pgxpool.Pool
	Loc    *time.Location
	Notify *notify.Notifier
}

type Budget struct {
	ID        int    `json:"id"`
	Category  string `json:"category"`
	Currency  string `json:"currency"`
	Amount    int64  `json:"amount"`
	Spent     int64  `json:"spent"`
	Remaining int64  `json:"remaining"`
	Pct       int    `json:"pct"`
}

// monthRange ?month=YYYY-MM (varsayılan bu ay).
func (h *Handler) monthRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().In(h.Loc)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, h.Loc)
	if v := r.URL.Query().Get("month"); v != "" {
		t, err := time.ParseInLocation("2006-01", v, h.Loc)
		if err != nil {
			return from, from, httpx.BadRequest("month YYYY-AA formatinda olmali")
		}
		from = t
	}
	return from, from.AddDate(0, 1, 0), nil
}

const spentSQL = `COALESCE((
  SELECT SUM(t.amount) FROM transactions t JOIN accounts a ON a.id = t.account_id
  WHERE a.user_id = b.user_id AND a.currency = b.currency AND t.type = 'expense'
    AND t.transfer_peer_id IS NULL AND COALESCE(t.category, 'other') = b.category
    AND t.occurred_at >= $2 AND t.occurred_at < $3), 0)::bigint`

func scan(row pgx.CollectableRow) (Budget, error) {
	var b Budget
	err := row.Scan(&b.ID, &b.Category, &b.Currency, &b.Amount, &b.Spent)
	b.Remaining = max(0, b.Amount-b.Spent)
	b.Pct = int(b.Spent * 100 / b.Amount)
	return b, err
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	from, to, err := h.monthRange(r)
	if err != nil {
		httpx.Fail(w, "budgets range", err)
		return
	}
	rows, err := h.DB.Query(r.Context(), `
SELECT b.id, b.category, b.currency, b.amount, `+spentSQL+`
FROM budgets b WHERE b.user_id = $1 ORDER BY b.category, b.currency`, auth.UserID(r.Context()), from, to)
	if err != nil {
		httpx.ServerError(w, "budgets list", err)
		return
	}
	list, err := pgx.CollectRows(rows, scan)
	if err != nil {
		httpx.ServerError(w, "budgets scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

type createRequest struct {
	Category string `json:"category"`
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Currency == "" {
		req.Currency = "TRY"
	}
	if err := ledger.ValidateCategory("expense", &req.Category); err != nil || req.Category == "" {
		httpx.WriteError(w, http.StatusBadRequest, "gecersiz gider kategorisi")
		return
	}
	if len(req.Currency) != 3 {
		httpx.WriteError(w, http.StatusBadRequest, "para birimi gecersiz")
		return
	}
	if err := ledger.ValidateAmount(req.Amount); err != nil {
		httpx.Fail(w, "budget create", err)
		return
	}
	var id int
	err := h.DB.QueryRow(r.Context(),
		"INSERT INTO budgets (user_id, category, currency, amount) VALUES ($1, $2, $3, $4) RETURNING id",
		auth.UserID(r.Context()), req.Category, req.Currency, req.Amount).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		httpx.WriteError(w, http.StatusConflict, "bu kategori icin zaten butce var")
		return
	}
	if err != nil {
		httpx.ServerError(w, "budget create", err)
		return
	}
	h.writeOne(w, r, id, http.StatusCreated)
}

func (h *Handler) writeOne(w http.ResponseWriter, r *http.Request, id, status int) {
	from, to, _ := h.monthRange(r)
	rows, err := h.DB.Query(r.Context(), `
SELECT b.id, b.category, b.currency, b.amount, `+spentSQL+`
FROM budgets b WHERE b.user_id = $1 AND b.id = $4`, auth.UserID(r.Context()), from, to, id)
	if err != nil {
		httpx.ServerError(w, "budget get", err)
		return
	}
	b, err := pgx.CollectOneRow(rows, scan)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "butce bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "budget scan", err)
		return
	}
	httpx.WriteJSON(w, status, b)
}

type updateRequest struct {
	Amount int64 `json:"amount"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req updateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := ledger.ValidateAmount(req.Amount); err != nil {
		httpx.Fail(w, "budget update", err)
		return
	}
	tag, err := h.DB.Exec(r.Context(), "UPDATE budgets SET amount = $3, updated_at = now() WHERE id = $1 AND user_id = $2",
		id, auth.UserID(r.Context()), req.Amount)
	if err != nil {
		httpx.ServerError(w, "budget update", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "butce bulunamadi")
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(), "DELETE FROM budgets WHERE id = $1 AND user_id = $2", id, auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "budget delete", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "butce bulunamadi")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Check bir gider eklendikten sonra çağrılır: işlemin ayındaki bütçe %80'i
// ya da tamamı geçildiyse kullanıcıya (ayda bir kez) bildirim gönderir.
func Check(ctx context.Context, db *pgxpool.Pool, n *notify.Notifier, loc *time.Location, txID int) {
	var userID, budgetID int
	var category, currency string
	var amount int64
	var at time.Time
	err := db.QueryRow(ctx, `
SELECT a.user_id, b.id, b.category, b.currency, b.amount, t.occurred_at
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN budgets b ON b.user_id = a.user_id AND b.currency = a.currency AND b.category = COALESCE(t.category, 'other')
WHERE t.id = $1 AND t.type = 'expense' AND t.transfer_peer_id IS NULL`, txID).
		Scan(&userID, &budgetID, &category, &currency, &amount, &at)
	if err != nil {
		return // bütçe yok ya da gider değil
	}
	at = at.In(loc)
	from := time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, loc)
	var spent int64
	err = db.QueryRow(ctx, `SELECT `+spentSQL+` FROM budgets b WHERE b.id = $1`, budgetID, from, from.AddDate(0, 1, 0)).Scan(&spent)
	if err != nil {
		return
	}
	month := from.Format("2006-01")
	label := CategoryLabel(category)
	switch {
	case spent > amount:
		n.Send(ctx, notify.Notice{
			UserID: userID, Kind: "budget:" + month, RefID: &budgetID, URL: "/budgets",
			Title: label + " bütçesi aşıldı",
			Body:  fmt.Sprintf("Bu ay %s harcadın; bütçen %s.", money(spent, currency), money(amount, currency)),
		})
	case spent*100 >= amount*80:
		n.Send(ctx, notify.Notice{
			UserID: userID, Kind: "budget80:" + month, RefID: &budgetID, URL: "/budgets",
			Title: label + " bütçesinin %80'i doldu",
			Body:  fmt.Sprintf("Bu ay %s harcadın; kalan %s.", money(spent, currency), money(amount-spent, currency)),
		})
	}
}

var labels = map[string]string{
	"groceries": "Market", "bills": "Fatura", "transport": "Ulaşım", "food": "Yemek", "rent": "Kira",
	"subscription": "Abonelik", "gift": "Hediye", "health": "Sağlık", "shopping": "Alışveriş", "other": "Diğer",
	"salary": "Maaş", "extra": "Ek gelir",
}

// CategoryLabel kategori anahtarının Türkçe adı (bildirim ve CSV için).
func CategoryLabel(c string) string {
	if l, ok := labels[c]; ok {
		return l
	}
	return c
}

// money 123456, TRY → "1.234,56 ₺"
func money(k int64, cur string) string {
	return ledger.FormatMoney(k) + " " + map[string]string{"TRY": "₺", "USD": "$", "EUR": "€", "GBP": "£"}[cur]
}
