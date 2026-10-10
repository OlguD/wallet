package groups

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
)

// Net bir üyenin bir para birimindeki grup bakiyesi.
// Pozitif: gruptan alacaklı; negatif: gruba borçlu.
type Net struct {
	Currency string `json:"currency"`
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Net      int64  `json:"net"`
}

type Transfer struct {
	Currency     string `json:"currency"`
	FromUserID   int    `json:"from_user_id"`
	FromUsername string `json:"from_username"`
	ToUserID     int    `json:"to_user_id"`
	ToUsername   string `json:"to_username"`
	Amount       int64  `json:"amount"`
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// nets grubun tüm zamanlardaki borç/alacak durumunu hesaplar:
//   - gider ödeyen +tutar, gelir alan -tutar
//   - gider payı -pay, gelir payı +pay
//   - hesaplaşmada ödeyen +tutar, alan -tutar
func nets(ctx context.Context, q querier, groupID int) ([]Net, error) {
	rows, err := q.Query(ctx, `
WITH e AS (
  SELECT a.currency, a.user_id,
         CASE WHEN t.type = 'expense' THEN t.amount ELSE -t.amount END AS amt
  FROM transactions t JOIN accounts a ON a.id = t.account_id
  WHERE t.group_id = $1
  UNION ALL
  SELECT a.currency, s.user_id,
         CASE WHEN t.type = 'expense' THEN -s.amount ELSE s.amount END
  FROM transaction_splits s
  JOIN transactions t ON t.id = s.transaction_id
  JOIN accounts a ON a.id = t.account_id
  WHERE t.group_id = $1
  UNION ALL
  SELECT currency, from_user_id, amount FROM settlements WHERE group_id = $1 AND affects_balance
  UNION ALL
  SELECT currency, to_user_id, -amount FROM settlements WHERE group_id = $1 AND affects_balance
)
SELECT e.currency, e.user_id, u.username, SUM(e.amt)::bigint
FROM e JOIN users u ON u.id = e.user_id
GROUP BY e.currency, e.user_id, u.username
ORDER BY e.currency, e.user_id`, groupID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Net])
}

// suggest borçları en az sayıda ödemeyle kapatacak transferleri önerir
// (her para biriminde en büyük borçlu en büyük alacaklıya öder).
func suggest(list []Net) []Transfer {
	byCur := map[string][]Net{}
	var currencies []string
	for _, n := range list {
		if n.Net == 0 {
			continue
		}
		if _, ok := byCur[n.Currency]; !ok {
			currencies = append(currencies, n.Currency)
		}
		byCur[n.Currency] = append(byCur[n.Currency], n)
	}
	sort.Strings(currencies)

	out := []Transfer{}
	for _, cur := range currencies {
		var debtors, creditors []Net
		for _, n := range byCur[cur] {
			if n.Net < 0 {
				n.Net = -n.Net
				debtors = append(debtors, n)
			} else {
				creditors = append(creditors, n)
			}
		}
		order := func(s []Net) {
			sort.Slice(s, func(i, j int) bool {
				if s[i].Net != s[j].Net {
					return s[i].Net > s[j].Net
				}
				return s[i].UserID < s[j].UserID
			})
		}
		order(debtors)
		order(creditors)
		for i, j := 0, 0; i < len(debtors) && j < len(creditors); {
			amt := min(debtors[i].Net, creditors[j].Net)
			out = append(out, Transfer{
				Currency:     cur,
				FromUserID:   debtors[i].UserID,
				FromUsername: debtors[i].Username,
				ToUserID:     creditors[j].UserID,
				ToUsername:   creditors[j].Username,
				Amount:       amt,
			})
			debtors[i].Net -= amt
			creditors[j].Net -= amt
			if debtors[i].Net == 0 {
				i++
			}
			if creditors[j].Net == 0 {
				j++
			}
		}
	}
	return out
}

func (h *Handler) requireMember(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return 0, false
	}
	member, err := h.isMember(r.Context(), id, auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "group member check", err)
		return 0, false
	}
	if !member {
		httpx.WriteError(w, http.StatusNotFound, "grup bulunamadi")
		return 0, false
	}
	return id, true
}

// Balances: kim kime borçlu. "balances" üyelerin net durumu,
// "suggestions" borçları kapatmak için önerilen ödemeler.
func (h *Handler) Balances(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	list, err := nets(r.Context(), h.DB, id)
	if err != nil {
		httpx.ServerError(w, "group balances", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"balances":    list,
		"suggestions": suggest(list),
	})
}

type memberSpend struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Paid     int64  `json:"paid"`
	Share    int64  `json:"share"`
}

type currencySpend struct {
	Currency     string        `json:"currency"`
	TotalExpense int64         `json:"total_expense"`
	Members      []memberSpend `json:"members"`
}

// Summary: seçilen dönemde grubun giderleri; her üyenin ödediği ve payına düşen.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	from, to, err := httpx.DateRange(r, h.Loc)
	if err != nil {
		httpx.Fail(w, "group summary range", err)
		return
	}
	rows, err := h.DB.Query(r.Context(), `
WITH gt AS (
  SELECT t.id, t.amount, a.currency, a.user_id AS payer
  FROM transactions t JOIN accounts a ON a.id = t.account_id
  WHERE t.group_id = $1 AND t.type = 'expense'
    AND t.occurred_at >= $2 AND t.occurred_at < $3
), x AS (
  SELECT currency, payer AS user_id, amount AS paid, 0::bigint AS share FROM gt
  UNION ALL
  SELECT gt.currency, s.user_id, 0, s.amount
  FROM gt JOIN transaction_splits s ON s.transaction_id = gt.id
)
SELECT x.currency, x.user_id, u.username, SUM(x.paid)::bigint, SUM(x.share)::bigint
FROM x JOIN users u ON u.id = x.user_id
GROUP BY x.currency, x.user_id, u.username
ORDER BY x.currency, x.user_id`, id, from, to)
	if err != nil {
		httpx.ServerError(w, "group summary", err)
		return
	}
	defer rows.Close()

	out := []currencySpend{}
	for rows.Next() {
		var cur string
		var m memberSpend
		if err := rows.Scan(&cur, &m.UserID, &m.Username, &m.Paid, &m.Share); err != nil {
			httpx.ServerError(w, "group summary scan", err)
			return
		}
		if len(out) == 0 || out[len(out)-1].Currency != cur {
			out = append(out, currencySpend{Currency: cur})
		}
		last := &out[len(out)-1]
		last.TotalExpense += m.Paid
		last.Members = append(last.Members, m)
	}
	if err := rows.Err(); err != nil {
		httpx.ServerError(w, "group summary rows", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"from":       from.Format(time.DateOnly),
		"to":         to.Format(time.DateOnly),
		"currencies": out,
	})
}
