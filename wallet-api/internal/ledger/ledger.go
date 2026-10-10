// Package ledger işlem yazma kurallarını tek yerde toplar: hesap kilidi,
// grup payları ve bakiye anlık görüntüleri. Tüm fonksiyonlar çağıranın
// açtığı DB transaction'ı içinde çalışır.
package ledger

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"wallet-api/internal/httpx"
)

// MaxAmount: kuruş cinsinden üst sınır (10 milyar birim); taşmaya karşı.
const MaxAmount int64 = 1_000_000_000_000

// BalanceSQL "a" takma adlı accounts satırının güncel bakiyesini verir:
// tarih sırasına göre en son işlemden sonraki bakiye (işlem yoksa 0).
const BalanceSQL = `COALESCE((
  SELECT b.amount FROM balances b
  JOIN transactions bt ON bt.id = b.transaction_id
  WHERE bt.account_id = a.id
  ORDER BY bt.occurred_at DESC, bt.id DESC
  LIMIT 1), 0)`

type Input struct {
	AccountID   int
	GroupID     *int
	Type        string
	Amount      int64
	Category    *string
	Description *string
	OccurredAt  time.Time
	Split       *SplitSpec
	RecurringID *int
}

func ValidateAmount(amount int64) error {
	if amount <= 0 || amount > MaxAmount {
		return httpx.BadRequest("tutar pozitif olmali (kurus cinsinden)")
	}
	return nil
}

func validate(typ string, amount int64, occurredAt time.Time) error {
	if typ != "income" && typ != "expense" {
		return httpx.BadRequest("type 'income' veya 'expense' olmali")
	}
	if err := ValidateAmount(amount); err != nil {
		return err
	}
	if occurredAt.After(time.Now().Add(24 * time.Hour)) {
		return httpx.BadRequest("islem tarihi gelecekte olamaz; ileri tarihli islemler icin tekrarlayan islem kullanin")
	}
	return nil
}

// CategoryOpening kart eklenirken girilen mevcut borç; gelir/gider
// toplamlarına sayılmaz (Summary bunu dışarıda bırakır).
const CategoryOpening = "opening"

var categories = map[string]map[string]bool{
	"expense": {"groceries": true, "bills": true, "transport": true, "food": true, "rent": true,
		"subscription": true, "gift": true, "health": true, "shopping": true, "other": true,
		CategoryOpening: true},
	"income": {"salary": true, "extra": true, "gift": true, "other": true},
}

// ValidateCategory kategorinin işlem tipine uygun olduğunu denetler; nil serbesttir.
func ValidateCategory(typ string, cat *string) error {
	if cat == nil || categories[typ][*cat] {
		return nil
	}
	return httpx.BadRequest("kategori bu islem tipi icin gecersiz")
}

// NormalizeDescription boşlukları kırpar; boşsa nil döner.
func NormalizeDescription(d *string) (*string, error) {
	if d == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*d)
	if utf8.RuneCountInString(s) > 255 {
		return nil, httpx.BadRequest("aciklama en fazla 255 karakter olabilir")
	}
	if s == "" {
		return nil, nil
	}
	return &s, nil
}

// Create actorID'nin kendi hesabına işlem ekler. Hesap satırı FOR UPDATE ile
// kilitlenir; aynı hesaba eşzamanlı yazımlar sıraya girer.
func Create(ctx context.Context, tx pgx.Tx, actorID int, in Input) (int, error) {
	if err := validate(in.Type, in.Amount, in.OccurredAt); err != nil {
		return 0, err
	}
	if err := ValidateCategory(in.Type, in.Category); err != nil {
		return 0, err
	}
	desc, err := NormalizeDescription(in.Description)
	if err != nil {
		return 0, err
	}

	var owner int
	err = tx.QueryRow(ctx, "SELECT user_id FROM accounts WHERE id = $1 FOR UPDATE", in.AccountID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != actorID) {
		return 0, httpx.NotFound("hesap bulunamadi")
	}
	if err != nil {
		return 0, err
	}

	var splits []Split
	if in.GroupID != nil {
		splits, err = ResolveSplits(ctx, tx, *in.GroupID, actorID, in.Amount, in.Split)
		if err != nil {
			return 0, err
		}
	} else if in.Split != nil {
		return 0, httpx.BadRequest("paylasim sadece grup islemlerinde kullanilabilir")
	}

	var id int
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (account_id, group_id, type, amount, category, description, occurred_at, recurring_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		in.AccountID, in.GroupID, in.Type, in.Amount, in.Category, desc, in.OccurredAt, in.RecurringID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	if err := insertSplits(ctx, tx, id, splits); err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx,
		"INSERT INTO balances (account_id, transaction_id, amount) VALUES ($1, $2, 0)",
		in.AccountID, id,
	)
	if err != nil {
		return 0, err
	}
	return id, Recompute(ctx, tx, in.AccountID)
}

// Recompute hesabın tüm bakiye anlık görüntülerini işlem tarihi sırasına göre
// yeniden hesaplar; sadece değişen satırlar güncellenir.
func Recompute(ctx context.Context, tx pgx.Tx, accountID int) error {
	_, err := tx.Exec(ctx, `
UPDATE balances b SET amount = s.running
FROM (
  SELECT id, (SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END)
              OVER (ORDER BY occurred_at, id))::bigint AS running
  FROM transactions
  WHERE account_id = $1
) s
WHERE b.transaction_id = s.id AND b.amount <> s.running`, accountID)
	return err
}

// ResolveSplits paylaşım isteğini doğrular ve payları hesaplar. Grup üyelik
// satırları FOR SHARE ile kilitlenir; böylece eşzamanlı gruptan ayrılma,
// ayrılan kişiye pay yazılmasıyla çakışamaz.
func ResolveSplits(ctx context.Context, tx pgx.Tx, groupID, payerID int, amount int64, spec *SplitSpec) ([]Split, error) {
	rows, err := tx.Query(ctx,
		"SELECT user_id FROM group_members WHERE group_id = $1 ORDER BY user_id FOR SHARE",
		groupID,
	)
	if err != nil {
		return nil, err
	}
	members, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return nil, err
	}
	isMember := make(map[int]bool, len(members))
	for _, m := range members {
		isMember[m] = true
	}
	if !isMember[payerID] {
		return nil, httpx.NotFound("grup bulunamadi")
	}

	if spec == nil {
		return EqualSplit(amount, members), nil
	}

	seen := map[int]bool{}
	checkUser := func(id int) error {
		if !isMember[id] {
			return httpx.BadRequest("paylasimdaki kullanicilar grubun uyesi olmali")
		}
		if seen[id] {
			return httpx.BadRequest("paylasimda ayni kullanici birden fazla kez var")
		}
		seen[id] = true
		return nil
	}

	switch spec.Type {
	case "equal":
		if len(spec.UserIDs) == 0 {
			return nil, httpx.BadRequest("esit paylasim icin en az bir kullanici secilmeli")
		}
		for _, id := range spec.UserIDs {
			if err := checkUser(id); err != nil {
				return nil, err
			}
		}
		return EqualSplit(amount, spec.UserIDs), nil

	case "exact":
		var total int64
		out := make([]Split, 0, len(spec.Shares))
		for _, s := range spec.Shares {
			if err := checkUser(s.UserID); err != nil {
				return nil, err
			}
			if s.Amount < 0 || s.Amount > MaxAmount {
				return nil, httpx.BadRequest("paylar negatif olamaz")
			}
			total += s.Amount
			if s.Amount > 0 {
				out = append(out, s)
			}
		}
		if total != amount {
			return nil, httpx.BadRequest("paylarin toplami islem tutarina esit olmali")
		}
		return out, nil
	}
	return nil, httpx.BadRequest("paylasim tipi 'equal' veya 'exact' olmali")
}

func insertSplits(ctx context.Context, tx pgx.Tx, txID int, splits []Split) error {
	if len(splits) == 0 {
		return nil
	}
	users := make([]int, len(splits))
	amounts := make([]int64, len(splits))
	for i, s := range splits {
		users[i], amounts[i] = s.UserID, s.Amount
	}
	_, err := tx.Exec(ctx, `
INSERT INTO transaction_splits (transaction_id, user_id, amount)
SELECT $1, u, a FROM unnest($2::int[], $3::bigint[]) AS x(u, a)`,
		txID, users, amounts,
	)
	return err
}

func loadSplits(ctx context.Context, tx pgx.Tx, txID int) ([]Split, error) {
	rows, err := tx.Query(ctx,
		"SELECT user_id, amount FROM transaction_splits WHERE transaction_id = $1 ORDER BY user_id",
		txID,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Split])
}

type current struct {
	accountID  int
	owner      int
	groupID    *int
	typ        string
	amount     int64
	category   *string
	desc       *string
	occurredAt time.Time
}

// lock işlemi ve hesabını kilitler, actorID'nin erişimini kontrol eder:
// hesabın sahibi ya da işlemin grubunun güncel üyesi olmalı.
func lock(ctx context.Context, tx pgx.Tx, actorID, txID int) (current, error) {
	var c current
	err := tx.QueryRow(ctx, `
SELECT t.account_id, a.user_id, t.group_id, t.type, t.amount, t.category, t.description, t.occurred_at
FROM transactions t
JOIN accounts a ON a.id = t.account_id
WHERE t.id = $1
FOR UPDATE`, txID,
	).Scan(&c.accountID, &c.owner, &c.groupID, &c.typ, &c.amount, &c.category, &c.desc, &c.occurredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, httpx.NotFound("islem bulunamadi")
	}
	if err != nil {
		return c, err
	}
	if c.owner == actorID {
		return c, nil
	}
	if c.groupID != nil {
		var member bool
		err = tx.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)",
			*c.groupID, actorID,
		).Scan(&member)
		if err != nil {
			return c, err
		}
		if member {
			return c, nil
		}
	}
	return c, httpx.NotFound("islem bulunamadi")
}

func checkNotSettlement(ctx context.Context, tx pgx.Tx, txID int) error {
	var linked bool
	err := tx.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM settlements WHERE $1 IN (transaction_id, counter_transaction_id))", txID,
	).Scan(&linked)
	if err != nil {
		return err
	}
	if linked {
		return httpx.Conflict("bu islem bir hesaplasmaya bagli; hesaplasma kaydini silin")
	}
	return nil
}

type Patch struct {
	Type        *string
	Amount      *int64
	Category    httpx.Optional[string]
	Description httpx.Optional[string]
	OccurredAt  *time.Time
	GroupID     httpx.Optional[int]
	Split       *SplitSpec
}

// Update işlemi düzenler. Grup işlemlerini grubun tüm üyeleri düzenleyebilir;
// işlemin grubunu sadece hesabın sahibi değiştirebilir. Tutar değişip yeni
// paylaşım verilmezse mevcut paylar oranı korunarak yeniden hesaplanır.
func Update(ctx context.Context, tx pgx.Tx, actorID, txID int, p Patch) error {
	cur, err := lock(ctx, tx, actorID, txID)
	if err != nil {
		return err
	}
	if err := checkNotSettlement(ctx, tx, txID); err != nil {
		return err
	}

	next := cur
	if p.Type != nil {
		next.typ = *p.Type
	}
	if p.Category.Set {
		next.category = p.Category.Value
	} else if ValidateCategory(next.typ, next.category) != nil {
		// Tip değişti ve eski kategori yeni tipe uymuyor: kategoriyi kaldır.
		next.category = nil
	}
	if err := ValidateCategory(next.typ, next.category); err != nil {
		return err
	}
	if p.Amount != nil {
		next.amount = *p.Amount
	}
	if p.OccurredAt != nil {
		next.occurredAt = *p.OccurredAt
	}
	if p.Description.Set {
		if next.desc, err = NormalizeDescription(p.Description.Value); err != nil {
			return err
		}
	}
	if p.GroupID.Set {
		if actorID != cur.owner {
			return httpx.Forbidden("islemin grubunu sadece islemi ekleyen degistirebilir")
		}
		next.groupID = p.GroupID.Value
	}
	if err := validate(next.typ, next.amount, next.occurredAt); err != nil {
		return err
	}

	groupChanged := !sameGroup(cur.groupID, next.groupID)
	var splits []Split
	splitsChanged := false
	switch {
	case next.groupID == nil:
		if p.Split != nil {
			return httpx.BadRequest("paylasim sadece grup islemlerinde kullanilabilir")
		}
		splitsChanged = groupChanged
	case p.Split != nil || groupChanged:
		if splits, err = ResolveSplits(ctx, tx, *next.groupID, cur.owner, next.amount, p.Split); err != nil {
			return err
		}
		splitsChanged = true
	case next.amount != cur.amount:
		old, err := loadSplits(ctx, tx, txID)
		if err != nil {
			return err
		}
		if len(old) == 0 {
			if splits, err = ResolveSplits(ctx, tx, *next.groupID, cur.owner, next.amount, nil); err != nil {
				return err
			}
		} else {
			splits = Proportional(next.amount, old)
		}
		splitsChanged = true
	}

	_, err = tx.Exec(ctx, `
UPDATE transactions
SET type = $2, amount = $3, description = $4, occurred_at = $5, group_id = $6, category = $7, updated_at = now()
WHERE id = $1`,
		txID, next.typ, next.amount, next.desc, next.occurredAt, next.groupID, next.category,
	)
	if err != nil {
		return err
	}
	if splitsChanged {
		if _, err := tx.Exec(ctx, "DELETE FROM transaction_splits WHERE transaction_id = $1", txID); err != nil {
			return err
		}
		if err := insertSplits(ctx, tx, txID, splits); err != nil {
			return err
		}
	}
	if next.typ != cur.typ || next.amount != cur.amount || !next.occurredAt.Equal(cur.occurredAt) {
		return Recompute(ctx, tx, cur.accountID)
	}
	return nil
}

// Delete erişim kurallarıyla işlemi siler.
func Delete(ctx context.Context, tx pgx.Tx, actorID, txID int) error {
	if _, err := lock(ctx, tx, actorID, txID); err != nil {
		return err
	}
	if err := checkNotSettlement(ctx, tx, txID); err != nil {
		return err
	}
	return Remove(ctx, tx, txID)
}

// Remove erişim kontrolü yapmadan işlemi, paylarını ve bakiye satırını siler,
// hesabın bakiyelerini yeniden hesaplar. Sadece iç kullanım içindir.
func Remove(ctx context.Context, tx pgx.Tx, txID int) error {
	var accountID int
	err := tx.QueryRow(ctx, `
SELECT a.id FROM transactions t JOIN accounts a ON a.id = t.account_id
WHERE t.id = $1 FOR UPDATE`, txID).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM balances WHERE transaction_id = $1", txID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM transactions WHERE id = $1", txID); err != nil {
		return err
	}
	return Recompute(ctx, tx, accountID)
}

func sameGroup(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
