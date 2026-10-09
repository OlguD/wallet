-- +goose Up

-- İşlem kategorisi (arayüzdeki market, fatura, maaş...). Geçerli değerler uygulamada denetlenir.
ALTER TABLE "transactions" ADD COLUMN "category" varchar(32);
ALTER TABLE "recurring_transactions" ADD COLUMN "category" varchar(32);
CREATE INDEX ON "transactions" ("category");

-- Hesap türü: banka, nakit, kredi kartı, birikim.
ALTER TABLE "accounts" ADD COLUMN "kind" varchar(16) NOT NULL DEFAULT 'bank'
  CHECK ("kind" IN ('bank', 'cash', 'card', 'savings'));

-- +goose Down
ALTER TABLE "accounts" DROP COLUMN "kind";
DROP INDEX IF EXISTS "transactions_category_idx";
ALTER TABLE "recurring_transactions" DROP COLUMN "category";
ALTER TABLE "transactions" DROP COLUMN "category";
