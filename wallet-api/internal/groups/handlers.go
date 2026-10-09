package groups

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
)

type Handler struct {
	DB  *pgxpool.Pool
	Loc *time.Location
}

type Member struct {
	UserID   int       `json:"user_id"`
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joined_at"`
}

type Group struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int       `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	Members   []Member  `json:"members,omitempty"`
}

func validName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	return name, n >= 1 && n <= 64
}

func (h *Handler) isMember(ctx context.Context, groupID, userID int) (bool, error) {
	var ok bool
	err := h.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)",
		groupID, userID,
	).Scan(&ok)
	return ok, err
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
SELECT g.id, g.name, g.created_by, g.created_at
FROM groups g
JOIN group_members gm ON gm.group_id = g.id
WHERE gm.user_id = $1
ORDER BY g.id`,
		auth.UserID(r.Context()),
	)
	if err != nil {
		httpx.ServerError(w, "groups list", err)
		return
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Group, error) {
		var g Group
		err := row.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt)
		return g, err
	})
	if err != nil {
		httpx.ServerError(w, "groups list scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

type nameRequest struct {
	Name string `json:"name"`
}

// Create grubu oluşturur ve oluşturanı aynı DB transaction'ında üye yapar.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req nameRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	name, ok := validName(req.Name)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "grup adi 1-64 karakter olmali")
		return
	}

	ctx := r.Context()
	userID := auth.UserID(ctx)

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "group create begin", err)
		return
	}
	defer tx.Rollback(ctx)

	g := Group{Name: name, CreatedBy: userID}
	err = tx.QueryRow(ctx,
		"INSERT INTO groups (name, created_by) VALUES ($1, $2) RETURNING id, created_at",
		name, userID,
	).Scan(&g.ID, &g.CreatedAt)
	if err != nil {
		httpx.ServerError(w, "group create", err)
		return
	}
	_, err = tx.Exec(ctx,
		"INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)",
		g.ID, userID,
	)
	if err != nil {
		httpx.ServerError(w, "group create member", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "group create commit", err)
		return
	}
	h.writeGroup(w, r, g.ID, http.StatusCreated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	member, err := h.isMember(r.Context(), id, auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "group get check", err)
		return
	}
	if !member {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}
	h.writeGroup(w, r, id, http.StatusOK)
}

// writeGroup grubu üyeleriyle birlikte yazar. Üyelik kontrolü çağıranın işidir.
func (h *Handler) writeGroup(w http.ResponseWriter, r *http.Request, id, status int) {
	ctx := r.Context()
	var g Group
	err := h.DB.QueryRow(ctx,
		"SELECT id, name, created_by, created_at FROM groups WHERE id = $1", id,
	).Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		httpx.ServerError(w, "group get", err)
		return
	}
	rows, err := h.DB.Query(ctx, `
SELECT u.id, u.username, gm.joined_at
FROM group_members gm
JOIN users u ON u.id = gm.user_id
WHERE gm.group_id = $1
ORDER BY gm.joined_at, u.id`, id)
	if err != nil {
		httpx.ServerError(w, "group members", err)
		return
	}
	g.Members, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Member, error) {
		var m Member
		err := row.Scan(&m.UserID, &m.Username, &m.JoinedAt)
		return m, err
	})
	if err != nil {
		httpx.ServerError(w, "group members scan", err)
		return
	}
	httpx.WriteJSON(w, status, g)
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req nameRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	name, ok := validName(req.Name)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "grup adi 1-64 karakter olmali")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
UPDATE groups SET name = $1, updated_at = now()
WHERE id = $2 AND EXISTS (SELECT 1 FROM group_members WHERE group_id = $2 AND user_id = $3)`,
		name, id, auth.UserID(r.Context()),
	)
	if err != nil {
		httpx.ServerError(w, "group rename", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}
	h.writeGroup(w, r, id, http.StatusOK)
}

type addMemberRequest struct {
	Username string `json:"username"`
}

// AddMember: grubun herhangi bir üyesi, kullanıcı adıyla yeni üye ekleyebilir.
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req addMemberRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()

	member, err := h.isMember(ctx, id, auth.UserID(ctx))
	if err != nil {
		httpx.ServerError(w, "add member check", err)
		return
	}
	if !member {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}

	var newUserID int
	err = h.DB.QueryRow(ctx,
		"SELECT id FROM users WHERE username = $1",
		strings.ToLower(strings.TrimSpace(req.Username)),
	).Scan(&newUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "kullanici bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "add member user", err)
		return
	}

	_, err = h.DB.Exec(ctx,
		"INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)",
		id, newUserID,
	)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		httpx.WriteError(w, http.StatusConflict, "kullanici zaten grupta")
		return
	}
	if err != nil {
		httpx.ServerError(w, "add member insert", err)
		return
	}
	h.writeGroup(w, r, id, http.StatusOK)
}

// Leave: kullanıcı gruptan ayrılır; grubun işlemlerini artık göremez.
// Kendi hesabındaki grup işlemleri hesap geçmişinde kalmaya devam eder.
// Grupta açık borcu/alacağı varsa önce hesaplaşması gerekir.
func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	userID := auth.UserID(ctx)

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "group leave begin", err)
		return
	}
	defer tx.Rollback(ctx)

	// Üyelik satırı kilitlenir; bu sırada bu kullanıcıya yeni pay yazılamaz.
	var one int
	err = tx.QueryRow(ctx,
		"SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 FOR UPDATE",
		id, userID,
	).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "group leave lock", err)
		return
	}

	list, err := nets(ctx, tx, id)
	if err != nil {
		httpx.ServerError(w, "group leave balances", err)
		return
	}
	for _, n := range list {
		if n.UserID == userID && n.Net != 0 {
			httpx.WriteError(w, http.StatusConflict, "grupta acik borcunuz veya alacaginiz var; ayrilmadan once hesaplasin")
			return
		}
	}

	if _, err := tx.Exec(ctx,
		"DELETE FROM group_members WHERE group_id = $1 AND user_id = $2", id, userID,
	); err != nil {
		httpx.ServerError(w, "group leave", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "group leave commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
