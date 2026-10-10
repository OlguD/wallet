-- +goose Up

-- Borca sayılmayan ödeme (hediye, harçlık): hesaplara işlenir ama grup
-- borç/alacak hesabını etkilemez.
ALTER TABLE "settlements" ADD COLUMN "affects_balance" boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE "settlements" DROP COLUMN "affects_balance";
