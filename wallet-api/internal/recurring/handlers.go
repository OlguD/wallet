package recurring

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
)

type Handler struct {
	DB     *pgxpool.Pool
	Loc    *time.Location
	Runner *Runner
}

type Rule struct {
	ID          int               `json:"id"`
	AccountID   int               `json:"account_id"`
	AccountName string            `json:"account_name"`
	Currency    string            `json:"currency"`
	GroupID     *int              `json:"group_id"`
	UserID      int               `json:"user_id"`
	Username    string            `json:"username"`
	Type        string            `json:"type"`
	Amount      int64             `json:"amount"`
	Category    *string           `json:"category"`
	Description *string           `json:"description"`
	Split       *ledger.SplitSpec `json:"split"`
	Frequency   string            `json:"frequency"`
	Interval    int               `json:"interval"`
	NextRunOn   httpx.Date        `json:"next_run_on"`
	EndOn       *httpx.Date       `json:"end_on"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
}

const selectRule = `
SELECT rt.id, rt.account_id, a.name, a.currency, rt.group_id, a.user_id, u.username,
       rt.type, rt.amount, rt.category, rt.description, rt.split, rt.frequency, rt.interval_count,
       rt.next_run_on, rt.end_on, rt.active, rt.created_at
FROM recurring_transactions rt
JOIN accounts a ON a.id = rt.account_id
JOIN users u ON u.id = a.user_id
`

func scanRule(row pgx.CollectableRow) (Rule, error) {
	var k Rule
	var next time.Time
	var end *time.Time
	err := row.Scan(&k.ID, &k.AccountID, &k.AccountName, &k.Currency, &k.GroupID, &k.UserID, &k.Username,
		&k.Type, &k.Amount, &k.Category, &k.Description, &k.Split, &k.Frequency, &k.Interval,
		&next, &end, &k.Active, &k.CreatedAt)
	k.NextRunOn = httpx.Date{Time: next}
	if end != nil {
		k.EndOn = &httpx.Date{Time: *end}
	}
	return k, err
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, where string, args ...any) {
	rows, err := h.DB.Query(r.Context(), selectRule+where+" ORDER BY rt.active DESC, rt.next_run_on, rt.id", args...)
	if err != nil {
		httpx.ServerError(w, "recurring list", err)
		return
	}
	list, err := pgx.CollectRows(rows, scanRule)
	if err != nil {
		httpx.ServerError(w, "recurring scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// List kullanıcının kendi hesaplarındaki kurallar.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, "WHERE a.user_id = $1", auth.UserID(r.Context()))
}

// ListByGroup grubun ortak tekrarlayan giderleri (kira, aidat...); üyeler görebilir.
func (h *Handler) ListByGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var member bool
	err := h.DB.QueryRow(r.Context(),
		"SELECT EXISTS (SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)",
		id, auth.UserID(r.Context()),
	).Scan(&member)
	if err != nil {
		httpx.ServerError(w, "recurring group check", err)
		return
	}
	if !member {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}
	h.list(w, r, "WHERE rt.group_id = $1", id)
}

func (h *Handler) writeOne(w http.ResponseWriter, r *http.Request, id, status int) {
	rows, err := h.DB.Query(r.Context(), selectRule+"WHERE rt.id = $1 AND a.user_id = $2",
		id, auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "recurring get", err)
		return
	}
	k, err := pgx.CollectOneRow(rows, scanRule)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "tekrarlayan islem bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "recurring get scan", err)
		return
	}
	httpx.WriteJSON(w, status, k)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}

type createRequest struct {
	AccountID   int               `json:"account_id"`
	GroupID     *int              `json:"group_id"`
	Type        string            `json:"type"`
	Amount      int64             `json:"amount"`
	Category    *string           `json:"category"`
	Description *string           `json:"description"`
	Split       *ledger.SplitSpec `json:"split"`
	Frequency   string            `json:"frequency"`
	Interval    int               `json:"interval"`
	StartOn     *httpx.Date       `json:"start_on"`
	EndOn       *httpx.Date       `json:"end_on"`
}

// Create yeni kural ekler. start_on bugün veya geçmişteyse vadesi gelen
// işlemler hemen oluşturulur (geçmiş en fazla 1 yıl).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Type != "income" && req.Type != "expense" {
		httpx.WriteError(w, http.StatusBadRequest, "type 'income' veya 'expense' olmali")
		return
	}
	if err := ledger.ValidateAmount(req.Amount); err != nil {
		httpx.Fail(w, "recurring create", err)
		return
	}
	if err := ledger.ValidateCategory(req.Type, req.Category); err != nil {
		httpx.Fail(w, "recurring create", err)
		return
	}
	if req.Frequency != "weekly" && req.Frequency != "monthly" && req.Frequency != "yearly" {
		httpx.WriteError(w, http.StatusBadRequest, "frequency 'weekly', 'monthly' veya 'yearly' olmali")
		return
	}
	if req.Interval == 0 {
		req.Interval = 1
	}
	if req.Interval < 1 || req.Interval > 12 {
		httpx.WriteError(w, http.StatusBadRequest, "interval 1-12 arasinda olmali")
		return
	}
	desc, err := ledger.NormalizeDescription(req.Description)
	if err != nil {
		httpx.Fail(w, "recurring create", err)
		return
	}
	today := httpx.Today(h.Loc)
	start := today
	if req.StartOn != nil {
		start = req.StartOn.Time
	}
	if start.Before(today.AddDate(-1, 0, 0)) {
		httpx.WriteError(w, http.StatusBadRequest, "baslangic tarihi en fazla 1 yil once olabilir")
		return
	}
	var endOn *time.Time
	if req.EndOn != nil {
		if req.EndOn.Before(start) {
			httpx.WriteError(w, http.StatusBadRequest, "bitis tarihi baslangictan once olamaz")
			return
		}
		endOn = &req.EndOn.Time
	}

	ctx := r.Context()
	userID := auth.UserID(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "recurring begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var owner int
	err = tx.QueryRow(ctx, "SELECT user_id FROM accounts WHERE id = $1", req.AccountID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != userID) {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "recurring account", err)
		return
	}
	if req.GroupID != nil {
		if _, err := ledger.ResolveSplits(ctx, tx, *req.GroupID, userID, req.Amount, req.Split); err != nil {
			httpx.Fail(w, "recurring split", err)
			return
		}
	} else if req.Split != nil {
		httpx.WriteError(w, http.StatusBadRequest, "paylasim sadece grup islemlerinde kullanilabilir")
		return
	}

	var id int
	err = tx.QueryRow(ctx, `
INSERT INTO recurring_transactions
  (account_id, group_id, type, amount, description, split, frequency, interval_count, anchor_day, next_run_on, end_on, category)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
		req.AccountID, req.GroupID, req.Type, req.Amount, desc, req.Split,
		req.Frequency, req.Interval, start.Day(), start, endOn, req.Category,
	).Scan(&id)
	if err != nil {
		httpx.ServerError(w, "recurring insert", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "recurring commit", err)
		return
	}

	if !start.After(today) {
		if err := h.Runner.RunDue(ctx); err != nil {
			httpx.ServerError(w, "recurring run", err)
			return
		}
	}
	h.writeOne(w, r, id, http.StatusCreated)
}

type updateRequest struct {
	Amount      *int64                           `json:"amount"`
	Category    httpx.Optional[string]           `json:"category"`
	Description httpx.Optional[string]           `json:"description"`
	Split       httpx.Optional[ledger.SplitSpec] `json:"split"`
	Active      *bool                            `json:"active"`
	EndOn       httpx.Optional[httpx.Date]       `json:"end_on"`
}

// Update sadece kuralın sahibi tarafından yapılır ve ileride üretilecek
// işlemleri etkiler. Duraklatılmış kural yeniden açılırsa kaçırılan
// dönemler oluşturulmaz, bir sonraki vadeden devam edilir.
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
	userID := auth.UserID(ctx)

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "recurring update begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var k rule
	var active bool
	err = tx.QueryRow(ctx, `
SELECT rt.group_id, rt.type, rt.amount, rt.category, rt.description, rt.split, rt.frequency, rt.interval_count,
       rt.anchor_day, rt.next_run_on, rt.end_on, rt.active
FROM recurring_transactions rt
JOIN accounts a ON a.id = rt.account_id
WHERE rt.id = $1 AND a.user_id = $2
FOR UPDATE OF rt`, id, userID,
	).Scan(&k.groupID, &k.typ, &k.amount, &k.category, &k.description, &k.split, &k.frequency, &k.interval,
		&k.anchorDay, &k.next, &k.endOn, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "tekrarlayan islem bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "recurring update lock", err)
		return
	}

	if req.Amount != nil {
		if err := ledger.ValidateAmount(*req.Amount); err != nil {
			httpx.Fail(w, "recurring update", err)
			return
		}
		k.amount = *req.Amount
	}
	if req.Description.Set {
		if k.description, err = ledger.NormalizeDescription(req.Description.Value); err != nil {
			httpx.Fail(w, "recurring update", err)
			return
		}
	}
	if req.Category.Set {
		if err := ledger.ValidateCategory(k.typ, req.Category.Value); err != nil {
			httpx.Fail(w, "recurring update", err)
			return
		}
		k.category = req.Category.Value
	}
	if req.Split.Set {
		k.split = req.Split.Value
	}
	if req.EndOn.Set {
		k.endOn = nil
		if req.EndOn.Value != nil {
			k.endOn = &req.EndOn.Value.Time
		}
	}
	today := httpx.Today(h.Loc)
	if req.Active != nil {
		if *req.Active && !active {
			for k.next.Before(today) {
				k.next = Next(k.next, k.frequency, k.interval, k.anchorDay)
			}
		}
		active = *req.Active
	}
	if active && k.endOn != nil && k.next.After(*k.endOn) {
		httpx.WriteError(w, http.StatusBadRequest, "bitis tarihi bir sonraki vadeden once olamaz")
		return
	}

	if k.groupID != nil {
		if _, err := ledger.ResolveSplits(ctx, tx, *k.groupID, userID, k.amount, k.split); err != nil {
			httpx.Fail(w, "recurring update split", err)
			return
		}
	} else if k.split != nil {
		httpx.WriteError(w, http.StatusBadRequest, "paylasim sadece grup islemlerinde kullanilabilir")
		return
	}

	_, err = tx.Exec(ctx, `
UPDATE recurring_transactions
SET amount = $2, description = $3, split = $4, end_on = $5, active = $6, next_run_on = $7, category = $8, updated_at = now()
WHERE id = $1`,
		id, k.amount, k.description, k.split, k.endOn, active, k.next, k.category,
	)
	if err != nil {
		httpx.ServerError(w, "recurring update", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "recurring update commit", err)
		return
	}
	if active && !k.next.After(today) {
		if err := h.Runner.RunDue(ctx); err != nil {
			httpx.ServerError(w, "recurring run", err)
			return
		}
	}
	h.writeOne(w, r, id, http.StatusOK)
}

// Delete kuralı siler; daha önce üretilmiş işlemler yerinde kalır.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
DELETE FROM recurring_transactions rt
USING accounts a
WHERE rt.id = $1 AND a.id = rt.account_id AND a.user_id = $2`,
		id, auth.UserID(r.Context()),
	)
	if err != nil {
		httpx.ServerError(w, "recurring delete", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "tekrarlayan islem bulunamadi")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
