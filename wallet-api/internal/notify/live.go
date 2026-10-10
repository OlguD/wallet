package notify

import (
	"net/http"
	"sync"
	"time"

	"wallet-api/internal/auth"
)

// Canlı güncelleme: açık uygulamalar GET /events (Server-Sent Events) ile
// bağlı kalır; kullanıcıya bildirim yazılınca "inbox" olayı gider, istemci
// gelen kutusunu ve verileri tazeler. Push'tan bağımsızdır, tek sunucu
// örneği içindir (bellekte).
type live struct {
	mu   sync.Mutex
	subs map[int]map[chan struct{}]struct{}
	stop chan struct{}
	once sync.Once
}

func (l *live) stopCh() chan struct{} {
	l.once.Do(func() { l.stop = make(chan struct{}) })
	return l.stop
}

func (l *live) subscribe(userID int) (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	l.mu.Lock()
	if l.subs == nil {
		l.subs = map[int]map[chan struct{}]struct{}{}
	}
	if l.subs[userID] == nil {
		l.subs[userID] = map[chan struct{}]struct{}{}
	}
	l.subs[userID][ch] = struct{}{}
	l.mu.Unlock()
	return ch, func() {
		l.mu.Lock()
		delete(l.subs[userID], ch)
		if len(l.subs[userID]) == 0 {
			delete(l.subs, userID)
		}
		l.mu.Unlock()
	}
}

// Ping kullanıcının açık uygulamalarına "yenile" der; bekleyen varsa birleşir.
func (n *Notifier) Ping(userID int) {
	if n == nil {
		return
	}
	n.live.mu.Lock()
	defer n.live.mu.Unlock()
	for ch := range n.live.subs[userID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// StopLive açık olay akışlarını kapatır (sunucu kapanırken).
func (n *Notifier) StopLive() { close(n.live.stopCh()) }

const keepAlive = 25 * time.Second

// Events kullanıcının canlı olay akışı (text/event-stream).
func (n *Notifier) Events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	// Sunucunun Read/WriteTimeout'u uzun süren akışı kesmesin.
	rc.SetReadDeadline(time.Time{})
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		http.Error(w, "stream desteklenmiyor", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("X-Accel-Buffering", "no")

	ch, unsubscribe := n.live.subscribe(auth.UserID(r.Context()))
	defer unsubscribe()
	stop := n.live.stopCh()

	write := func(s string) bool {
		if _, err := w.Write([]byte(s)); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !write("retry: 5000\n: ok\n\n") {
		return
	}
	t := time.NewTicker(keepAlive)
	defer t.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-stop:
			return
		case <-ch:
			if !write("event: inbox\ndata: {}\n\n") {
				return
			}
		case <-t.C:
			if !write(": ping\n\n") {
				return
			}
		}
	}
}
