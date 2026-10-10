-- +goose Up

-- Kredi kartı takibi: limit ve son ödeme günü (sadece kind = 'card').
-- Kartın borcu ayrıca tutulmaz: negatif bakiye borçtur, kalan limit = limit - borç.
ALTER TABLE "accounts" ADD COLUMN "credit_limit" bigint CHECK ("credit_limit" > 0);
ALTER TABLE "accounts" ADD COLUMN "due_day" smallint CHECK ("due_day" BETWEEN 1 AND 31);

-- +goose Down
ALTER TABLE "accounts" DROP COLUMN "due_day";
ALTER TABLE "accounts" DROP COLUMN "credit_limit";
