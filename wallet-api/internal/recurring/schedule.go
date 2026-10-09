package recurring

import "time"

// Next bir sonraki çalışma gününü verir. Aylık/yıllık kurallarda gün,
// anchorDay'e göre belirlenir ve kısa aylarda ay sonuna çekilir
// (31'inde başlayan kural: 31 Oca → 28/29 Şub → 31 Mar).
// Tarihler UTC gece yarısı olarak taşınır.
func Next(cur time.Time, freq string, interval, anchorDay int) time.Time {
	switch freq {
	case "weekly":
		return cur.AddDate(0, 0, 7*interval)
	case "yearly":
		return clampDay(cur.Year()+interval, cur.Month(), anchorDay)
	default: // monthly
		first := time.Date(cur.Year(), cur.Month()+time.Month(interval), 1, 0, 0, 0, 0, time.UTC)
		return clampDay(first.Year(), first.Month(), anchorDay)
	}
}

func clampDay(year int, month time.Month, day int) time.Time {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(year, month, min(day, last), 0, 0, 0, 0, time.UTC)
}
