package rates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const sample = `{"exra":{"SunDovizApi":"3.0","update_time":"2026-10-09 08:04:34","calc_mode_show":false},
"online":{"kur":{"usd_alis":"49.10","usd_satis":"49.40","eur_alis":"55.10","eur_satis":"55.50","gbp_alis":"65.00","gbp_satis":"65.45"},
 "arbitraj":{"eur_usd_alis":"1.1180","eur_usd_satis":"1.1281"}},
"gise":{"kur":{"usd_alis":"49.00","usd_satis":"49.50","eur_alis":"55.00","eur_satis":"55.60","gbp_alis":"64.90","gbp_satis":"65.50"},"arbitraj":{}}}`

func TestFetchCacheAndStale(t *testing.T) {
	var calls atomic.Int32
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if fail.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		w.Write([]byte(sample))
	}))
	defer srv.Close()

	loc, _ := time.LoadLocation("Europe/Istanbul")
	h := New(loc)
	h.URL = srv.URL
	h.TTL = time.Hour

	s, err := h.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Online.Rates["USD"]; got.Buy != 49.10 || got.Sell != 49.40 {
		t.Fatalf("USD = %+v", got)
	}
	if got := s.Desk.Rates["GBP"]; got.Buy != 64.90 {
		t.Fatalf("desk GBP = %+v", got)
	}
	if got := s.Online.Cross["EUR/USD"]; got.Buy != 1.1180 || got.Sell != 1.1281 {
		t.Fatalf("EUR/USD = %+v", got)
	}
	if s.CrossMode {
		t.Fatal("cross mode should be false")
	}
	if want := time.Date(2026, 10, 9, 8, 4, 34, 0, loc); !s.UpdatedAt.Equal(want) {
		t.Fatalf("updated_at = %v", s.UpdatedAt)
	}

	// Önbellekten gelir.
	h.Get(context.Background())
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}

	// Süre dolmuş ve kaynak çökmüş: eski veri stale olarak döner.
	fail.Store(true)
	h.until = time.Time{}
	s, err = h.Get(context.Background())
	if err != nil || !s.Stale || s.Online.Rates["EUR"].Sell != 55.50 {
		t.Fatalf("stale = %+v, %v", s, err)
	}
}

func TestFetchErrorWithoutCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"online":{"kur":{}}}`))
	}))
	defer srv.Close()
	h := New(time.UTC)
	h.URL = srv.URL
	if _, err := h.Get(context.Background()); err == nil {
		t.Fatal("expected error for incomplete data")
	}
}
