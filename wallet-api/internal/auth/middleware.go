package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"wallet-api/internal/httpx"
)

type ctxKey struct{}

// UserID RequireAuth'tan geçmiş bir isteğin kullanıcı kimliğini döner.
func UserID(ctx context.Context) int {
	id, _ := ctx.Value(ctxKey{}).(int)
	return id
}

// RequireAuth geçerli oturumu olmayan istekleri 401 ile reddeder.
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r)
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "oturum acmaniz gerekiyor")
			return
		}
		userID, err := lookupSession(r.Context(), h.DB, token)
		if errors.Is(err, errNoSession) {
			httpx.WriteError(w, http.StatusUnauthorized, "oturum gecersiz veya suresi dolmus")
			return
		}
		if err != nil {
			log.Println("auth middleware:", err)
			httpx.WriteError(w, http.StatusInternalServerError, "sunucu hatasi")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
