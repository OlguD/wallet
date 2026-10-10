-- +goose Up

-- Hesaplaşmanın karşı tarafı (kaydı girmeyen üye) ödemeyi gelen kutusundan
-- kendi hesabına işler (counter_transaction_id) ya da kapatır. Böylece para iki
-- tarafın hesabına da tek kayıtla yansır, iki kez girilmez.
-- none: eski kayıtlar; pending: karşı tarafın onayı bekleniyor.
ALTER TABLE "settlements" ADD COLUMN "counter_transaction_id" integer UNIQUE REFERENCES "transactions" ("id");
ALTER TABLE "settlements" ADD COLUMN "counter_status" varchar(16) NOT NULL DEFAULT 'none'
  CHECK ("counter_status" IN ('none', 'pending', 'booked', 'dismissed'));
CREATE INDEX ON "settlements" ("from_user_id") WHERE "counter_status" = 'pending';
CREATE INDEX ON "settlements" ("to_user_id") WHERE "counter_status" = 'pending';

-- +goose Down
ALTER TABLE "settlements" DROP COLUMN "counter_status";
ALTER TABLE "settlements" DROP COLUMN "counter_transaction_id";
