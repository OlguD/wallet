package groups

import (
	"reflect"
	"testing"
)

func TestSuggest(t *testing.T) {
	got := suggest([]Net{
		{"TRY", 1, "a", 6000},
		{"TRY", 2, "b", -4000},
		{"TRY", 3, "c", -2000},
		{"USD", 1, "a", -50},
		{"USD", 2, "b", 50},
		{"USD", 3, "c", 0},
	})
	want := []Transfer{
		{"TRY", 2, "b", 1, "a", 4000},
		{"TRY", 3, "c", 1, "a", 2000},
		{"USD", 1, "a", 2, "b", 50},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	if got := suggest(nil); len(got) != 0 {
		t.Fatalf("bos: %v", got)
	}
}
