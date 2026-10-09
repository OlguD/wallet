package transactions

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
	"wallet-api/internal/receipts"
)

// Counterparty işlem oluştururken gönderilen karşı taraf alanları.
type Counterparty struct {
	Name *string `json:"counterparty_name"`
	IBAN *string `json:"counterparty_iban"`
	Bank *string `json:"counterparty_bank"`
}

type counterpartyPatch struct {
	Name, IBAN, Bank httpx.Optional[string]
}

func normField(v *string, max int, what string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(s) > max {
		return nil, httpx.BadRequest(what + " cok uzun")
	}
	return &s, nil
}

// setCounterparty karşı taraf alanlarını günceller; sadece hesabın sahibi.
// IBAN boşluksuz büyük harfe çevrilir; banka boşsa IBAN'dan bulunur.
func setCounterparty(ctx context.Context, tx pgx.Tx, userID, txID int, p counterpartyPatch) error {
	if !p.Name.Set && !p.IBAN.Set && !p.Bank.Set {
		return nil
	}
	name, err := normField(p.Name.Value, 128, "karsi taraf adi")
	if err != nil {
		return err
	}
	var iban *string
	if p.IBAN.Value != nil {
		v := strings.ToUpper(strings.ReplaceAll(*p.IBAN.Value, " ", ""))
		if v != "" {
			if len(v) > 34 || !strings.HasPrefix(v, "TR") && len(v) < 15 {
				return httpx.BadRequest("IBAN gecersiz")
			}
			iban = &v
		}
	}
	bank, err := normField(p.Bank.Value, 64, "banka adi")
	if err != nil {
		return err
	}
	if bank == nil && iban != nil && !p.Bank.Set {
		if b := receipts.BankByIBAN(*iban); b != "" {
			bank = &b
		}
	}
	tag, err := tx.Exec(ctx, `
UPDATE transactions t SET
  counterparty_name = CASE WHEN $3 THEN $4 ELSE t.counterparty_name END,
  counterparty_iban = CASE WHEN $5 THEN $6 ELSE t.counterparty_iban END,
  counterparty_bank = CASE WHEN $7 OR ($5 AND $8::text IS NOT NULL) THEN $8 ELSE t.counterparty_bank END
FROM accounts a
WHERE t.id = $1 AND a.id = t.account_id AND a.user_id = $2`,
		txID, userID, p.Name.Set, name, p.IBAN.Set, iban, p.Bank.Set, bank)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.NotFound("islem bulunamadi")
	}
	return nil
}

type counterpartyRow struct {
	Key      string  `json:"key"`
	Name     *string `json:"name"`
	IBAN     *string `json:"iban"`
	Bank     *string `json:"bank"`
	Currency string  `json:"currency"`
	Count    int     `json:"count"`
	Total    int64   `json:"total"`
	LastAt   string  `json:"last_at"`
}

// Counterparties "kime gitti / kimden geldi" raporu: karşı tarafa göre toplamlar.
// ?type=expense|income (varsayılan expense), ?from=&to= (opsiyonel).
func (h *Handler) Counterparties(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	typ := r.URL.Query().Get("type")
	if typ == "" {
		typ = "expense"
	}
	if typ != "expense" && typ != "income" {
		httpx.WriteError(w, http.StatusBadRequest, "type 'income' veya 'expense' olmali")
		return
	}
	var from, to any
	if r.URL.Query().Get("from") != "" || r.URL.Query().Get("to") != "" {
		f, t, err := httpx.DateRange(r, h.Loc)
		if err != nil {
			httpx.Fail(w, "counterparties range", err)
			return
		}
		from, to = f, t
	}
	rows, err := h.DB.Query(ctx, `
WITH x AS (
  SELECT t.*, a.currency,
         COALESCE(t.counterparty_iban, 'name:' || lower(t.counterparty_name)) AS k
  FROM transactions t JOIN accounts a ON a.id = t.account_id
  WHERE a.user_id = $1 AND t.type = $2
    AND (t.counterparty_iban IS NOT NULL OR t.counterparty_name IS NOT NULL)
    AND ($3::timestamptz IS NULL OR t.occurred_at >= $3)
    AND ($4::timestamptz IS NULL OR t.occurred_at < $4)
)
SELECT k,
       (array_agg(counterparty_name ORDER BY counterparty_name IS NULL, occurred_at DESC))[1],
       (array_agg(counterparty_iban ORDER BY counterparty_iban IS NULL, occurred_at DESC))[1],
       (array_agg(counterparty_bank ORDER BY counterparty_bank IS NULL, occurred_at DESC))[1],
       currency, count(*)::int, SUM(amount)::bigint, to_char(max(occurred_at), 'YYYY-MM-DD')
FROM x GROUP BY k, currency
ORDER BY SUM(amount) DESC
LIMIT 200`, auth.UserID(ctx), typ, from, to)
	if err != nil {
		httpx.ServerError(w, "counterparties", err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[counterpartyRow])
	if err != nil {
		httpx.ServerError(w, "counterparties scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}
