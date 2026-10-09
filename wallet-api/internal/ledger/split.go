package ledger

import (
	"math/bits"
	"slices"
	"sort"
)

type Split struct {
	UserID int   `json:"user_id"`
	Amount int64 `json:"amount"`
}

// SplitSpec bir grup işleminin üyeler arasında nasıl bölüneceği.
//   - {"type":"equal","user_ids":[1,2]}: listelenenler arasında eşit
//   - {"type":"exact","shares":[{"user_id":1,"amount":700},{"user_id":2,"amount":300}]}
//
// Verilmezse grubun tüm güncel üyeleri arasında eşit bölünür.
type SplitSpec struct {
	Type    string  `json:"type"`
	UserIDs []int   `json:"user_ids,omitempty"`
	Shares  []Split `json:"shares,omitempty"`
}

// EqualSplit tutarı kullanıcılar arasında eşit böler; artan kuruşlar
// küçük id'li kullanıcılardan başlayarak birer birer dağıtılır.
func EqualSplit(amount int64, userIDs []int) []Split {
	ids := slices.Clone(userIDs)
	slices.Sort(ids)
	n := int64(len(ids))
	out := make([]Split, len(ids))
	for i, id := range ids {
		out[i] = Split{UserID: id, Amount: amount / n}
		if int64(i) < amount%n {
			out[i].Amount++
		}
	}
	return out
}

// Proportional mevcut payların oranını koruyarak tutarı yeniden böler
// (en büyük kalan yöntemi). Toplam her zaman amount'a eşittir.
func Proportional(amount int64, old []Split) []Split {
	var total uint64
	for _, s := range old {
		total += uint64(s.Amount)
	}
	if total == 0 {
		ids := make([]int, len(old))
		for i, s := range old {
			ids[i] = s.UserID
		}
		return EqualSplit(amount, ids)
	}

	type part struct {
		idx int
		rem uint64
	}
	out := make([]Split, len(old))
	parts := make([]part, len(old))
	var assigned int64
	for i, s := range old {
		// amount*share/total; ara çarpım 64 biti aşabileceği için 128 bit.
		hi, lo := bits.Mul64(uint64(amount), uint64(s.Amount))
		q, r := bits.Div64(hi, lo, total)
		out[i] = Split{UserID: s.UserID, Amount: int64(q)}
		parts[i] = part{i, r}
		assigned += int64(q)
	}
	sort.SliceStable(parts, func(a, b int) bool {
		if parts[a].rem != parts[b].rem {
			return parts[a].rem > parts[b].rem
		}
		return out[parts[a].idx].UserID < out[parts[b].idx].UserID
	})
	for i := int64(0); i < amount-assigned; i++ {
		out[parts[i].idx].Amount++
	}
	return out
}
