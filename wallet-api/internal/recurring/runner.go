package recurring

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/httpx"
	"wallet-api/internal/ledger"
)

// Runner vadesi gelen tekrar kurallarından işlem üretir.
type Runner struct {
	DB  *pgxpool.Pool
	Loc *time.Location
}

// Start başlangıçta ve sonra her interval'da RunDue çalıştırır.
func (r *Runner) Start(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			if err := r.RunDue(ctx); err != nil && ctx.Err() == nil {
				log.Println("recurring run:", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// RunDue vadesi gelmiş tüm kuralları işler. Kaçırılan dönemler de
// (örn. sunucu kapalıyken) tek tek oluşturulur.
func (r *Runner) RunDue(ctx context.Context) error {
	for {
		done, err := r.runOne(ctx)
		if err != nil || done {
			return err
		}
	}
}

type rule struct {
	id, accountID, owner int
	groupID              *int
	typ                  string
	amount               int64
	category             *string
	description          *string
	split                *ledger.SplitSpec
	frequency            string
	interval, anchorDay  int
	next                 time.Time
	endOn                *time.Time
}

// runOne bir kuralı kilitleyip vadesi gelen tüm dönemlerini tek DB
// transaction'ında üretir. SKIP LOCKED sayesinde birden fazla sunucu
// örneği aynı kuralı iki kez işlemez.
func (r *Runner) runOne(ctx context.Context) (bool, error) {
	today := httpx.Today(r.Loc)

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var k rule
	err = tx.QueryRow(ctx, `
SELECT rt.id, rt.account_id, a.user_id, rt.group_id, rt.type, rt.amount, rt.category, rt.description,
       rt.split, rt.frequency, rt.interval_count, rt.anchor_day, rt.next_run_on, rt.end_on
FROM recurring_transactions rt
JOIN accounts a ON a.id = rt.account_id
WHERE rt.active AND rt.next_run_on <= $1
ORDER BY rt.next_run_on, rt.id
LIMIT 1
FOR UPDATE OF rt SKIP LOCKED`, today,
	).Scan(&k.id, &k.accountID, &k.owner, &k.groupID, &k.typ, &k.amount, &k.category, &k.description,
		&k.split, &k.frequency, &k.interval, &k.anchorDay, &k.next, &k.endOn)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	for !k.next.After(today) && (k.endOn == nil || !k.next.After(*k.endOn)) {
		in := ledger.Input{
			AccountID:   k.accountID,
			GroupID:     k.groupID,
			Type:        k.typ,
			Amount:      k.amount,
			Category:    k.category,
			Description: k.description,
			OccurredAt:  time.Date(k.next.Year(), k.next.Month(), k.next.Day(), 0, 0, 0, 0, r.Loc),
			Split:       k.split,
			RecurringID: &k.id,
		}
		_, err := ledger.Create(ctx, tx, k.owner, in)
		var appErr *httpx.Error
		if errors.As(err, &appErr) && in.Split != nil {
			// Paylaşımdaki biri gruptan ayrılmış olabilir: güncel üyelere eşit böl.
			log.Printf("recurring %d: paylasim gecersiz (%s), esit bolunuyor", k.id, appErr.Msg)
			in.Split = nil
			_, err = ledger.Create(ctx, tx, k.owner, in)
		}
		if errors.As(err, &appErr) {
			// Kural artık uygulanamıyor (örn. sahibi gruptan ayrıldı): durdur.
			log.Printf("recurring %d: durduruldu: %s", k.id, appErr.Msg)
			tx.Rollback(ctx)
			_, err = r.DB.Exec(ctx,
				"UPDATE recurring_transactions SET active = false, updated_at = now() WHERE id = $1", k.id)
			return false, err
		}
		if err != nil {
			return false, err
		}
		k.next = Next(k.next, k.frequency, k.interval, k.anchorDay)
	}

	active := k.endOn == nil || !k.next.After(*k.endOn)
	_, err = tx.Exec(ctx,
		"UPDATE recurring_transactions SET next_run_on = $2, active = $3, updated_at = now() WHERE id = $1",
		k.id, k.next, active,
	)
	if err != nil {
		return false, err
	}
	return false, tx.Commit(ctx)
}
