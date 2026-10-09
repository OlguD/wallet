package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	cookieName = "session"
	sessionTTL = 90 * 24 * time.Hour
	// Kalan süre bunun altına düşünce oturum uzatılır (kayan süre).
	renewBelow = 60 * 24 * time.Hour
)

var errNoSession = errors.New("oturum yok")

func newToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func createSession(ctx context.Context, db *pgxpool.Pool, userID int) (string, time.Time, error) {
	token, hash, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(sessionTTL)
	_, err = db.Exec(ctx,
		"INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID, hash, expires,
	)
	return token, expires, err
}

// lookupSession geçerli oturumun kullanıcısını döner, gerekirse süresini uzatır.
func lookupSession(ctx context.Context, db *pgxpool.Pool, token string) (int, error) {
	hash := hashToken(token)
	var userID int
	var expires time.Time
	err := db.QueryRow(ctx,
		`SELECT s.user_id, s.expires_at FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND s.expires_at > now() AND u.deleted_at IS NULL`,
		hash,
	).Scan(&userID, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errNoSession
	}
	if err != nil {
		return 0, err
	}
	if time.Until(expires) < renewBelow {
		_, err = db.Exec(ctx,
			"UPDATE sessions SET expires_at = $1 WHERE token_hash = $2",
			time.Now().Add(sessionTTL), hash,
		)
		if err != nil {
			return 0, err
		}
	}
	return userID, nil
}

func deleteSession(ctx context.Context, db *pgxpool.Pool, token string) error {
	_, err := db.Exec(ctx, "DELETE FROM sessions WHERE token_hash = $1", hashToken(token))
	return err
}

// tokenFromRequest önce Authorization: Bearer başlığına, sonra cookie'ye bakar.
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if t, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(t)
		}
	}
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	return ""
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
