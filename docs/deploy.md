# Sunucuya kurulum

Bir Linux sunucu (Docker + Docker Compose) ve Cloudflare'de yönetilen bir alan adı yeterli.
Dört servis çalışır:

- `traefik` — HTTPS (Let's Encrypt, HTTP-01) ve yönlendirme: `/api/*` → `api`, diğer her şey → `web`
- `web` — Caddy, derlenmiş arayüzü içeride (HTTP) sunar
- `api` — Go sunucusu; açılışta migration'ları uygular
- `db` — PostgreSQL; sadece iç ağda, internete ve dışarıya kapalı

## 1. Cloudflare

1. **DNS:** alan adı için `A` kaydı → sunucunun IP'si. Proxy (turuncu bulut) açık olabilir.
2. **SSL/TLS → Overview:** mod **Full (strict)**.
3. Sertifika HTTP-01 ile alınır: Let's Encrypt `http://<alan-adı>/.well-known/acme-challenge/...` adresine gelir.
   İlk sertifika alınana kadar **SSL/TLS → Edge Certificates → Always Use HTTPS** kapalı olmalı (sonra açılabilir;
   Traefik yenilemeyi de aynı yolla yapar, kapalı bırakmak en güvenlisi — HTTP→HTTPS yönlendirmesini zaten Traefik yapar).

## 2. Sunucu

80 ve 443 portları açık olmalı (bulut sağlayıcının güvenlik kuralları + sunucudaki iptables).

```sh
git clone https://github.com/OlguD/wallet.git && cd wallet
cp .env.prod.example .env.prod
# .env.prod: DOMAIN, POSTGRES_PASSWORD (openssl rand -hex 24), VAPID_SUBJECT=https://<alan-adı>
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
docker compose -f docker-compose.prod.yml logs -f traefik api
```

Sağlık: `https://<alan-adı>/api/health`.

## Güncelleme

```sh
cd wallet && git pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

## Notlar

- Oturum cookie'si `Secure` (HTTPS zorunlu). Dekontlar `receipts`, veritabanı `db_data`, sertifikalar `letsencrypt` volume'ünde.
- `.env.prod` git'e girmez; sadece sunucuda durur.
- iPhone Kestirmesi adresi: `https://<alan-adı>/api/inbox` (Ayarlar → "iPhone'dan dekont gönder").
- Yerel geliştirme değişmedi: `docker compose up -d` (sadece db), `go run ./cmd/server`, `npm run dev`.
