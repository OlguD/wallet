package ledger

import "testing"

func TestFormatMoney(t *testing.T) {
	cases := map[int64]string{0: "0,00", 5: "0,05", 123456: "1.234,56", 100000000: "1.000.000,00", -2550: "-25,50"}
	for in, want := range cases {
		if got := FormatMoney(in); got != want {
			t.Errorf("FormatMoney(%d) = %q, want %q", in, got, want)
		}
	}
	if got := DecimalTR(123456); got != "1234,56" {
		t.Errorf("DecimalTR = %q", got)
	}
}
