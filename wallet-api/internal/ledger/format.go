package ledger

import (
	"strconv"
	"strings"
)

// FormatMoney kuruşu Türkçe gösterime çevirir: 123456 → "1.234,56".
func FormatMoney(k int64) string {
	neg := k < 0
	if neg {
		k = -k
	}
	i := strconv.FormatInt(k/100, 10)
	var b strings.Builder
	for n, r := range i {
		if n > 0 && (len(i)-n)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	s := b.String() + "," + strconv.FormatInt(100+k%100, 10)[1:]
	if neg {
		return "-" + s
	}
	return s
}

// DecimalTR basamak ayırıcısız ondalık (Excel/CSV için): 123456 → "1234,56".
func DecimalTR(k int64) string {
	sign := ""
	if k < 0 {
		sign, k = "-", -k
	}
	return sign + strconv.FormatInt(k/100, 10) + "," + strconv.FormatInt(100+k%100, 10)[1:]
}
