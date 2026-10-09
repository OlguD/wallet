package goals

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
)

type Contribution struct {
	ID         int       `json:"id"`
	Amount     int64     `json:"amount"`
	Note       *string   `json:"note"`
	OccurredAt time.Time `json:"occurred_at"`
}

// ownGoal hedefin kullanıcıya ait olup olmadığını kontrol eder; değilse 404 yazar.
func (h *Handler) ownGoal(ctx context.Context, q pgx.Tx, w http.ResponseWriter, id int, lock bool) bool {
	sql := `SELECT g.id FROM savings_goals g JOIN accounts a ON a.id = g.account_id
WHERE g.id = $1 AND a.user_id = $2`
	if lock {
		sql += " FOR UPDATE OF g"
	}
	var got int
	err := q.QueryRow(ctx, sql, id, auth.UserID(ctx)).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "hedef bulunamadi")
		return false
	}
	if err != nil {
		httpx.ServerError(w, "goal owner", err)
		return false
	}
	return true
}

// ListContributions hedefin katkı geçmişi, yeniden eskiye.
func (h *Handler) ListContributions(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "contributions begin", err)
		return
	}
	defer tx.Rollback(ctx)
	if !h.ownGoal(ctx, tx, w, id, false) {
		return
	}
	rows, err := tx.Query(ctx, `
SELECT id, amount, note, occurred_at FROM goal_contributions
WHERE goal_id = $1 ORDER BY occurred_at DESC, id DESC LIMIT 200`, id)
	if err != nil {
		httpx.ServerError(w, "contributions list", err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Contribution])
	if err != nil {
		httpx.ServerError(w, "contributions scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

type contributionRequest struct {
	// Amount pozitif: hedefe ekle, negatif: hedeften çek.
	Amount int64   `json:"amount"`
	Note   *string `json:"note"`
}

// AddContribution hedefe para ayırır veya çeker; güncel hedefi döner.
// Birikmiş tutardan fazlası çekilemez.
func (h *Handler) AddContribution(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req contributionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Amount == 0 || req.Amount > ledger.MaxAmount || req.Amount < -ledger.MaxAmount {
		httpx.WriteError(w, http.StatusBadRequest, "tutar sifirdan farkli olmali (kurus cinsinden)")
		return
	}
	var note *string
	if req.Note != nil {
		n := strings.TrimSpace(*req.Note)
		if utf8.RuneCountInString(n) > 120 {
			httpx.WriteError(w, http.StatusBadRequest, "not en fazla 120 karakter olabilir")
			return
		}
		if n != "" {
			note = &n
		}
	}

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "contribution begin", err)
		return
	}
	defer tx.Rollback(ctx)
	if !h.ownGoal(ctx, tx, w, id, true) {
		return
	}
	if req.Amount < 0 {
		var current int64
		err := tx.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0)::bigint FROM goal_contributions WHERE goal_id = $1", id).Scan(&current)
		if err != nil {
			httpx.ServerError(w, "contribution sum", err)
			return
		}
		if current+req.Amount < 0 {
			httpx.WriteError(w, http.StatusBadRequest, "hedefte bu kadar birikim yok")
			return
		}
	}
	_, err = tx.Exec(ctx, "INSERT INTO goal_contributions (goal_id, amount, note) VALUES ($1, $2, $3)", id, req.Amount, note)
	if err != nil {
		httpx.ServerError(w, "contribution insert", err)
		return
	}
	_, err = tx.Exec(ctx, "UPDATE savings_goals SET updated_at = now() WHERE id = $1", id)
	if err != nil {
		httpx.ServerError(w, "contribution touch", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "contribution commit", err)
		return
	}
	h.writeOne(w, r, id, http.StatusCreated)
}

// DeleteContribution yanlış girilmiş bir katkıyı siler; güncel hedefi döner.
func (h *Handler) DeleteContribution(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	cid, ok := httpx.PathID(w, r, "cid")
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "contribution delete begin", err)
		return
	}
	defer tx.Rollback(ctx)
	if !h.ownGoal(ctx, tx, w, id, true) {
		return
	}
	var rest int64
	err = tx.QueryRow(ctx, `
SELECT COALESCE(SUM(amount), 0)::bigint FROM goal_contributions WHERE goal_id = $1 AND id <> $2`, id, cid).Scan(&rest)
	if err != nil {
		httpx.ServerError(w, "contribution delete sum", err)
		return
	}
	if rest < 0 {
		httpx.WriteError(w, http.StatusBadRequest, "bu kayit silinirse birikim eksiye duser")
		return
	}
	tag, err := tx.Exec(ctx, "DELETE FROM goal_contributions WHERE id = $1 AND goal_id = $2", cid, id)
	if err != nil {
		httpx.ServerError(w, "contribution delete", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "kayit bulunamadi")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "contribution delete commit", err)
		return
	}
	h.writeOne(w, r, id, http.StatusOK)
}
