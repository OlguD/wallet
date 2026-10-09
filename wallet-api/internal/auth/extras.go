package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/httpx"
)

var tourIDRe = regexp.MustCompile(`^[a-z0-9_.-]{1,48}$`)

type toursRequest struct {
	IDs []string `json:"ids"`
}

// MarkTours tanıtım turlarını görüldü olarak işaretler; tekrar gösterilmezler.
func (h *Handler) MarkTours(w http.ResponseWriter, r *http.Request) {
	var req toursRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > 50 {
		httpx.WriteError(w, http.StatusBadRequest, "ids 1-50 eleman olmali")
		return
	}
	for _, id := range req.IDs {
		if !tourIDRe.MatchString(id) {
			httpx.WriteError(w, http.StatusBadRequest, "gecersiz tur kimligi")
			return
		}
	}
	var seen []string
	err := h.DB.QueryRow(r.Context(), `
UPDATE users SET seen_tours = ARRAY(SELECT DISTINCT unnest(seen_tours || $2::text[]) ORDER BY 1), updated_at = now()
WHERE id = $1 RETURNING seen_tours`, UserID(r.Context()), req.IDs).Scan(&seen)
	if err != nil {
		httpx.ServerError(w, "mark tours", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"seen_tours": seen})
}

// RotateInboxToken yeni bir gelen kutusu anahtarı üretir (eskisi geçersiz olur).
// Anahtar sadece bu yanıtta görünür; sunucuda özeti saklanır.
func (h *Handler) RotateInboxToken(w http.ResponseWriter, r *http.Request) {
	token, hash, err := newToken()
	if err != nil {
		httpx.ServerError(w, "inbox token", err)
		return
	}
	_, err = h.DB.Exec(r.Context(), "UPDATE users SET inbox_token_hash = $2, updated_at = now() WHERE id = $1",
		UserID(r.Context()), hash)
	if err != nil {
		httpx.ServerError(w, "inbox token save", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"token": token})
}

// RevokeInboxToken gelen kutusu anahtarını kaldırır.
func (h *Handler) RevokeInboxToken(w http.ResponseWriter, r *http.Request) {
	_, err := h.DB.Exec(r.Context(), "UPDATE users SET inbox_token_hash = NULL, updated_at = now() WHERE id = $1",
		UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "inbox token revoke", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RequireInboxToken sadece gelen kutusu anahtarıyla (Authorization: Bearer)
// gelen istekleri kabul eder. Bu anahtar yalnızca dekont yüklemeye yetkilidir.
func (h *Handler) RequireInboxToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r)
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "anahtar gerekli")
			return
		}
		var userID int
		err := h.DB.QueryRow(r.Context(), "SELECT id FROM users WHERE inbox_token_hash = $1", hashToken(token)).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusUnauthorized, "anahtar gecersiz")
			return
		}
		if err != nil {
			log.Println("inbox auth:", err)
			httpx.WriteError(w, http.StatusInternalServerError, "sunucu hatasi")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, userID)))
	})
}
