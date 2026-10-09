// Package goals birikim hedeflerini yönetir. Bir hedef bir hesaba bağlıdır
// (para orada durur, para birimi oradan gelir); ilerleme hedefe ayrılan
// katkıların toplamıdır. Katkılar hesap bakiyesini değiştirmez.
package goals

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
)

type Handler struct {
	DB  *pgxpool.Pool
	Loc *time.Location
}

type Goal struct {
	ID           int         `json:"id"`
	AccountID    int         `json:"account_id"`
	AccountName  string      `json:"account_name"`
	Currency     string      `json:"currency"`
	Name         string      `json:"name"`
	Icon         *string     `json:"icon"`
	TargetAmount int64       `json:"target_amount"`
	TargetDate   *httpx.Date `json:"target_date"`
	Current      int64       `json:"current"`
	Remaining    int64       `json:"remaining"`
	ProgressPct  int         `json:"progress_pct"`
	// MonthlyNeeded hedef tarihe kadar her ay biriktirilmesi gereken tutar.
	MonthlyNeeded *int64 `json:"monthly_needed"`
	// AccountBalance bağlı hesabın güncel bakiyesi (bilgi amaçlı).
	AccountBalance int64     `json:"account_balance"`
	CreatedAt      time.Time `json:"created_at"`
}

const selectGoal = `
SELECT g.id, g.account_id, a.name, a.currency, g.name, g.icon, g.target_amount, g.target_date,
       COALESCE((SELECT SUM(c.amount) FROM goal_contributions c WHERE c.goal_id = g.id), 0)::bigint,
       ` + ledger.BalanceSQL + `, g.created_at
FROM savings_goals g
JOIN accounts a ON a.id = g.account_id
`

func (h *Handler) scan(row pgx.CollectableRow) (Goal, error) {
	var g Goal
	var target *time.Time
	err := row.Scan(&g.ID, &g.AccountID, &g.AccountName, &g.Currency, &g.Name, &g.Icon, &g.TargetAmount,
		&target, &g.Current, &g.AccountBalance, &g.CreatedAt)
	if err != nil {
		return g, err
	}
	g.Remaining = max(0, g.TargetAmount-g.Current)
	g.ProgressPct = int(min(100, max(0, g.Current)*100/g.TargetAmount))
	if target != nil {
		g.TargetDate = &httpx.Date{Time: *target}
		if g.Remaining > 0 {
			today := httpx.Today(h.Loc)
			months := int64((target.Year()-today.Year())*12 + int(target.Month()) - int(today.Month()))
			months = max(1, months)
			need := (g.Remaining + months - 1) / months
			g.MonthlyNeeded = &need
		}
	}
	return g, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), selectGoal+"WHERE a.user_id = $1 ORDER BY g.id", auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "goals list", err)
		return
	}
	list, err := pgx.CollectRows(rows, h.scan)
	if err != nil {
		httpx.ServerError(w, "goals scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) writeOne(w http.ResponseWriter, r *http.Request, id, status int) {
	rows, err := h.DB.Query(r.Context(), selectGoal+"WHERE g.id = $1 AND a.user_id = $2", id, auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "goal get", err)
		return
	}
	g, err := pgx.CollectOneRow(rows, h.scan)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "hedef bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "goal get scan", err)
		return
	}
	httpx.WriteJSON(w, status, g)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}

func validName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	return name, n >= 1 && n <= 64
}

// Arayüzdeki hedef ikonları.
var icons = map[string]bool{"target": true, "car": true, "home": true, "gift": true, "bag": true, "heart": true, "plane": true, "phone": true, "school": true}

func validIcon(icon *string) bool {
	return icon == nil || icons[*icon]
}

type createRequest struct {
	AccountID    int         `json:"account_id"`
	Name         string      `json:"name"`
	Icon         *string     `json:"icon"`
	TargetAmount int64       `json:"target_amount"`
	TargetDate   *httpx.Date `json:"target_date"`
	// InitialAmount hedefe hemen ayrılacak başlangıç tutarı (opsiyonel).
	InitialAmount int64 `json:"initial_amount"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	name, ok := validName(req.Name)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "hedef adi 1-64 karakter olmali")
		return
	}
	if err := ledger.ValidateAmount(req.TargetAmount); err != nil {
		httpx.Fail(w, "goal create", err)
		return
	}
	if !validIcon(req.Icon) {
		httpx.WriteError(w, http.StatusBadRequest, "gecersiz ikon")
		return
	}
	if req.InitialAmount < 0 || req.InitialAmount > ledger.MaxAmount {
		httpx.WriteError(w, http.StatusBadRequest, "baslangic tutari gecersiz")
		return
	}
	var target *time.Time
	if req.TargetDate != nil {
		target = &req.TargetDate.Time
	}

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "goal create begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var id int
	err = tx.QueryRow(ctx, `
INSERT INTO savings_goals (account_id, name, icon, target_amount, target_date)
SELECT a.id, $3, $4, $5, $6 FROM accounts a WHERE a.id = $1 AND a.user_id = $2
RETURNING id`,
		req.AccountID, auth.UserID(ctx), name, req.Icon, req.TargetAmount, target,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "hesap bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "goal create", err)
		return
	}
	if req.InitialAmount > 0 {
		_, err = tx.Exec(ctx, "INSERT INTO goal_contributions (goal_id, amount) VALUES ($1, $2)", id, req.InitialAmount)
		if err != nil {
			httpx.ServerError(w, "goal initial contribution", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "goal create commit", err)
		return
	}
	h.writeOne(w, r, id, http.StatusCreated)
}

type updateRequest struct {
	Name         *string                    `json:"name"`
	Icon         httpx.Optional[string]     `json:"icon"`
	TargetAmount *int64                     `json:"target_amount"`
	TargetDate   httpx.Optional[httpx.Date] `json:"target_date"`
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
	var name *string
	if req.Name != nil {
		n, ok := validName(*req.Name)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "hedef adi 1-64 karakter olmali")
			return
		}
		name = &n
	}
	if req.TargetAmount != nil {
		if err := ledger.ValidateAmount(*req.TargetAmount); err != nil {
			httpx.Fail(w, "goal update", err)
			return
		}
	}
	if !validIcon(req.Icon.Value) {
		httpx.WriteError(w, http.StatusBadRequest, "gecersiz ikon")
		return
	}
	var target *time.Time
	if req.TargetDate.Value != nil {
		target = &req.TargetDate.Value.Time
	}

	tag, err := h.DB.Exec(r.Context(), `
UPDATE savings_goals g
SET name = COALESCE($3, g.name),
    target_amount = COALESCE($4, g.target_amount),
    target_date = CASE WHEN $5 THEN $6 ELSE g.target_date END,
    icon = CASE WHEN $7 THEN $8 ELSE g.icon END,
    updated_at = now()
FROM accounts a
WHERE g.id = $1 AND a.id = g.account_id AND a.user_id = $2`,
		id, auth.UserID(r.Context()), name, req.TargetAmount, req.TargetDate.Set, target, req.Icon.Set, req.Icon.Value,
	)
	if err != nil {
		httpx.ServerError(w, "goal update", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "hedef bulunamadi")
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
DELETE FROM savings_goals g USING accounts a
WHERE g.id = $1 AND a.id = g.account_id AND a.user_id = $2`,
		id, auth.UserID(r.Context()),
	)
	if err != nil {
		httpx.ServerError(w, "goal delete", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "hedef bulunamadi")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
