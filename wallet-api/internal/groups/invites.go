package groups

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/notify"
)

// Invite: gruba davet. Davet edilen kişi gelen kutusundan kabul edince üye olur.
type Invite struct {
	ID          int       `json:"id"`
	GroupID     int       `json:"group_id"`
	GroupName   string    `json:"group_name"`
	InviterID   int       `json:"inviter_id"`
	InviterName string    `json:"inviter_name"`
	InviteeID   int       `json:"invitee_id"`
	InviteeName string    `json:"invitee_name"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) pendingInvites(ctx context.Context, where string, arg any) ([]Invite, error) {
	rows, err := h.DB.Query(ctx, `
SELECT i.id, i.group_id, g.name, i.inviter_id, ui.username, i.invitee_id, ue.username, i.created_at
FROM group_invites i
JOIN groups g ON g.id = i.group_id
JOIN users ui ON ui.id = i.inviter_id
JOIN users ue ON ue.id = i.invitee_id
WHERE i.status = 'pending' AND `+where+`
ORDER BY i.created_at DESC`, arg)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Invite])
}

type inviteRequest struct {
	Username string `json:"username"`
}

// CreateInvite grubun bir üyesi, kullanıcı adıyla başka birini davet eder.
func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req inviteRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	me := auth.UserID(ctx)
	member, err := h.isMember(ctx, id, me)
	if err != nil {
		httpx.ServerError(w, "invite check", err)
		return
	}
	if !member {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}

	var inviteeID int
	err = h.DB.QueryRow(ctx, "SELECT id FROM users WHERE username = $1 AND deleted_at IS NULL",
		strings.ToLower(strings.TrimSpace(req.Username))).Scan(&inviteeID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "kullanici bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "invite user", err)
		return
	}
	if already, err := h.isMember(ctx, id, inviteeID); err != nil {
		httpx.ServerError(w, "invite member check", err)
		return
	} else if already {
		httpx.WriteError(w, http.StatusConflict, "kullanici zaten grupta")
		return
	}

	var inviteID int
	err = h.DB.QueryRow(ctx,
		"INSERT INTO group_invites (group_id, inviter_id, invitee_id) VALUES ($1, $2, $3) RETURNING id",
		id, me, inviteeID).Scan(&inviteID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		httpx.WriteError(w, http.StatusConflict, "bu kullanici zaten davet edildi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "invite insert", err)
		return
	}

	var groupName, inviter string
	h.DB.QueryRow(ctx, "SELECT g.name, u.username FROM groups g, users u WHERE g.id = $1 AND u.id = $2", id, me).Scan(&groupName, &inviter)
	h.Notify.Send(ctx, notify.Notice{
		UserID: inviteeID, Kind: "invite", RefID: &inviteID,
		Title: "Grup daveti: " + groupName,
		Body:  inviter + " seni \"" + groupName + "\" grubuna davet etti.",
		URL:   "/inbox",
	})
	h.writeGroup(w, r, id, http.StatusCreated)
}

// MyInvites kullanıcının bekleyen davetleri (gelen kutusu).
func (h *Handler) MyInvites(w http.ResponseWriter, r *http.Request) {
	list, err := h.pendingInvites(r.Context(), "i.invitee_id = $1", auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "my invites", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// respond daveti kilitler ve kullanıcıya ait olduğunu doğrular.
func (h *Handler) respond(w http.ResponseWriter, r *http.Request, accept bool) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	me := auth.UserID(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "invite respond begin", err)
		return
	}
	defer tx.Rollback(ctx)

	var groupID, inviterID int
	err = tx.QueryRow(ctx, `
SELECT group_id, inviter_id FROM group_invites
WHERE id = $1 AND invitee_id = $2 AND status = 'pending' FOR UPDATE`, id, me).Scan(&groupID, &inviterID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "davet bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "invite respond lock", err)
		return
	}
	status := "declined"
	if accept {
		status = "accepted"
		_, err = tx.Exec(ctx, "INSERT INTO group_members (group_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", groupID, me)
		if err != nil {
			httpx.ServerError(w, "invite accept member", err)
			return
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE group_invites SET status = $2, responded_at = now() WHERE id = $1", id, status); err != nil {
		httpx.ServerError(w, "invite respond update", err)
		return
	}
	// Davete ait bildirim okundu sayılır.
	if _, err := tx.Exec(ctx, "UPDATE notifications SET read_at = now() WHERE user_id = $1 AND kind = 'invite' AND ref_id = $2 AND read_at IS NULL", me, id); err != nil {
		httpx.ServerError(w, "invite respond notification", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "invite respond commit", err)
		return
	}
	if accept {
		var groupName, username string
		h.DB.QueryRow(ctx, "SELECT g.name, u.username FROM groups g, users u WHERE g.id = $1 AND u.id = $2", groupID, me).Scan(&groupName, &username)
		h.Notify.Send(ctx, notify.Notice{
			UserID: inviterID, Kind: "invite_accepted", RefID: &id,
			Title: username + " gruba katıldı", Body: "\"" + groupName + "\" davetini kabul etti.",
			URL: "/groups/" + strconv.Itoa(groupID),
		})
		h.writeGroup(w, r, groupID, http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request)  { h.respond(w, r, true) }
func (h *Handler) DeclineInvite(w http.ResponseWriter, r *http.Request) { h.respond(w, r, false) }

// CancelInvite grubun herhangi bir üyesi bekleyen daveti geri çekebilir.
func (h *Handler) CancelInvite(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	inviteID, ok := httpx.PathID(w, r, "iid")
	if !ok {
		return
	}
	ctx := r.Context()
	tag, err := h.DB.Exec(ctx, `
UPDATE group_invites i SET status = 'cancelled', responded_at = now()
WHERE i.id = $1 AND i.group_id = $2 AND i.status = 'pending'
  AND EXISTS (SELECT 1 FROM group_members gm WHERE gm.group_id = i.group_id AND gm.user_id = $3)`,
		inviteID, id, auth.UserID(ctx))
	if err != nil {
		httpx.ServerError(w, "invite cancel", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "davet bulunamadi")
		return
	}
	h.writeGroup(w, r, id, http.StatusOK)
}
