// Package rates döviz kurlarını Sun Döviz'den (sundoviz.com) alır. Kurlar
// TL karşılığıdır (USD, EUR, GBP); "alış" büronun döviz alırken, "satış"
// döviz satarken uyguladığı kurdur. Sonuç kısa süre önbellekte tutulur;
// kaynak erişilemezse son başarılı veri "stale" olarak döner.
package rates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"wallet-api/internal/httpx"
)

const DefaultURL = "https://online.sundoviz.com/services/apirates.php?app=online"

// Uygulamanın desteklediği dövizler (TRY dışındakiler).
var currencies = []string{"USD", "EUR", "GBP"}

type Quote struct {
	Buy  float64 `json:"buy"`  // alış
	Sell float64 `json:"sell"` // satış
}

type Channel struct {
	// Rates: 1 birim dövizin TL karşılığı.
	Rates map[string]Quote `json:"rates"`
	// Cross: "EUR/USD" gibi çapraz (arbitraj) kurlar.
	Cross map[string]Quote `json:"cross"`
}

type Snapshot struct {
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"` // kaynağın son güncelleme zamanı
	FetchedAt time.Time `json:"fetched_at"`
	Stale     bool      `json:"stale"`
	// CrossMode true ise dövizden dövize çevirmede çapraz kur, değilse TL üzerinden hesaplanır.
	CrossMode bool `json:"cross_mode"`
	// Online: EFT/havale kurları, Desk: gişe (nakit) kurları.
	Online Channel `json:"online"`
	Desk   Channel `json:"desk"`
}

type Handler struct {
	URL    string
	Loc    *time.Location
	TTL    time.Duration
	Client *http.Client

	mu    sync.Mutex
	last  *Snapshot
	until time.Time
}

func New(loc *time.Location) *Handler {
	return &Handler{
		URL:    DefaultURL,
		Loc:    loc,
		TTL:    2 * time.Minute,
		Client: &http.Client{Timeout: 8 * time.Second},
	}
}

// Get önbellekteki kuru döner; süresi dolmuşsa kaynaktan yeniler.
func (h *Handler) Get(ctx context.Context) (*Snapshot, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	if h.last != nil && now.Before(h.until) {
		return h.last, nil
	}
	s, err := h.fetch(ctx)
	if err != nil {
		if h.last == nil {
			return nil, err
		}
		// Kaynak geçici olarak erişilemez: eskiyi döndür, kısa süre sonra yeniden dene.
		stale := *h.last
		stale.Stale = true
		h.until = now.Add(30 * time.Second)
		return &stale, nil
	}
	h.last = s
	h.until = now.Add(h.TTL)
	return s, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	s, err := h.Get(r.Context())
	if err != nil {
		httpx.ServerError(w, "rates fetch", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

// Kaynağın JSON biçimi; sayılar metin olarak gelir.
type rawChannel struct {
	Kur      map[string]string `json:"kur"`
	Arbitraj map[string]string `json:"arbitraj"`
}

type rawResponse struct {
	Exra struct {
		UpdateTime   string          `json:"update_time"`
		CalcModeShow json.RawMessage `json:"calc_mode_show"`
	} `json:"exra"`
	Online *rawChannel `json:"online"`
	Gise   *rawChannel `json:"gise"`
}

func (h *Handler) fetch(ctx context.Context) (*Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "wallet-api/1.0")
	res, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sundoviz: http %d", res.StatusCode)
	}
	var raw rawResponse
	if err := json.NewDecoder(http.MaxBytesReader(nil, res.Body, 1<<20)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("sundoviz: %w", err)
	}
	return parse(raw, h.Loc, time.Now())
}

func parse(raw rawResponse, loc *time.Location, now time.Time) (*Snapshot, error) {
	if raw.Online == nil || raw.Gise == nil {
		return nil, errors.New("sundoviz: eksik veri")
	}
	online, err := parseChannel(raw.Online)
	if err != nil {
		return nil, err
	}
	desk, err := parseChannel(raw.Gise)
	if err != nil {
		return nil, err
	}
	s := &Snapshot{Source: "sundoviz.com", FetchedAt: now, Online: online, Desk: desk, UpdatedAt: now}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", raw.Exra.UpdateTime, loc); err == nil {
		s.UpdatedAt = t
	}
	switch v := strings.Trim(strings.ToLower(string(raw.Exra.CalcModeShow)), `"`); v {
	case "true", "1":
		s.CrossMode = true
	}
	return s, nil
}

func parseChannel(c *rawChannel) (Channel, error) {
	ch := Channel{Rates: map[string]Quote{}, Cross: map[string]Quote{}}
	for _, cur := range currencies {
		k := strings.ToLower(cur)
		q, ok := quote(c.Kur, k)
		if !ok {
			return ch, fmt.Errorf("sundoviz: %s kuru yok", cur)
		}
		ch.Rates[cur] = q
	}
	// "eur_usd_alis" → "EUR/USD"
	for key := range c.Arbitraj {
		pair, ok := strings.CutSuffix(key, "_alis")
		if !ok {
			continue
		}
		if q, ok := quote(c.Arbitraj, pair); ok {
			ch.Cross[strings.ToUpper(strings.Replace(pair, "_", "/", 1))] = q
		}
	}
	return ch, nil
}

func quote(m map[string]string, prefix string) (Quote, bool) {
	buy, err1 := strconv.ParseFloat(strings.TrimSpace(m[prefix+"_alis"]), 64)
	sell, err2 := strconv.ParseFloat(strings.TrimSpace(m[prefix+"_satis"]), 64)
	if err1 != nil || err2 != nil || buy <= 0 || sell <= 0 {
		return Quote{}, false
	}
	return Quote{Buy: buy, Sell: sell}, true
}
