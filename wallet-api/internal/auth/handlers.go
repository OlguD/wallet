package auth

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/httpx"
)

type Handler struct {
	DB *pgxpool.Pool
	// SecureCookie HTTPS arkasında true olmalı.
	SecureCookie bool
	// WebAuthn nil ise Face ID (passkey) girişi kapalıdır.
	WebAuthn *webauthn.WebAuthn
}

var usernameRe = regexp.MustCompile(`^[a-z0-9_.-]{3,32}$`)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	// SeenTours kullanıcının gördüğü tanıtım turları (sadece /me'de dolar).
	SeenTours []string `json:"seen_tours,omitempty"`
	// HasInboxToken: iOS Kestirme için gelen kutusu anahtarı oluşturulmuş mu.
	HasInboxToken bool `json:"has_inbox_token"`
}

type authResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

func normalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	req.Username = normalizeUsername(req.Username)
	if !usernameRe.MatchString(req.Username) {
		httpx.WriteError(w, http.StatusBadRequest,
			"kullanici adi 3-32 karakter olmali; harf, rakam, _ . - icerebilir")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		httpx.WriteError(w, http.StatusBadRequest, "sifre 8-72 karakter arasinda olmali")
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		httpx.ServerError(w, "register hash", err)
		return
	}

	var u User
	err = h.DB.QueryRow(r.Context(),
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username, created_at",
		req.Username, hash,
	).Scan(&u.ID, &u.Username, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteError(w, http.StatusConflict, "bu kullanici adi alinmis")
			return
		}
		httpx.ServerError(w, "register insert", err)
		return
	}

	h.startSession(w, r, u, http.StatusCreated)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	req.Username = normalizeUsername(req.Username)

	ctx := r.Context()
	if wait, err := h.lockedFor(ctx, req.Username); err != nil {
		httpx.ServerError(w, "login lock check", err)
		return
	} else if wait > 0 {
		writeLocked(w, wait)
		return
	}

	var u User
	var hash *string
	var deleted bool
	err := h.DB.QueryRow(ctx,
		"SELECT id, username, created_at, password_hash, deleted_at IS NOT NULL FROM users WHERE username = $1",
		req.Username,
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &hash, &deleted)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		httpx.ServerError(w, "login select", err)
		return
	}

	ok := false
	if err != nil || hash == nil {
		CheckPassword(string(dummyHash), req.Password)
	} else {
		ok = CheckPassword(*hash, req.Password)
	}
	if !ok {
		h.failLogin(w, r, req.Username)
		return
	}
	h.clearFailures(ctx, req.Username)
	if deleted {
		httpx.WriteError(w, http.StatusForbidden, "bu hesap silinmis")
		return
	}

	h.startSession(w, r, u, http.StatusOK)
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, u User, status int) {
	token, expires, err := createSession(r.Context(), h.DB, u.ID)
	if err != nil {
		httpx.ServerError(w, "create session", err)
		return
	}
	setSessionCookie(w, token, expires, h.SecureCookie)
	httpx.WriteJSON(w, status, authResponse{User: u, Token: token})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := tokenFromRequest(r); token != "" {
		if err := deleteSession(r.Context(), h.DB, token); err != nil {
			httpx.ServerError(w, "logout", err)
			return
		}
	}
	clearSessionCookie(w, h.SecureCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	var u User
	err := h.DB.QueryRow(r.Context(),
		"SELECT id, username, created_at, seen_tours, inbox_token_hash IS NOT NULL FROM users WHERE id = $1",
		UserID(r.Context()),
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &u.SeenTours, &u.HasInboxToken)
	if u.SeenTours == nil {
		u.SeenTours = []string{}
	}
	if err != nil {
		httpx.ServerError(w, "me", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}
