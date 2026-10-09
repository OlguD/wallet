package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/httpx"
)

const (
	// MaxFailures art arda bu kadar hatalı şifreden sonra hesap kilitlenir.
	MaxFailures = 5
	// LockDuration kilidin süresi.
	LockDuration = 5 * time.Minute
)

// lockedFor kullanıcı adı kilitliyse kalan süreyi döner.
func (h *Handler) lockedFor(ctx context.Context, username string) (time.Duration, error) {
	var until *time.Time
	err := h.DB.QueryRow(ctx, "SELECT locked_until FROM login_attempts WHERE username = $1", username).Scan(&until)
	if errors.Is(err, pgx.ErrNoRows) || until == nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return max(0, time.Until(*until)), nil
}

// recordFailure hatalı denemeyi sayar; sınıra ulaşınca kilitler ve kalan süreyi döner.
func (h *Handler) recordFailure(ctx context.Context, username string) (time.Duration, int, error) {
	var failures int
	var until *time.Time
	err := h.DB.QueryRow(ctx, `
INSERT INTO login_attempts (username, failures, updated_at) VALUES ($1, 1, now())
ON CONFLICT (username) DO UPDATE SET
  -- Kilit süresi dolmuşsa sayaç baştan başlar.
  failures = CASE WHEN login_attempts.locked_until IS NOT NULL AND login_attempts.locked_until <= now()
                  THEN 1 ELSE login_attempts.failures + 1 END,
  locked_until = CASE WHEN login_attempts.locked_until IS NOT NULL AND login_attempts.locked_until <= now()
                      THEN NULL ELSE login_attempts.locked_until END,
  updated_at = now()
RETURNING failures, locked_until`, username).Scan(&failures, &until)
	if err != nil {
		return 0, 0, err
	}
	if failures >= MaxFailures {
		lockUntil := time.Now().Add(LockDuration)
		_, err = h.DB.Exec(ctx, "UPDATE login_attempts SET locked_until = $2, failures = 0 WHERE username = $1", username, lockUntil)
		return LockDuration, 0, err
	}
	return 0, MaxFailures - failures, nil
}

func (h *Handler) clearFailures(ctx context.Context, username string) {
	if _, err := h.DB.Exec(ctx, "DELETE FROM login_attempts WHERE username = $1", username); err != nil {
		log.Println("clear login failures:", err)
	}
}

// failLogin hatalı şifre yanıtını yazar; kalan deneme hakkını ya da kilidi bildirir.
func (h *Handler) failLogin(w http.ResponseWriter, r *http.Request, username string) {
	wait, left, err := h.recordFailure(r.Context(), username)
	if err != nil {
		httpx.ServerError(w, "login failure record", err)
		return
	}
	if wait > 0 {
		writeLocked(w, wait)
		return
	}
	w.Header().Set("X-Attempts-Left", strconv.Itoa(left))
	httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{
		"error":         "kullanici adi veya sifre hatali",
		"attempts_left": left,
	})
}

func writeLocked(w http.ResponseWriter, wait time.Duration) {
	secs := int(wait.Seconds() + 0.999)
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	httpx.WriteJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":       fmt.Sprintf("cok fazla hatali deneme; %d dakika sonra tekrar deneyin", (secs+59)/60),
		"retry_after": secs,
	})
}

// verifyPassword oturumdaki kullanıcının şifresini doğrular (şifre değiştirme,
// hesap silme). Aynı 5 deneme / 5 dakika kuralı uygulanır.
func (h *Handler) verifyPassword(w http.ResponseWriter, r *http.Request, password string) (string, bool) {
	ctx := r.Context()
	var username, hash string
	err := h.DB.QueryRow(ctx, "SELECT username, COALESCE(password_hash, '') FROM users WHERE id = $1", UserID(ctx)).Scan(&username, &hash)
	if err != nil {
		httpx.ServerError(w, "verify password", err)
		return "", false
	}
	if wait, err := h.lockedFor(ctx, username); err != nil {
		httpx.ServerError(w, "verify lock", err)
		return "", false
	} else if wait > 0 {
		writeLocked(w, wait)
		return "", false
	}
	if hash == "" || !CheckPassword(hash, password) {
		wait, left, err := h.recordFailure(ctx, username)
		if err != nil {
			httpx.ServerError(w, "verify failure", err)
			return "", false
		}
		if wait > 0 {
			writeLocked(w, wait)
			return "", false
		}
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "mevcut sifre hatali", "attempts_left": left})
		return "", false
	}
	h.clearFailures(ctx, username)
	return username, true
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword şifreyi değiştirir; bu cihaz dışındaki tüm oturumlar kapanır.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		httpx.WriteError(w, http.StatusBadRequest, "sifre 8-72 karakter arasinda olmali")
		return
	}
	if req.NewPassword == req.CurrentPassword {
		httpx.WriteError(w, http.StatusBadRequest, "yeni sifre eskisiyle ayni olamaz")
		return
	}
	if _, ok := h.verifyPassword(w, r, req.CurrentPassword); !ok {
		return
	}
	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		httpx.ServerError(w, "change password hash", err)
		return
	}
	ctx := r.Context()
	userID := UserID(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "change password begin", err)
		return
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "UPDATE users SET password_hash = $2, password_changed_at = now(), updated_at = now() WHERE id = $1", userID, hash); err != nil {
		httpx.ServerError(w, "change password update", err)
		return
	}
	// Diğer cihazlardaki oturumlar kapanır; bu oturum açık kalır.
	if _, err := tx.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2", userID, hashToken(tokenFromRequest(r))); err != nil {
		httpx.ServerError(w, "change password sessions", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "change password commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount hesabı kapatır. Veriler silinmez: işlemler, hesaplar, grup
// geçmişi ve borç kayıtları yerinde kalır (grup arkadaşlarının hesapları
// bozulmasın diye). Kullanıcı giriş yapamaz, davet alamaz; oturumları, gelen
// kutusu anahtarı ve bildirim abonelikleri kaldırılır, bekleyen davetleri iptal edilir.
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req deleteAccountRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if _, ok := h.verifyPassword(w, r, req.Password); !ok {
		return
	}
	ctx := r.Context()
	userID := UserID(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.ServerError(w, "delete account begin", err)
		return
	}
	defer tx.Rollback(ctx)
	for _, q := range []string{
		"UPDATE users SET deleted_at = now(), inbox_token_hash = NULL, updated_at = now() WHERE id = $1",
		"DELETE FROM sessions WHERE user_id = $1",
		"DELETE FROM push_subscriptions WHERE user_id = $1",
		"UPDATE group_invites SET status = 'cancelled', responded_at = now() WHERE status = 'pending' AND (invitee_id = $1 OR inviter_id = $1)",
		"UPDATE recurring_transactions rt SET active = false, updated_at = now() FROM accounts a WHERE a.id = rt.account_id AND a.user_id = $1",
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			httpx.ServerError(w, "delete account", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.ServerError(w, "delete account commit", err)
		return
	}
	clearSessionCookie(w, h.SecureCookie)
	w.WriteHeader(http.StatusNoContent)
}
