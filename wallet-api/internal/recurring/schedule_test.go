package recurring

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func TestNext(t *testing.T) {
	cases := []struct {
		cur, freq string
		interval  int
		anchor    int
		want      string
	}{
		{"2026-01-31", "monthly", 1, 31, "2026-02-28"},
		{"2026-02-28", "monthly", 1, 31, "2026-03-31"},
		{"2028-01-31", "monthly", 1, 31, "2028-02-29"},
		{"2026-11-15", "monthly", 2, 15, "2027-01-15"},
		{"2026-10-08", "weekly", 1, 8, "2026-10-15"},
		{"2026-10-08", "weekly", 2, 8, "2026-10-22"},
		{"2028-02-29", "yearly", 1, 29, "2029-02-28"},
		{"2029-02-28", "yearly", 3, 29, "2032-02-29"},
	}
	for _, c := range cases {
		got := Next(d(c.cur), c.freq, c.interval, c.anchor)
		if got.Format(time.DateOnly) != c.want {
			t.Errorf("Next(%s, %s, %d, %d) = %s, want %s",
				c.cur, c.freq, c.interval, c.anchor, got.Format(time.DateOnly), c.want)
		}
	}
}
