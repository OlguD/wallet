package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"

	"wallet-api/internal/httpx"
)

// Face ID / Touch ID ile giriş (WebAuthn passkey). Kullanıcı önce şifreyle
// girip Ayarlar'dan passkey ekler; sonra giriş ekranında kullanıcı adı
// yazmadan Face ID ile girer (discoverable credential).

const challengeTTL = 5 * time.Minute

// NewWebAuthn RP ayarlarından WebAuthn örneği kurar. rpID alan adıdır
// (wallet.ornek.com), origins tam adreslerdir (https://wallet.ornek.com).
func NewWebAuthn(rpID string, origins []string) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Cüzdan",
		RPOrigins:     origins,
	})
}

type passkeyUser struct {
	id     int
	name   string
	handle []byte
	creds  []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.handle }
func (u *passkeyUser) WebAuthnName() string                       { return u.name }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.name }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func (h *Handler) passkeysEnabled(w http.ResponseWriter) bool {
	if h.WebAuthn == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "face id girisi sunucuda kapali")
		return false
	}
	return true
}

func (h *Handler) loadCreds(ctx context.Context, u *passkeyUser) error {
	rows, err := h.DB.Query(ctx, "SELECT credential FROM passkeys WHERE user_id = $1", u.id)
	if err != nil {
		return err
	}
	u.creds, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (webauthn.Credential, error) {
		var c webauthn.Credential
		var raw []byte
		if err := r.Scan(&raw); err != nil {
			return c, err
		}
		return c, json.Unmarshal(raw, &c)
	})
	return err
}

// passkeyUserByID kullanıcıyı yükler; passkey kimliği yoksa oluşturur.
func (h *Handler) passkeyUserByID(ctx context.Context, id int) (*passkeyUser, error) {
	u := &passkeyUser{id: id}
	if err := h.DB.QueryRow(ctx, "SELECT username, webauthn_id FROM users WHERE id = $1", id).Scan(&u.name, &u.handle); err != nil {
		return nil, err
	}
	if u.handle == nil {
		handle := make([]byte, 32)
		rand.Read(handle)
		if err := h.DB.QueryRow(ctx,
			"UPDATE users SET webauthn_id = COALESCE(webauthn_id, $2) WHERE id = $1 RETURNING webauthn_id",
			id, handle).Scan(&u.handle); err != nil {
			return nil, err
		}
	}
	return u, h.loadCreds(ctx, u)
}

func (h *Handler) saveChallenge(ctx context.Context, kind string, userID *int, s *webauthn.SessionData) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	b := make([]byte, 24)
	rand.Read(b)
	id := base64.RawURLEncoding.EncodeToString(b)
	// Süresi geçenleri de temizle.
	if _, err := h.DB.Exec(ctx, "DELETE FROM webauthn_challenges WHERE expires_at < now()"); err != nil {
		return "", err
	}
	_, err = h.DB.Exec(ctx,
		"INSERT INTO webauthn_challenges (id, kind, user_id, session, expires_at) VALUES ($1, $2, $3, $4, $5)",
		id, kind, userID, raw, time.Now().Add(challengeTTL))
	return id, err
}

// takeChallenge tek kullanımlıktır: okunurken silinir.
func (h *Handler) takeChallenge(ctx context.Context, id, kind string, userID *int) (*webauthn.SessionData, error) {
	var raw []byte
	err := h.DB.QueryRow(ctx, `
DELETE FROM webauthn_challenges
WHERE id = $1 AND kind = $2 AND user_id IS NOT DISTINCT FROM $3 AND expires_at > now()
RETURNING session`, id, kind, userID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var s webauthn.SessionData
	return &s, json.Unmarshal(raw, &s)
}

type beginResponse struct {
	ChallengeID string `json:"challenge_id"`
	Options     any    `json:"options"`
}

// BeginPasskeyRegistration oturum açık kullanıcı için passkey oluşturma seçenekleri.
func (h *Handler) BeginPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.passkeysEnabled(w) {
		return
	}
	ctx := r.Context()
	me := UserID(ctx)
	u, err := h.passkeyUserByID(ctx, me)
	if err != nil {
		httpx.ServerError(w, "passkey user", err)
		return
	}
	exclude := make([]protocol.CredentialDescriptor, len(u.creds))
	for i, c := range u.creds {
		exclude[i] = c.Descriptor()
	}
	opts, session, err := h.WebAuthn.BeginRegistration(u,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		}),
		webauthn.WithExclusions(exclude),
	)
	if err != nil {
		httpx.ServerError(w, "passkey begin registration", err)
		return
	}
	id, err := h.saveChallenge(ctx, "register", &me, session)
	if err != nil {
		httpx.ServerError(w, "passkey save challenge", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, beginResponse{ChallengeID: id, Options: opts.Response})
}

type Passkey struct {
	ID         string     `json:"id"` // base64url
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// FinishPasskeyRegistration: ?challenge=ID&name=iPhone; gövde tarayıcının credential yanıtı.
func (h *Handler) FinishPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.passkeysEnabled(w) {
		return
	}
	ctx := r.Context()
	me := UserID(ctx)
	session, err := h.takeChallenge(ctx, r.URL.Query().Get("challenge"), "register", &me)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusBadRequest, "istek suresi doldu; tekrar dene")
		return
	}
	if err != nil {
		httpx.ServerError(w, "passkey take challenge", err)
		return
	}
	u, err := h.passkeyUserByID(ctx, me)
	if err != nil {
		httpx.ServerError(w, "passkey user", err)
		return
	}
	cred, err := h.WebAuthn.FinishRegistration(u, *session, r)
	if err != nil {
		log.Println("passkey finish registration:", err)
		httpx.WriteError(w, http.StatusBadRequest, "face id kaydi dogrulanamadi")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" || utf8.RuneCountInString(name) > 64 {
		name = "Face ID"
	}
	raw, _ := json.Marshal(cred)
	p := Passkey{ID: base64.RawURLEncoding.EncodeToString(cred.ID), Name: name}
	err = h.DB.QueryRow(ctx,
		"INSERT INTO passkeys (id, user_id, name, credential) VALUES ($1, $2, $3, $4) RETURNING created_at",
		cred.ID, me, name, raw).Scan(&p.CreatedAt)
	if err != nil {
		httpx.ServerError(w, "passkey insert", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) ListPasskeys(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(),
		"SELECT id, name, created_at, last_used_at FROM passkeys WHERE user_id = $1 ORDER BY created_at",
		UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "passkeys list", err)
		return
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Passkey, error) {
		var p Passkey
		var id []byte
		err := r.Scan(&id, &p.Name, &p.CreatedAt, &p.LastUsedAt)
		p.ID = base64.RawURLEncoding.EncodeToString(id)
		return p, err
	})
	if err != nil {
		httpx.ServerError(w, "passkeys scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) DeletePasskey(w http.ResponseWriter, r *http.Request) {
	id, err := base64.RawURLEncoding.DecodeString(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "bulunamadi")
		return
	}
	tag, err := h.DB.Exec(r.Context(), "DELETE FROM passkeys WHERE id = $1 AND user_id = $2", id, UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "passkey delete", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "bulunamadi")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BeginPasskeyLogin kullanıcı adı istemeden (discoverable) giriş seçenekleri.
func (h *Handler) BeginPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	if !h.passkeysEnabled(w) {
		return
	}
	opts, session, err := h.WebAuthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		httpx.ServerError(w, "passkey begin login", err)
		return
	}
	id, err := h.saveChallenge(r.Context(), "login", nil, session)
	if err != nil {
		httpx.ServerError(w, "passkey save challenge", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, beginResponse{ChallengeID: id, Options: opts.Response})
}

// FinishPasskeyLogin: ?challenge=ID; doğrulanırsa şifreli girişteki gibi oturum açar.
func (h *Handler) FinishPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	if !h.passkeysEnabled(w) {
		return
	}
	ctx := r.Context()
	session, err := h.takeChallenge(ctx, r.URL.Query().Get("challenge"), "login", nil)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusBadRequest, "istek suresi doldu; tekrar dene")
		return
	}
	if err != nil {
		httpx.ServerError(w, "passkey take challenge", err)
		return
	}
	var found *passkeyUser
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		var id int
		if err := h.DB.QueryRow(ctx, "SELECT id FROM users WHERE webauthn_id = $1", userHandle).Scan(&id); err != nil {
			return nil, err
		}
		u, err := h.passkeyUserByID(ctx, id)
		found = u
		return u, err
	}
	_, cred, err := h.WebAuthn.FinishPasskeyLogin(handler, *session, r)
	if err != nil || found == nil {
		log.Println("passkey finish login:", err)
		httpx.WriteError(w, http.StatusUnauthorized, "face id dogrulanamadi")
		return
	}
	raw, _ := json.Marshal(cred)
	if _, err := h.DB.Exec(ctx,
		"UPDATE passkeys SET credential = $2, last_used_at = now() WHERE id = $1", cred.ID, raw); err != nil {
		httpx.ServerError(w, "passkey update", err)
		return
	}

	var u User
	var deleted bool
	if err := h.DB.QueryRow(ctx, "SELECT id, username, created_at, deleted_at IS NOT NULL FROM users WHERE id = $1", found.id).
		Scan(&u.ID, &u.Username, &u.CreatedAt, &deleted); err != nil {
		httpx.ServerError(w, "passkey login user", err)
		return
	}
	if deleted {
		httpx.WriteError(w, http.StatusForbidden, "bu hesap silinmis")
		return
	}
	h.startSession(w, r, u, http.StatusOK)
}
