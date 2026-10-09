# Sunucuya kurulum

Bir Linux sunucu (Docker + Docker Compose), Cloudflare'de yönetilen bir alan adı yeterli.
Üç servis çalışır: `db` (PostgreSQL, dışarı kapalı), `api` (Go), `web` (Caddy: arayüz + `/api` yönlendirme + HTTPS).

## 1. Cloudflare

1. **DNS:** alan adı için `A` kaydı → sunucunun IP'si. Proxy (turuncu bulut) açık olabilir.
2. **SSL/TLS → Overview:** mod **Full (strict)**.
3. **API token:** My Profile → API Tokens → *Create Token* → "Edit zone DNS" şablonu → sadece bu alan adı.
   Caddy bu token ile Let's Encrypt sertifikasını DNS doğrulamasıyla alır (proxy açıkken de çalışır).

## 2. Sunucu

```sh
git clone https://github.com/OlguD/wallet.git && cd wallet
cp .env.prod.example .env.prod
# .env.prod içini doldur: DOMAIN, ACME_EMAIL, CLOUDFLARE_API_TOKEN, POSTGRES_PASSWORD (openssl rand -base64 24)
```

Bildirim anahtarları (bir kez):

```sh
docker compose -f docker-compose.prod.yml --env-file .env.prod build api
docker compose -f docker-compose.prod.yml --env-file .env.prod run --rm --no-deps api gen-vapid
# çıkan VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY satırlarını .env.prod'a ekle
```

Başlat:

```sh
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
docker compose -f docker-compose.prod.yml logs -f web api
```

Migration'lar API açılırken otomatik uygulanır (`MIGRATE_ON_START=true`). Sağlık: `https://<alan-adı>/api/health`.

Sunucunun 80 ve 443 (TCP+UDP) portları açık olmalı.

## Güncelleme

```sh
git pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

## Notlar

- Oturum cookie'si `Secure` (HTTPS zorunlu). Dekontlar `receipts` volume'ünde, veritabanı `db_data` volume'ünde durur.
- `.env.prod` git'e girmez; sunucuda sakla.
- iPhone Kestirmesi adresi: `https://<alan-adı>/api/inbox` (Ayarlar → "iPhone'dan dekont gönder").
- Android paylaş menüsü ve bildirimler HTTPS üzerinde, uygulama ana ekrana eklendikten sonra çalışır.
- Yerel geliştirme değişmedi: `docker compose up -d` (sadece db), `go run ./cmd/server`, `npm run dev`.
