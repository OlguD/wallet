package reminders

import (
	"testing"
	"time"
)

func TestNextDue(t *testing.T) {
	loc := time.UTC
	d := func(y int, m time.Month, dd int) time.Time { return time.Date(y, m, dd, 0, 0, 0, 0, loc) }
	cases := []struct {
		due   int
		today time.Time
		want  time.Time
	}{
		{15, d(2026, 10, 10), d(2026, 10, 15)},
		{15, d(2026, 10, 15), d(2026, 10, 15)},
		{15, d(2026, 10, 16), d(2026, 11, 15)},
		{31, d(2026, 11, 5), d(2026, 11, 30)},
		{31, d(2026, 2, 28), d(2026, 2, 28)},
		{5, d(2026, 12, 20), d(2027, 1, 5)},
	}
	for _, c := range cases {
		if got := NextDue(c.due, c.today); !got.Equal(c.want) {
			t.Errorf("NextDue(%d, %s) = %s, want %s", c.due, c.today.Format("2006-01-02"), got.Format("2006-01-02"), c.want.Format("2006-01-02"))
		}
	}
}
