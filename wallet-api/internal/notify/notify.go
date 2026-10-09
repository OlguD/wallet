// Package notify uygulama içi bildirimleri (gelen kutusu) yazar ve kullanıcının
// cihazlarına Web Push ile gönderir. VAPID anahtarları yoksa sadece uygulama
// içi bildirim yazılır.
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
)

type Notifier struct {
	DB *pgxpool.Pool
	// VAPID anahtarları (VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY); Subject: "mailto:..." veya site adresi.
	PublicKey  string
	PrivateKey string
	Subject    string
}

// Notice tek bir bildirim. RefID verilirse aynı (kullanıcı, tür, ref) için
// ikinci bildirim yazılmaz.
type Notice struct {
	UserID int
	Kind   string // invite, invite_accepted, group_expense, recurring, budget:2026-10 ...
	Title  string
	Body   string
	URL    string
	RefID  *int
}

func (n *Notifier) PushEnabled() bool { return n.PublicKey != "" && n.PrivateKey != "" }

func clip(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}

// Send bildirimi yazar ve arka planda push gönderir. Hata loglanır; çağıranın
// asıl işlemini bozmaz.
func (n *Notifier) Send(ctx context.Context, x Notice) {
	if n == nil {
		return
	}
	x.Title, x.Body = clip(x.Title, 160), clip(x.Body, 400)
	var id int
	err := n.DB.QueryRow(ctx, `
INSERT INTO notifications (user_id, kind, title, body, url, ref_id)
SELECT $1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6
FROM users WHERE id = $1 AND deleted_at IS NULL
ON CONFLICT (user_id, kind, ref_id) WHERE ref_id IS NOT NULL DO NOTHING
RETURNING id`, x.UserID, x.Kind, x.Title, x.Body, x.URL, x.RefID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return // tekrar ya da silinmiş kullanıcı
	}
	if err != nil {
		log.Println("notify insert:", err)
		return
	}
	if n.PushEnabled() {
		go n.push(x)
	}
}

func (n *Notifier) push(x Notice) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rows, err := n.DB.Query(ctx, "SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1", x.UserID)
	if err != nil {
		log.Println("push subs:", err)
		return
	}
	subs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (webpush.Subscription, error) {
		var s webpush.Subscription
		return s, r.Scan(&s.Endpoint, &s.Keys.P256dh, &s.Keys.Auth)
	})
	if err != nil {
		log.Println("push subs scan:", err)
		return
	}
	payload, _ := json.Marshal(map[string]string{"title": x.Title, "body": x.Body, "url": x.URL, "tag": x.Kind})
	for i := range subs {
		res, err := webpush.SendNotificationWithContext(ctx, payload, &subs[i], &webpush.Options{
			Subscriber:      n.Subject,
			VAPIDPublicKey:  n.PublicKey,
			VAPIDPrivateKey: n.PrivateKey,
			TTL:             24 * 60 * 60,
			Urgency:         webpush.UrgencyNormal,
		})
		if err != nil {
			log.Println("push send:", err)
			continue
		}
		res.Body.Close()
		// Abonelik geçersizleşmiş (uygulama silinmiş, izin geri alınmış): temizle.
		if res.StatusCode == http.StatusGone || res.StatusCode == http.StatusNotFound {
			n.DB.Exec(ctx, "DELETE FROM push_subscriptions WHERE endpoint = $1", subs[i].Endpoint)
		}
	}
}

type Item struct {
	ID        int        `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	URL       *string    `json:"url"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// List son bildirimler ve okunmamış sayısı.
func (n *Notifier) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserID(ctx)
	rows, err := n.DB.Query(ctx, `
SELECT id, kind, title, body, url, read_at, created_at FROM notifications
WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT 50`, userID)
	if err != nil {
		httpx.ServerError(w, "notifications", err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
	if err != nil {
		httpx.ServerError(w, "notifications scan", err)
		return
	}
	var unread int
	if err := n.DB.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL", userID).Scan(&unread); err != nil {
		httpx.ServerError(w, "notifications unread", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
}

type readRequest struct {
	IDs []int `json:"ids"` // boşsa hepsi
}

// MarkRead bildirimleri okundu yapar.
func (n *Notifier) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req readRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	_, err := n.DB.Exec(r.Context(), `
UPDATE notifications SET read_at = now()
WHERE user_id = $1 AND read_at IS NULL AND (cardinality($2::int[]) = 0 OR id = ANY($2))`,
		auth.UserID(r.Context()), req.IDs)
	if err != nil {
		httpx.ServerError(w, "notifications read", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Key istemcinin abone olurken kullanacağı VAPID açık anahtarı.
func (n *Notifier) Key(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"enabled": n.PushEnabled(), "public_key": n.PublicKey})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	ExpirationTime any `json:"expirationTime"`
}

// Subscribe tarayıcının PushSubscription'ını kaydeder.
func (n *Notifier) Subscribe(w http.ResponseWriter, r *http.Request) {
	if !n.PushEnabled() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "bildirimler sunucuda kapali")
		return
	}
	var req subscribeRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if !strings.HasPrefix(req.Endpoint, "https://") || len(req.Endpoint) > 1000 || req.Keys.P256dh == "" || req.Keys.Auth == "" ||
		len(req.Keys.P256dh) > 200 || len(req.Keys.Auth) > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "gecersiz abonelik")
		return
	}
	_, err := n.DB.Exec(r.Context(), `
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES ($1, $2, $3, $4)
ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
		auth.UserID(r.Context()), req.Endpoint, req.Keys.P256dh, req.Keys.Auth)
	if err != nil {
		httpx.ServerError(w, "push subscribe", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (n *Notifier) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req unsubscribeRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	_, err := n.DB.Exec(r.Context(), "DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2", auth.UserID(r.Context()), req.Endpoint)
	if err != nil {
		httpx.ServerError(w, "push unsubscribe", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Test kullanıcının cihazlarına deneme bildirimi gönderir.
func (n *Notifier) Test(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	n.Send(r.Context(), Notice{UserID: userID, Kind: "test", Title: "Bildirimler açık 🎉", Body: "Cüzdan'dan bildirim alacaksın.", URL: "/inbox"})
	w.WriteHeader(http.StatusNoContent)
}
