package transactions

import (
	"encoding/csv"
	"net/http"
	"time"

	"wallet-api/internal/auth"
	"wallet-api/internal/budgets"
	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
)

// ExportCSV kullanıcının işlemlerini Excel'in Türkçe ayarlarıyla açılacak
// biçimde (UTF-8 BOM, ";" ayraç, "," ondalık) indirir. ?from=&to= (varsayılan bu ay).
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	from, to, err := httpx.DateRange(r, h.Loc)
	if err != nil {
		httpx.Fail(w, "export range", err)
		return
	}
	ctx := r.Context()
	rows, err := h.DB.Query(ctx, `
SELECT t.occurred_at, a.name, a.currency, t.type::text, t.category, t.amount, t.description,
       t.counterparty_name, t.counterparty_iban, t.counterparty_bank, g.name, pa.name, b.amount
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN groups g ON g.id = t.group_id
LEFT JOIN transactions pt ON pt.id = t.transfer_peer_id
LEFT JOIN accounts pa ON pa.id = pt.account_id
LEFT JOIN balances b ON b.transaction_id = t.id
WHERE a.user_id = $1 AND t.occurred_at >= $2 AND t.occurred_at < $3
ORDER BY t.occurred_at, t.id`, auth.UserID(ctx), from, to)
	if err != nil {
		httpx.ServerError(w, "export", err)
		return
	}
	defer rows.Close()

	name := "cuzdan-" + from.Format("2006-01-02") + "_" + to.AddDate(0, 0, -1).Format("2006-01-02") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.Write([]string{"Tarih", "Saat", "Hesap", "Para birimi", "Tür", "Kategori", "Tutar", "İşlem sonrası bakiye",
		"Açıklama", "Karşı taraf", "IBAN", "Banka", "Grup", "Transfer hesabı"})
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	for rows.Next() {
		var at time.Time
		var account, currency, typ string
		var category, desc, cpName, cpIBAN, cpBank, group, peer *string
		var amount int64
		var balance *int64
		if err := rows.Scan(&at, &account, &currency, &typ, &category, &amount, &desc, &cpName, &cpIBAN, &cpBank, &group, &peer, &balance); err != nil {
			return
		}
		at = at.In(h.Loc)
		kind := map[string]string{"income": "Gelir", "expense": "Gider"}[typ]
		if peer != nil {
			kind = "Transfer"
		}
		signed := amount
		if typ == "expense" {
			signed = -amount
		}
		cat := ""
		if category != nil {
			cat = budgets.CategoryLabel(*category)
		}
		bal := ""
		if balance != nil {
			bal = ledger.DecimalTR(*balance)
		}
		cw.Write([]string{at.Format("02.01.2006"), at.Format("15:04"), account, currency, kind, cat, ledger.DecimalTR(signed), bal,
			str(desc), str(cpName), str(cpIBAN), str(cpBank), str(group), str(peer)})
	}
	cw.Flush()
}
