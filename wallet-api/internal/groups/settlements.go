package groups

import (
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
)

var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

type Settlement struct {
	ID            int       `json:"id"`
	GroupID       int       `json:"group_id"`
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
}

const selectSettlement = `
SELECT s.id, s.group_id, s.from_user_id, fu.username, s.to_user_id, tu.username,
       s.amount, s.currency, s.transaction_id, s.note, s.created_by, s.occurred_at, s.created_at
FROM settlements s
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
	FromUserID int        `json:"from_user_id"`
	ToUserID   int        `json:"to_user_id"`
	Amount     int64      `json:"amount"`
	Currency   string     `json:"currency"`
	AccountID  *int       `json:"account_id"`
	Note       *string    `json:"note"`
	OccurredAt *time.Time `json:"occurred_at"`
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
		if currency != "" && currency != accCurrency {
			httpx.WriteError(w, http.StatusBadRequest, "para birimi secilen hesabin para birimiyle ayni olmali")
			return
		}
		currency = accCurrency

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
		desc := fmt.Sprintf("Hesaplasma (%s): %s -> %s", groupName, fromName, toName)
		id, err := ledger.Create(ctx, tx, userID, ledger.Input{
			AccountID:   *req.AccountID,
			Type:        typ,
			Amount:      req.Amount,
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
INSERT INTO settlements (group_id, from_user_id, to_user_id, amount, currency, transaction_id, note, created_by, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
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
	httpx.WriteJSON(w, http.StatusCreated, s)
}

// DeleteSettlement: ödemenin tarafları veya kaydı giren silebilir.
// Hesaba işlenmiş ödeme işlemi de silinir.
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

	var txID *int
	err = tx.QueryRow(ctx, `
DELETE FROM settlements s
WHERE s.id = $1 AND s.group_id = $2
  AND $3 IN (s.from_user_id, s.to_user_id, s.created_by)
  AND EXISTS (SELECT 1 FROM group_members WHERE group_id = $2 AND user_id = $3)
RETURNING s.transaction_id`,
		sid, groupID, userID,
	).Scan(&txID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "hesaplasma bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "settlement delete", err)
		return
	}
	if txID != nil {
		if err := ledger.Remove(ctx, tx, *txID); err != nil {
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
