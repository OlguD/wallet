-- +goose Up

-- Face ID / Touch ID ile giriş (WebAuthn passkey). Cihazdaki gizli anahtar
-- iCloud Anahtar Zinciri'nde kalır; burada sadece açık anahtar ve sayaç durur.
-- webauthn_id: kullanıcının rastgele passkey kimliği (user handle).
ALTER TABLE "users" ADD COLUMN "webauthn_id" bytea UNIQUE;

CREATE TABLE "passkeys" (
  "id" bytea PRIMARY KEY, -- credential ID
  "user_id" integer NOT NULL REFERENCES "users" ("id"),
  "name" varchar(64) NOT NULL,
  "credential" jsonb NOT NULL, -- webauthn.Credential (açık anahtar, bayraklar, sayaç)
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_used_at" timestamptz
);
CREATE INDEX ON "passkeys" ("user_id");

-- Kayıt/giriş törenlerinin tek kullanımlık meydan okumaları (5 dk).
CREATE TABLE "webauthn_challenges" (
  "id" varchar(64) PRIMARY KEY,
  "kind" varchar(16) NOT NULL, -- register | login
  "user_id" integer REFERENCES "users" ("id"),
  "session" jsonb NOT NULL,
  "expires_at" timestamptz NOT NULL
);

-- +goose Down
DROP TABLE "webauthn_challenges";
DROP TABLE "passkeys";
ALTER TABLE "users" DROP COLUMN "webauthn_id";
