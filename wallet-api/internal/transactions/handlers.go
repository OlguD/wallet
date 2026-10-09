package transactions

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
	"wallet-api/internal/receipts"
)

type Handler struct {
	DB  *pgxpool.Pool
	Loc *time.Location
}

type SplitView struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Amount   int64  `json:"amount"`
}

type Transaction struct {
	ID           int         `json:"id"`
	AccountID    int         `json:"account_id"`
	GroupID      *int        `json:"group_id"`
	Type         string      `json:"type"`
	Amount       int64       `json:"amount"`
	Currency     string      `json:"currency"`
	Category     *string     `json:"category"`
	AccountName  string      `json:"account_name"`
	Description  *string     `json:"description"`
	OccurredAt   time.Time   `json:"occurred_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	RecurringID  *int        `json:"recurring_id"`
	BalanceAfter *int64      `json:"balance_after,omitempty"`
	UserID       int         `json:"user_id"`
	Username     string      `json:"username"`
	Splits       []SplitView `json:"splits,omitempty"`
	// Karşı taraf (dekonttan ya da elle): giderde alıcı, gelirde gönderen.
	CounterpartyName *string `json:"counterparty_name"`
	CounterpartyIBAN *string `json:"counterparty_iban"`
	CounterpartyBank *string `json:"counterparty_bank"`
	ReceiptID        *int    `json:"receipt_id"`
}

// $1 her zaman görüntüleyen kullanıcıdır; başkasının hesabındaki
// bakiye (grup işlemlerinde) gösterilmez.
const selectTx = `
SELECT t.id, t.account_id, t.group_id, t.type, t.amount, a.currency, t.category,
       CASE WHEN a.user_id = $1 THEN a.name ELSE '' END, t.description,
       t.occurred_at, t.created_at, t.updated_at, t.recurring_id,
       CASE WHEN a.user_id = $1 THEN b.amount END, a.user_id, u.username,
       t.counterparty_name, t.counterparty_iban, t.counterparty_bank, rc.id
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN users u ON u.id = a.user_id
LEFT JOIN balances b ON b.transaction_id = t.id
LEFT JOIN receipts rc ON rc.transaction_id = t.id
`

func scanTx(row pgx.CollectableRow) (Transaction, error) {
	var t Transaction
	err := row.Scan(&t.ID, &t.AccountID, &t.GroupID, &t.Type, &t.Amount, &t.Currency, &t.Category,
		&t.AccountName, &t.Description,
		&t.OccurredAt, &t.CreatedAt, &t.UpdatedAt, &t.RecurringID,
		&t.BalanceAfter, &t.UserID, &t.Username,
		&t.CounterpartyName, &t.CounterpartyIBAN, &t.CounterpartyBank, &t.ReceiptID)
	return t, err
}

// query selectTx'e verilen WHERE/ORDER parçasını ekler (parametreler $2'den başlar)
// ve grup işlemlerinin paylarını doldurur.
func (h *Handler) query(ctx context.Context, viewerID int, rest string, args ...any) ([]Transaction, error) {
	rows, err := h.DB.Query(ctx, selectTx+rest, append([]any{viewerID}, args...)...)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, scanTx)
	if err != nil {
		return nil, err
	}

	var ids []int
	idx := map[int]int{}
	for i, t := range list {
		if t.GroupID != nil {
			ids = append(ids, t.ID)
			idx[t.ID] = i
		}
	}
	if len(ids) == 0 {
		return list, nil
	}
	rows, err = h.DB.Query(ctx, `
SELECT s.transaction_id, s.user_id, u.username, s.amount
FROM transaction_splits s
JOIN users u ON u.id = s.user_id
WHERE s.transaction_id = ANY($1)
ORDER BY s.transaction_id, s.user_id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var txID int
		var s SplitView
		if err := rows.Scan(&txID, &s.UserID, &s.Username, &s.Amount); err != nil {
			return nil, err
		}
		list[idx[txID]].Splits = append(list[idx[txID]].Splits, s)
	}
	return list, rows.Err()
}

// visibleTo: hesabın sahibi veya işlemin grubunun güncel üyesi.
const visibleTo = `
WHERE t.id = $2
  AND (a.user_id = $1 OR EXISTS (
        SELECT 1 FROM group_members gm WHERE gm.group_id = t.group_id AND gm.user_id = $1))`

func (h *Handler) writeOne(w http.ResponseWriter, r *http.Request, id, status int) {
	list, err := h.query(r.Context(), auth.UserID(r.Context()), visibleTo, id)
	if err != nil {
		httpx.ServerError(w, "transaction get", err)
		return
	}
	if len(list) == 0 {
		httpx.WriteError(w, http.StatusNotFound, "islem bulunamadi")
		return
	}
	httpx.WriteJSON(w, status, list[0])
}

type createRequest struct {
	AccountID   int               `json:"account_id"`
	GroupID     *int              `json:"group_id"`
	Type        string            `json:"type"`
	Amount      int64             `json:"amount"`
	Category    *string           `json:"category"`
	Description *string           `json:"description"`
	OccurredAt  *time.Time        `json:"occurred_at"`
	Split       *ledger.SplitSpec `json:"split"`
	Counterparty
	// ReceiptID işlemi bir dekonta bağlar (dekont "kullanıldı" olur).
	ReceiptID *int `json:"receipt_id"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	in := ledger.Input{
		AccountID:   req.AccountID,
		GroupID:     req.GroupID,
		Type:        req.Type,
		Amount:      req.Amount,
		Category:    req.Category,
		Description: req.Description,
		OccurredAt:  time.Now(),
		Split:       req.Split,
	}
	if req.OccurredAt != nil {
		in.OccurredAt = *req.OccurredAt
	}

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "transaction begin", err)
		return
	}
	defer tx.Rollback(ctx)

	userID := auth.UserID(ctx)
	id, err := ledger.Create(ctx, tx, userID, in)
	if err != nil {
		httpx.Fail(w, "transaction create", err)
		return
	}
	cp := counterpartyPatch{
		Name: httpx.Optional[string]{Set: req.Name != nil, Value: req.Name},
		IBAN: httpx.Optional[string]{Set: req.IBAN != nil, Value: req.IBAN},
		Bank: httpx.Optional[string]{Set: req.Bank != nil, Value: req.Bank},
	}
	if err := setCounterparty(ctx, tx, userID, id, cp); err != nil {
		httpx.Fail(w, "transaction counterparty", err)
		return
	}
	if req.ReceiptID != nil {
		if err := receipts.Attach(ctx, tx, userID, *req.ReceiptID, id); err != nil {
			httpx.Fail(w, "transaction receipt", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "transaction commit", err)
		return
	}
	h.writeOne(w, r, id, http.StatusCreated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}

type updateRequest struct {
	Type        *string                `json:"type"`
	Amount      *int64                 `json:"amount"`
	Category    httpx.Optional[string] `json:"category"`
	Description httpx.Optional[string] `json:"description"`
	OccurredAt  *time.Time             `json:"occurred_at"`
	GroupID     httpx.Optional[int]    `json:"group_id"`
	Split       *ledger.SplitSpec      `json:"split"`
	Name        httpx.Optional[string] `json:"counterparty_name"`
	IBAN        httpx.Optional[string] `json:"counterparty_iban"`
	Bank        httpx.Optional[string] `json:"counterparty_bank"`
	ReceiptID   *int                   `json:"receipt_id"`
}

// Update: sadece gönderilen alanlar değişir. group_id: null işlemi kişisele çevirir.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req updateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "transaction update begin", err)
		return
	}
	defer tx.Rollback(ctx)

	err = ledger.Update(ctx, tx, auth.UserID(ctx), id, ledger.Patch{
		Type:        req.Type,
		Amount:      req.Amount,
		Category:    req.Category,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
		GroupID:     req.GroupID,
		Split:       req.Split,
	})
	if err != nil {
		httpx.Fail(w, "transaction update", err)
		return
	}
	userID := auth.UserID(ctx)
	if err := setCounterparty(ctx, tx, userID, id, counterpartyPatch{req.Name, req.IBAN, req.Bank}); err != nil {
		httpx.Fail(w, "transaction counterparty", err)
		return
	}
	if req.ReceiptID != nil {
		if err := receipts.Attach(ctx, tx, userID, *req.ReceiptID, id); err != nil {
			httpx.Fail(w, "transaction receipt", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "transaction update commit", err)
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "transaction delete begin", err)
		return
	}
	defer tx.Rollback(ctx)

	// Bağlı dekont silinmez; gelen kutusuna geri döner. (İşlem silinmeden önce:
	// silme yetkisi yoksa aşağıdaki Delete hata verir ve bu da geri alınır.)
	if _, err := tx.Exec(ctx, "UPDATE receipts SET status = 'pending', transaction_id = NULL, updated_at = now() WHERE transaction_id = $1", id); err != nil {
		httpx.ServerError(w, "transaction delete receipt", err)
		return
	}
	if err := ledger.Delete(ctx, tx, auth.UserID(ctx), id); err != nil {
		httpx.Fail(w, "transaction delete", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "transaction delete commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListByAccount kullanıcının bir hesabındaki işlemleri (kişisel + grup) yeniden eskiye listeler.
func (h *Handler) ListByAccount(w http.ResponseWriter, r *http.Request) {
	accountID, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	userID := auth.UserID(ctx)

	var exists bool
	err := h.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)",
		accountID, userID,
	).Scan(&exists)
	if err != nil {
		httpx.ServerError(w, "transactions list check", err)
		return
	}
	if !exists {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}

	limit, offset := httpx.Pagination(r)
	list, err := h.query(ctx, userID, `
WHERE t.account_id = $2
ORDER BY t.occurred_at DESC, t.id DESC
LIMIT $3 OFFSET $4`, accountID, limit, offset)
	if err != nil {
		httpx.ServerError(w, "transactions list", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// List kullanıcının tüm hesaplarındaki işlemleri listeler.
// Filtreler: ?from=&to= (YYYY-AA-GG, to hariç), ?account_id=, ?category=, ?group_id=, ?counterparty=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	var from, to *time.Time
	if q.Get("from") != "" || q.Get("to") != "" {
		f, t, err := httpx.DateRange(r, h.Loc)
		if err != nil {
			httpx.Fail(w, "transactions list range", err)
			return
		}
		from, to = &f, &t
	}
	optInt := func(name string) (*int, bool) {
		v := q.Get(name)
		if v == "" {
			return nil, true
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, name+" sayi olmali")
			return nil, false
		}
		return &n, true
	}
	accountID, ok := optInt("account_id")
	if !ok {
		return
	}
	groupID, ok := optInt("group_id")
	if !ok {
		return
	}
	var category, counterparty *string
	if v := q.Get("category"); v != "" {
		category = &v
	}
	// counterparty: rapordaki "key" (IBAN veya "name:<küçük harf ad>").
	if v := q.Get("counterparty"); v != "" {
		counterparty = &v
	}

	limit, offset := httpx.Pagination(r)
	list, err := h.query(ctx, auth.UserID(ctx), `
WHERE a.user_id = $1
  AND ($2::timestamptz IS NULL OR t.occurred_at >= $2)
  AND ($3::timestamptz IS NULL OR t.occurred_at < $3)
  AND ($4::int IS NULL OR t.account_id = $4)
  AND ($5::int IS NULL OR t.group_id = $5)
  AND ($6::text IS NULL OR t.category = $6)
  AND ($9::text IS NULL OR COALESCE(t.counterparty_iban, 'name:' || lower(t.counterparty_name)) = $9)
ORDER BY t.occurred_at DESC, t.id DESC
LIMIT $7 OFFSET $8`, from, to, accountID, groupID, category, limit, offset, counterparty)
	if err != nil {
		httpx.ServerError(w, "transactions list all", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// ListByGroup grubun işlemlerini listeler; sadece güncel üyeler görebilir.
func (h *Handler) ListByGroup(w http.ResponseWriter, r *http.Request) {
	groupID, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	userID := auth.UserID(ctx)

	var member bool
	err := h.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)",
		groupID, userID,
	).Scan(&member)
	if err != nil {
		httpx.ServerError(w, "group transactions check", err)
		return
	}
	if !member {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}

	limit, offset := httpx.Pagination(r)
	list, err := h.query(ctx, userID, `
WHERE t.group_id = $2
ORDER BY t.occurred_at DESC, t.id DESC
LIMIT $3 OFFSET $4`, groupID, limit, offset)
	if err != nil {
		httpx.ServerError(w, "group transactions list", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

type accountSummary struct {
	AccountID int   `json:"account_id"`
	Income    int64 `json:"income"`
	Expense   int64 `json:"expense"`
}

type categorySummary struct {
	Currency string `json:"currency"`
	Type     string `json:"type"`
	Category string `json:"category"`
	Amount   int64  `json:"amount"`
}

type summaryRow struct {
	Currency string `json:"currency"`
	Income   int64  `json:"income"`
	Expense  int64  `json:"expense"`
	Net      int64  `json:"net"`
}

// Summary kullanıcının tüm hesaplarındaki gelir/gider toplamlarını para birimine göre döner.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	from, to, err := httpx.DateRange(r, h.Loc)
	if err != nil {
		httpx.Fail(w, "summary range", err)
		return
	}
	rows, err := h.DB.Query(r.Context(), `
SELECT a.currency,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0)::bigint,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)::bigint
FROM transactions t
JOIN accounts a ON a.id = t.account_id
WHERE a.user_id = $1 AND t.occurred_at >= $2 AND t.occurred_at < $3
GROUP BY a.currency
ORDER BY a.currency`,
		auth.UserID(r.Context()), from, to,
	)
	if err != nil {
		httpx.ServerError(w, "summary", err)
		return
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (summaryRow, error) {
		var s summaryRow
		err := row.Scan(&s.Currency, &s.Income, &s.Expense)
		s.Net = s.Income - s.Expense
		return s, err
	})
	if err != nil {
		httpx.ServerError(w, "summary scan", err)
		return
	}
	rows, err = h.DB.Query(r.Context(), `
SELECT a.id,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0)::bigint,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)::bigint
FROM accounts a
LEFT JOIN transactions t ON t.account_id = a.id AND t.occurred_at >= $2 AND t.occurred_at < $3
WHERE a.user_id = $1
GROUP BY a.id
ORDER BY a.id`,
		auth.UserID(r.Context()), from, to,
	)
	if err != nil {
		httpx.ServerError(w, "summary accounts", err)
		return
	}
	accounts, err := pgx.CollectRows(rows, pgx.RowToStructByPos[accountSummary])
	if err != nil {
		httpx.ServerError(w, "summary accounts scan", err)
		return
	}

	rows, err = h.DB.Query(r.Context(), `
SELECT a.currency, t.type::text, COALESCE(t.category, 'other'), SUM(t.amount)::bigint
FROM transactions t
JOIN accounts a ON a.id = t.account_id
WHERE a.user_id = $1 AND t.occurred_at >= $2 AND t.occurred_at < $3
GROUP BY 1, 2, 3
ORDER BY 1, 2, 4 DESC`,
		auth.UserID(r.Context()), from, to,
	)
	if err != nil {
		httpx.ServerError(w, "summary categories", err)
		return
	}
	cats, err := pgx.CollectRows(rows, pgx.RowToStructByPos[categorySummary])
	if err != nil {
		httpx.ServerError(w, "summary categories scan", err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"from":       from.Format(time.DateOnly),
		"to":         to.Format(time.DateOnly),
		"totals":     list,
		"accounts":   accounts,
		"categories": cats,
	})
}
