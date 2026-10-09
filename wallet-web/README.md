# wallet-web

Svelte 5 + Vite PWA. iPhone'da Safari'den "Ana Ekrana Ekle" ile uygulama gibi çalışır.

## Çalıştırma

```fish
# 1) kökte veritabanı
docker compose up -d
# 2) API (wallet-api içinde)
go run ./cmd/server
# 3) web (wallet-web içinde)
npm install
npm run dev          # geliştirme, http://localhost:5173 ve LAN adresi
npm run build; and npm run preview -- --port 5173   # production derlemesi
```

`/api/*` istekleri Vite tarafından Go sunucusuna (`API_URL`, varsayılan `http://localhost:8080`)
aktarılır; tarayıcı için tek origin olduğundan oturum cookie'si sorunsuz çalışır.
Prod'da da aynı düzen kullanılmalı: ters vekil (Caddy/nginx) `/` → `dist/`, `/api` → Go.

## Yapı

- `src/lib/` — API istemcisi, para (kuruş) biçimlendirme, i18n (TR/EN), tema/dil tercihleri, yönlendirici, durum
- `src/components/` — ortak bileşenler; `AddSheet` hızlı ekleme/düzenleme (üç tema yerleşimi)
- `src/pages/` — Özet, İşlemler, Hesaplar, Gruplar, Ayarlar, Giriş
- `src/app.css` — tema değişkenleri: `data-theme="a|b|c"` (Gece, Defter, Cepler)
- `public/` — manifest, service worker, fontlar (yerel), ikonlar (`npm run icons` ile üretilir)

## Notlar

- Service worker yalnızca HTTPS'te kaydolur. Yerel ağda (http) uygulama ana ekrandan
  tam ekran çalışır ama çevrimdışı önbellek devreye girmez.
- Ana ekran ikonu seçili temaya göre değişir; ikon, eklendiği andaki temanınkidir.
