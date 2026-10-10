// Package reminders zamanlanmış hatırlatmaları gönderir: kredi kartı son
// ödeme günü, ertesi gün işlenecek düzenli ödemeler ve ay başında geçen
// ayın özeti. Bildirimler notify'ın (kullanıcı, tür, ref) tekilliğiyle bir
// kez gider; tür alanı tarihi içerir.
package reminders

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/budgets"
	"wallet-api/internal/ledger"
	"wallet-api/internal/notify"
)

// SendFrom: bu saatten önce bildirim gönderilmez (gece rahatsız etmesin).
const SendFrom = 9

type Runner struct {
	DB     *pgxpool.Pool
	Loc    *time.Location
	Notify *notify.Notifier
}

func (r *Runner) Start(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			r.Run(ctx, time.Now().In(r.Loc))
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// Run tüm hatırlatmaları bir kez kontrol eder.
func (r *Runner) Run(ctx context.Context, now time.Time) {
	if now.Hour() < SendFrom {
		return
	}
	for name, f := range map[string]func(context.Context, time.Time) error{
		"card due": r.cardDue, "recurring tomorrow": r.recurringTomorrow, "monthly": r.monthly,
	} {
		if err := f(ctx, now); err != nil && ctx.Err() == nil {
			log.Printf("reminders %s: %v", name, err)
		}
	}
}

var symbols = map[string]string{"TRY": "₺", "USD": "$", "EUR": "€", "GBP": "£"}

func money(k int64, cur string) string {
	if s, ok := symbols[cur]; ok {
		return ledger.FormatMoney(k) + " " + s
	}
	return ledger.FormatMoney(k) + " " + cur
}

func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// NextDue bugün ya da sonraki ilk son ödeme tarihi; kısa aylarda ay sonu.
func NextDue(dueDay int, today time.Time) time.Time {
	today = day(today)
	for k := 0; ; k++ {
		first := time.Date(today.Year(), today.Month()+time.Month(k), 1, 0, 0, 0, 0, today.Location())
		last := first.AddDate(0, 1, -1).Day()
		d := first.AddDate(0, 0, min(dueDay, last)-1)
		if !d.Before(today) {
			return d
		}
	}
}

// cardDue borcu olan kartlar için son ödemeden 3 gün önce ve son gün haber verir.
func (r *Runner) cardDue(ctx context.Context, now time.Time) error {
	rows, err := r.DB.Query(ctx, `
SELECT a.id, a.user_id, a.name, a.currency, a.due_day, `+ledger.BalanceSQL+`
FROM accounts a JOIN users u ON u.id = a.user_id
WHERE a.kind = 'card' AND a.due_day IS NOT NULL AND u.deleted_at IS NULL`)
	if err != nil {
		return err
	}
	type card struct {
		id, user, dueDay int
		name, cur        string
		balance          int64
	}
	var cards []card
	for rows.Next() {
		var c card
		if err := rows.Scan(&c.id, &c.user, &c.name, &c.cur, &c.dueDay, &c.balance); err != nil {
			rows.Close()
			return err
		}
		cards = append(cards, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	today := day(now)
	for _, c := range cards {
		if c.balance >= 0 {
			continue // borç yok
		}
		due := NextDue(c.dueDay, today)
		left := int(due.Sub(today).Hours() / 24)
		var title string
		switch left {
		case 3:
			title = c.name + ": son ödemeye 3 gün kaldı"
		case 0:
			title = c.name + ": son ödeme günü bugün"
		default:
			continue
		}
		id := c.id
		r.Notify.Send(ctx, notify.Notice{
			UserID: c.user, Kind: fmt.Sprintf("card_due:%s:%d", due.Format("2006-01-02"), left), RefID: &id,
			Title: title,
			Body:  fmt.Sprintf("Güncel borç %s · son ödeme %s", money(-c.balance, c.cur), due.Format("02.01.2006")),
			URL:   fmt.Sprintf("/accounts/%d", c.id),
		})
	}
	return nil
}

// recurringTomorrow yarın işlenecek düzenli giderleri bir gün önceden haber verir.
func (r *Runner) recurringTomorrow(ctx context.Context, now time.Time) error {
	tomorrow := day(now).AddDate(0, 0, 1)
	rows, err := r.DB.Query(ctx, `
SELECT rt.id, a.user_id, rt.amount, a.currency, COALESCE(NULLIF(rt.description, ''), rt.category, '')
FROM recurring_transactions rt
JOIN accounts a ON a.id = rt.account_id
JOIN users u ON u.id = a.user_id
WHERE rt.active AND rt.type = 'expense' AND rt.next_run_on = $1 AND u.deleted_at IS NULL
  AND (rt.end_on IS NULL OR rt.end_on >= $1)`, tomorrow)
	if err != nil {
		return err
	}
	type due struct {
		id, user  int
		amount    int64
		cur, what string
	}
	var list []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.user, &d.amount, &d.cur, &d.what); err != nil {
			rows.Close()
			return err
		}
		list = append(list, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, d := range list {
		what := d.what
		if l := budgets.CategoryLabel(what); l != what && l != "" {
			what = l
		}
		if what == "" {
			what = "Düzenli ödeme"
		}
		id := d.id
		r.Notify.Send(ctx, notify.Notice{
			UserID: d.user, Kind: "rec_due:" + tomorrow.Format("2006-01-02"), RefID: &id,
			Title: "Yarın: " + what,
			Body:  money(d.amount, d.cur) + " yarın otomatik işlenecek.",
			URL:   "/recurring",
		})
	}
	return nil
}

// monthly ayın ilk gününde geçen ayın özetini gönderir.
func (r *Runner) monthly(ctx context.Context, now time.Time) error {
	if now.Day() != 1 {
		return nil
	}
	to := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, r.Loc)
	from := to.AddDate(0, -1, 0)
	rows, err := r.DB.Query(ctx, `
SELECT a.user_id, a.currency,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0)::bigint,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)::bigint
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN users u ON u.id = a.user_id
WHERE t.occurred_at >= $1 AND t.occurred_at < $2 AND u.deleted_at IS NULL
  AND t.transfer_peer_id IS NULL AND t.category IS DISTINCT FROM 'opening'
GROUP BY 1, 2`, from, to)
	if err != nil {
		return err
	}
	type tot struct {
		cur             string
		income, expense int64
	}
	byUser := map[int][]tot{}
	for rows.Next() {
		var u int
		var x tot
		if err := rows.Scan(&u, &x.cur, &x.income, &x.expense); err != nil {
			rows.Close()
			return err
		}
		byUser[u] = append(byUser[u], x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	month := monthNames[from.Month()-1]
	for u, list := range byUser {
		// En çok hareket olan para birimi önce (genelde TL).
		sort.Slice(list, func(i, j int) bool { return list[i].income+list[i].expense > list[j].income+list[j].expense })
		var parts []string
		for _, x := range list {
			parts = append(parts, fmt.Sprintf("Gelir %s, gider %s", money(x.income, x.cur), money(x.expense, x.cur)))
		}
		uid := u
		r.Notify.Send(ctx, notify.Notice{
			UserID: u, Kind: "monthly:" + from.Format("2006-01"), RefID: &uid,
			Title: month + " özetin hazır",
			Body:  strings.Join(parts, " · ") + ". Kategoriler ve geçen ayla karşılaştırma için dokun.",
			URL:   "/report?month=" + from.Format("2006-01"),
		})
	}
	return nil
}

var monthNames = [12]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}
