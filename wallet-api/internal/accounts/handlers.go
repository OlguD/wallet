package accounts

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
)

type Handler struct {
	DB *pgxpool.Pool
}

var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

var kinds = map[string]bool{"bank": true, "cash": true, "card": true, "savings": true}

func validKind(k *string) (string, bool) {
	if k == nil || *k == "" {
		return "bank", true
	}
	return *k, kinds[*k]
}

type Account struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	Kind      string    `json:"kind"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	// Sadece kartlarda: limit (kuruş) ve ayın kaçında son ödeme. Borç = -bakiye.
	CreditLimit *int64 `json:"credit_limit"`
	DueDay      *int   `json:"due_day"`
}

const selectAccount = `
SELECT a.id, a.name, a.currency, a.kind, a.created_at, a.credit_limit, a.due_day, ` + ledger.BalanceSQL + `
FROM accounts a`

func scanAccount(row pgx.Row) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.Name, &a.Currency, &a.Kind, &a.CreatedAt, &a.CreditLimit, &a.DueDay, &a.Balance)
	return a, err
}

// cardFields kart limitini ve son ödeme gününü doğrular. Kart olmayan hesapta
// ikisi de temizlenir (nil).
func cardFields(kind string, limit *int64, dueDay *int) (*int64, *int, error) {
	if kind != "card" {
		return nil, nil, nil
	}
	if limit != nil && *limit == 0 {
		limit = nil
	}
	if limit != nil && (*limit < 0 || *limit > ledger.MaxAmount) {
		return nil, nil, httpx.BadRequest("kart limiti pozitif olmali")
	}
	if dueDay != nil && *dueDay == 0 {
		dueDay = nil
	}
	if dueDay != nil && (*dueDay < 1 || *dueDay > 31) {
		return nil, nil, httpx.BadRequest("son odeme gunu 1-31 arasinda olmali")
	}
	return limit, dueDay, nil
}

func validName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	return name, n >= 1 && n <= 64
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(),
		selectAccount+" WHERE a.user_id = $1 ORDER BY a.id",
		auth.UserID(r.Context()),
	)
	if err != nil {
		httpx.ServerError(w, "accounts list", err)
		return
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Account, error) {
		return scanAccount(row)
	})
	if err != nil {
		httpx.ServerError(w, "accounts list scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	a, err := scanAccount(h.DB.QueryRow(r.Context(),
		selectAccount+" WHERE a.id = $1 AND a.user_id = $2",
		id, auth.UserID(r.Context()),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "account get", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

type createRequest struct {
	Name        string  `json:"name"`
	Currency    string  `json:"currency"`
	Kind        *string `json:"kind"`
	CreditLimit *int64  `json:"credit_limit"`
	DueDay      *int    `json:"due_day"`
	// OpeningDebt kart eklenirken mevcut borç (kuruş). "Açılış borcu" işlemi
	// olarak yazılır; gelir/gider toplamlarına sayılmaz (kategori opening).
	OpeningDebt int64 `json:"opening_debt"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	name, ok := validName(req.Name)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "hesap adi 1-64 karakter olmali")
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "TRY"
	}
	if !currencyRe.MatchString(currency) {
		httpx.WriteError(w, http.StatusBadRequest, "para birimi 3 harfli ISO 4217 kodu olmali (orn. TRY)")
		return
	}

	kind, ok := validKind(req.Kind)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "hesap turu bank, cash, card veya savings olmali")
		return
	}

	limit, dueDay, err := cardFields(kind, req.CreditLimit, req.DueDay)
	if err != nil {
		httpx.Fail(w, "account create", err)
		return
	}
	if req.OpeningDebt != 0 {
		if kind != "card" {
			httpx.WriteError(w, http.StatusBadRequest, "acilis borcu sadece kartlarda kullanilabilir")
			return
		}
		if err := ledger.ValidateAmount(req.OpeningDebt); err != nil {
			httpx.Fail(w, "account create", err)
			return
		}
	}

	ctx := r.Context()
	userID := auth.UserID(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "account create begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var id int
	err = tx.QueryRow(ctx,
		`INSERT INTO accounts (user_id, name, currency, kind, credit_limit, due_day)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		userID, name, currency, kind, limit, dueDay,
	).Scan(&id)
	if isUniqueViolation(err) {
		httpx.WriteError(w, http.StatusConflict, "bu isimde bir hesabiniz zaten var")
		return
	}
	if err != nil {
		httpx.ServerError(w, "account create", err)
		return
	}
	if req.OpeningDebt > 0 {
		cat, desc := ledger.CategoryOpening, "Açılış borcu"
		_, err = ledger.Create(ctx, tx, userID, ledger.Input{
			AccountID: id, Type: "expense", Amount: req.OpeningDebt,
			Category: &cat, Description: &desc, OccurredAt: time.Now(),
		})
		if err != nil {
			httpx.Fail(w, "account opening debt", err)
			return
		}
	}
	a, err := scanAccount(tx.QueryRow(ctx, selectAccount+" WHERE a.id = $1", id))
	if err != nil {
		httpx.ServerError(w, "account create get", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "account create commit", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, a)
}

type updateRequest struct {
	Name        *string `json:"name"`
	Kind        *string `json:"kind"`
	CreditLimit *int64  `json:"credit_limit"`
	DueDay      *int    `json:"due_day"`
}

// Update hesap adını, türünü ve kart bilgilerini değiştirir; para birimi
// değiştirilemez. Kart bilgileri gönderilmezse korunur, 0 gönderilirse
// temizlenir; tür kart olmaktan çıkarsa ikisi de silinir.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req updateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	var name *string
	if req.Name != nil {
		n, ok := validName(*req.Name)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "hesap adi 1-64 karakter olmali")
			return
		}
		name = &n
	}
	if req.Kind != nil {
		if _, ok := validKind(req.Kind); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "hesap turu bank, cash, card veya savings olmali")
			return
		}
	}
	// Kart alanları türden bağımsız doğrulanır; tür kart değilse SQL temizler.
	setLimit, setDue := req.CreditLimit != nil, req.DueDay != nil
	limit, dueDay, err := cardFields("card", req.CreditLimit, req.DueDay)
	if err != nil {
		httpx.Fail(w, "account update", err)
		return
	}

	tag, err := h.DB.Exec(r.Context(), `
UPDATE accounts SET name = COALESCE($1, name), kind = COALESCE($2, kind),
  credit_limit = CASE WHEN COALESCE($2, kind) <> 'card' THEN NULL WHEN $5 THEN $6 ELSE credit_limit END,
  due_day = CASE WHEN COALESCE($2, kind) <> 'card' THEN NULL WHEN $7 THEN $8 ELSE due_day END,
  updated_at = now()
WHERE id = $3 AND user_id = $4`,
		name, req.Kind, id, auth.UserID(r.Context()), setLimit, limit, setDue, dueDay,
	)
	if isUniqueViolation(err) {
		httpx.WriteError(w, http.StatusConflict, "bu isimde bir hesabiniz zaten var")
		return
	}
	if err != nil {
		httpx.ServerError(w, "account update", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}
	h.Get(w, r)
}

// Delete sadece hiç işlemi, tekrar kuralı ve hedefi olmayan hesabı siler;
// geçmişi olan hesap silinemez (veri doğruluğu).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	var owned, used bool
	err := h.DB.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2),
       EXISTS (SELECT 1 FROM transactions WHERE account_id = $1)
    OR EXISTS (SELECT 1 FROM recurring_transactions WHERE account_id = $1)
    OR EXISTS (SELECT 1 FROM savings_goals WHERE account_id = $1)`,
		id, auth.UserID(ctx),
	).Scan(&owned, &used)
	if err != nil {
		httpx.ServerError(w, "account delete check", err)
		return
	}
	if !owned {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}
	if used {
		httpx.WriteError(w, http.StatusConflict, "islemi olan hesap silinemez")
		return
	}
	// Kontrol ile silme arasında işlem eklenirse FK hatası verir; o da 409'a çevrilir.
	_, err = h.DB.Exec(ctx, "DELETE FROM accounts WHERE id = $1 AND user_id = $2", id, auth.UserID(ctx))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		httpx.WriteError(w, http.StatusConflict, "islemi olan hesap silinemez")
		return
	}
	if err != nil {
		httpx.ServerError(w, "account delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
