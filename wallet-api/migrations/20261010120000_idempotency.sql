-- +goose Up

-- Çevrimdışı kuyruktan tekrar gönderilen istekler iki kez işlenmesin:
-- istemci her değişiklik isteğine benzersiz bir Idempotency-Key ekler,
-- sunucu ilk yanıtı saklar ve tekrarında aynısını döner. 7 gün tutulur.
CREATE TABLE "idempotency_keys" (
  "user_id" integer NOT NULL REFERENCES "users" ("id"),
  "key" varchar(64) NOT NULL,
  "method" varchar(8) NOT NULL,
  "path" varchar(200) NOT NULL,
  "status" integer NOT NULL DEFAULT 0,
  "body" bytea,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  PRIMARY KEY ("user_id", "key")
);
CREATE INDEX ON "idempotency_keys" ("created_at");
COMMENT ON COLUMN "idempotency_keys"."status" IS '0: işleniyor; diğerleri saklanan HTTP durum kodu';

-- +goose Down
DROP TABLE "idempotency_keys";
