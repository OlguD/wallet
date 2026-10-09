// Package idem değişiklik isteklerini Idempotency-Key başlığına göre tekilleştirir.
// Çevrimdışı kuyruktaki bir istek, ilk denemede sunucuya ulaşıp yanıtı
// kaybolduysa tekrar gönderildiğinde ikinci kez işlenmez; saklanan yanıt döner.
package idem

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
)

var keyRe = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// TTL anahtarların saklanma süresi.
const TTL = 7 * 24 * time.Hour

type recorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if r.buf.Len() < 256<<10 {
		r.buf.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

// Middleware RequireAuth'tan sonra çalışır. Başlık yoksa ya da istek GET ise dokunmaz.
func Middleware(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" || r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			if !keyRe.MatchString(key) {
				httpx.WriteError(w, http.StatusBadRequest, "Idempotency-Key gecersiz")
				return
			}
			ctx := r.Context()
			userID := auth.UserID(ctx)

			tag, err := db.Exec(ctx, `
INSERT INTO idempotency_keys (user_id, key, method, path) VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING`, userID, key, r.Method, r.URL.Path)
			if err != nil {
				httpx.ServerError(w, "idem insert", err)
				return
			}
			if tag.RowsAffected() == 0 {
				replay(ctx, db, w, r, userID, key)
				return
			}

			rec := &recorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			// 5xx: kalıcı değil, tekrar denenebilsin diye anahtar silinir.
			if rec.status == 0 || rec.status >= 500 {
				db.Exec(context.WithoutCancel(ctx), "DELETE FROM idempotency_keys WHERE user_id = $1 AND key = $2", userID, key)
				return
			}
			_, err = db.Exec(context.WithoutCancel(ctx),
				"UPDATE idempotency_keys SET status = $3, body = $4 WHERE user_id = $1 AND key = $2",
				userID, key, rec.status, rec.buf.Bytes())
			if err != nil {
				log.Println("idem save:", err)
			}
		})
	}
}

func replay(ctx context.Context, db *pgxpool.Pool, w http.ResponseWriter, r *http.Request, userID int, key string) {
	var method, path string
	var status int
	var body []byte
	err := db.QueryRow(ctx, "SELECT method, path, status, body FROM idempotency_keys WHERE user_id = $1 AND key = $2",
		userID, key).Scan(&method, &path, &status, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusConflict, "istek isleniyor, tekrar deneyin")
		return
	}
	if err != nil {
		httpx.ServerError(w, "idem replay", err)
		return
	}
	if method != r.Method || path != r.URL.Path {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "Idempotency-Key baska bir istek icin kullanilmis")
		return
	}
	if status == 0 {
		httpx.WriteError(w, http.StatusConflict, "istek isleniyor, tekrar deneyin")
		return
	}
	w.Header().Set("Idempotent-Replay", "true")
	if len(body) > 0 {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	w.Write(body)
}

// Cleanup süresi dolan anahtarları periyodik olarak siler.
func Cleanup(ctx context.Context, db *pgxpool.Pool) {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		if _, err := db.Exec(ctx, "DELETE FROM idempotency_keys WHERE created_at < $1", time.Now().Add(-TTL)); err != nil && ctx.Err() == nil {
			log.Println("idem cleanup:", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
