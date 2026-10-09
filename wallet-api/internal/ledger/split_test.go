package ledger

import (
	"reflect"
	"testing"
)

func sum(s []Split) int64 {
	var t int64
	for _, x := range s {
		t += x.Amount
	}
	return t
}

func TestEqualSplit(t *testing.T) {
	got := EqualSplit(1000, []int{3, 1, 2})
	want := []Split{{1, 334}, {2, 333}, {3, 333}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := EqualSplit(1, []int{1, 2, 3}); sum(got) != 1 || got[0].Amount != 1 {
		t.Fatalf("1 kurus: %v", got)
	}
}

func TestProportional(t *testing.T) {
	old := []Split{{1, 6000}, {2, 4000}}
	got := Proportional(15000, old)
	want := []Split{{1, 9000}, {2, 6000}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	got = Proportional(100, []Split{{1, 1}, {2, 1}, {3, 1}})
	if sum(got) != 100 || got[0].Amount != 34 {
		t.Fatalf("got %v", got)
	}

	// 64 biti aşan ara çarpım.
	big := int64(1_000_000_000_000)
	got = Proportional(big, []Split{{1, big - 1}, {2, 1}})
	if sum(got) != big {
		t.Fatalf("overflow: %v", got)
	}

	got = Proportional(10, []Split{{1, 0}, {2, 0}})
	if sum(got) != 10 {
		t.Fatalf("sifir paylar: %v", got)
	}
}
