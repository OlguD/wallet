# Wallet API

Tüm gövdeler JSON. Hatalar her zaman `{"error": "mesaj"}` biçiminde döner.
Para tutarları **kuruş** cinsinden tam sayıdır (`12550` = 125,50 ₺). Tarihler: zaman damgası RFC 3339,
takvim günü `YYYY-AA-GG`. Listeler `?limit=50&offset=0` ile sayfalanır (limit en fazla 200).

Ortam değişkenleri: `DATABASE_URL` (zorunlu), `ADDR` (`:8080`), `APP_TIMEZONE` (`Europe/Istanbul`),
`COOKIE_SECURE` (HTTPS arkasında `true`).

## Kimlik doğrulama

Oturum, DB'de tutulan opak bir token'dır (90 gün, kullanıldıkça uzar).
Login/register token'ı hem gövdede döner hem `session` adlı HttpOnly cookie olarak yazar.
İstemci cookie'ye güvenebilir (PWA için önerilen) ya da `Authorization: Bearer <token>` gönderebilir.

| Metot | Yol | Gövde | Cevap |
|---|---|---|---|
| POST | `/auth/register` | `{username, password}` | 201 `{user, token}` |
| POST | `/auth/login` | `{username, password}` | `{user, token}` |
| POST | `/auth/logout` | – | 204 |
| GET | `/me` | – | `{id, username, created_at}` |

Kullanıcı adı küçük harfe çevrilir; 3-32 karakter, `a-z 0-9 _ . -`. Şifre 8-72 karakter.

## Hesaplar

| Metot | Yol | Gövde | Cevap |
|---|---|---|---|
| GET | `/accounts` | – | `[Account]` |
| POST | `/accounts` | `{name, currency?, kind?}` (varsayılan `TRY`, `bank`) | 201 `Account` |
| GET | `/accounts/{id}` | – | `Account` |
| PATCH | `/accounts/{id}` | `{name?, kind?}` | `Account` |
| DELETE | `/accounts/{id}` | – | 204; işlemi/kuralı/hedefi varsa 409 |
| GET | `/accounts/{id}/transactions` | – | `[Transaction]` (yeniden eskiye) |

`Account`: `{id, name, currency, kind, balance, created_at}` — `kind`: `bank` | `cash` | `card` | `savings`.

## İşlemler

| Metot | Yol | Gövde | Cevap |
|---|---|---|---|
| GET | `/transactions?from=&to=&account_id=&group_id=&category=` | – | `[Transaction]` (tüm hesaplarım) |
| POST | `/transactions` | `{account_id, type, amount, category?, group_id?, split?, description?, occurred_at?}` | 201 `Transaction` |
| GET | `/transactions/{id}` | – | `Transaction` |
| PATCH | `/transactions/{id}` | gönderilen alanlar: `type, amount, category, description, occurred_at, group_id, split` | `Transaction` |
| DELETE | `/transactions/{id}` | – | 204 |
| GET | `/summary?from=&to=` | – | `{from, to, totals, accounts, categories}` (aşağıda) |

- `type`: `income` | `expense`; `amount` > 0. `occurred_at` verilmezse şimdi; gelecek tarih kabul edilmez.
- **Bakiye** işlem tarihi sırasına göre hesaplanır. Geçmiş tarihli ekleme, düzenleme ve silmede
  sonraki tüm bakiyeler aynı DB transaction'ında yeniden hesaplanır.
- **Grup işlemi**: `group_id` verilirse para ekleyenin hesabından düşer ve üyelere paylaştırılır:
  - `split` yok → grubun tüm güncel üyeleri arasında eşit
  - `{"type":"equal","user_ids":[1,2]}` → seçilenler arasında eşit
  - `{"type":"exact","shares":[{"user_id":1,"amount":7000},{"user_id":2,"amount":3000}]}` → toplam = tutar
  - Artan kuruşlar küçük id'li üyelere gider.
- **Düzenleme/silme**: hesabın sahibi veya işlemin grubunun güncel üyeleri yapabilir.
  Grubu (`group_id`, `null` = kişisel) sadece sahibi değiştirebilir. Tutar değişip `split` verilmezse
  mevcut paylar oranı korunarak yeniden hesaplanır. Hesaplaşmaya bağlı işlem düzenlenemez (409).
- **Kategori**: gider `groceries, bills, transport, food, rent, subscription, gift, health, shopping, other`;
  gelir `salary, extra, gift, other`. Tip değişip kategori uymazsa kategori kaldırılır.
- `/summary` varsayılan olarak içinde bulunulan ayı döner; `to` hariçtir.
  `totals: [{currency, income, expense, net}]`, `accounts: [{account_id, income, expense}]`,
  `categories: [{currency, type, category, amount}]`.

`Transaction`:
```
{id, account_id, account_name, group_id, type, amount, currency, category, description, occurred_at, created_at, updated_at,
 recurring_id, balance_after?, user_id, username, splits?: [{user_id, username, amount}]}
```
`user_id/username` hesabın sahibi (ödeyen). Başka üyenin işleminde `balance_after` gösterilmez.

## Gruplar (ev yönetimi)

| Metot | Yol | Gövde | Cevap |
|---|---|---|---|
| GET | `/groups` | – | `[Group]` |
| POST | `/groups` | `{name}` | 201 `Group` (oluşturan otomatik üye) |
| GET | `/groups/{id}` | – | `Group` + `members` |
| PATCH | `/groups/{id}` | `{name}` | `Group` |
| POST | `/groups/{id}/members` | `{username}` | `Group` (herhangi bir üye ekleyebilir) |
| POST | `/groups/{id}/leave` | – | 204; açık borç/alacak varsa 409 |
| GET | `/groups/{id}/transactions` | – | `[Transaction]` |
| GET | `/groups/{id}/balances` | – | kim kime borçlu (aşağıda) |
| GET | `/groups/{id}/summary?from=&to=` | – | dönem giderleri |
| GET | `/groups/{id}/settlements` | – | `[Settlement]` |
| POST | `/groups/{id}/settlements` | `{from_user_id, to_user_id, amount, currency?, account_id?, note?, occurred_at?}` | 201 `Settlement` |
| DELETE | `/groups/{id}/settlements/{sid}` | – | 204 |
| GET | `/groups/{id}/recurring` | – | grubun tekrarlayan giderleri |

Üye olunmayan grup için her zaman 404 döner.

**Borç durumu** (`/balances`):
```
{"balances": [{currency, user_id, username, net}],
 "suggestions": [{currency, from_user_id, from_username, to_user_id, to_username, amount}]}
```
`net` > 0 alacaklı, < 0 borçlu. `suggestions` borçları en az ödemeyle kapatma önerisi.

**Hesaplaşma**: ödemeyi ödeyen ya da alan taraf girer. `account_id` verilirse ödeme girenin
o hesabına da işlenir (ödeyense gider, alansa gelir); hesaplaşma silinince o işlem de silinir.
Taraflar ve kaydı giren silebilir.

**Grup özeti** (`/summary`): `{from, to, currencies: [{currency, total_expense, members: [{user_id, username, paid, share}]}]}`
— `paid` üyenin ödediği, `share` payına düşen gider.

## Tekrarlayan işlemler (kira, aidat, fatura)

| Metot | Yol | Gövde | Cevap |
|---|---|---|---|
| GET | `/recurring` | – | `[Rule]` (kendi kurallarım) |
| POST | `/recurring` | `{account_id, type, amount, frequency, interval?, start_on?, end_on?, group_id?, split?, category?, description?}` | 201 `Rule` |
| GET | `/recurring/{id}` | – | `Rule` |
| PATCH | `/recurring/{id}` | `{amount?, category?, description?, split?, active?, end_on?}` | `Rule` |
| DELETE | `/recurring/{id}` | – | 204 (üretilmiş işlemler kalır) |

- `frequency`: `weekly` | `monthly` | `yearly`; `interval` 1-12 (örn. 2 haftada bir).
- Gün `start_on`'dan alınır; kısa aylarda ay sonuna çekilir (31 → 28/29 Şub → 31 Mar).
- Sunucu her 15 dakikada ve açılışta vadesi gelenleri işler; kaçırılan dönemleri de oluşturur.
  `start_on` bugün/geçmişse (en fazla 1 yıl) işlemler hemen oluşur.
- Grup kuralında paylaşımdaki biri gruptan ayrılmışsa o dönem eşit bölünür; sahibi ayrılmışsa kural durur.
- Duraklatılan kural (`active: false`) yeniden açılınca kaçırılan dönemler atlanır.
- Sadece sahibi düzenleyebilir; grup üyeleri `/groups/{id}/recurring` ile görebilir.

`Rule`: `{id, account_id, account_name, currency, group_id, user_id, username, type, amount, description, split, frequency, interval, next_run_on, end_on, active, created_at}`

## Birikim hedefleri

| Metot | Yol | Gövde | Cevap |
|---|---|---|---|
| GET | `/goals` | – | `[Goal]` |
| POST | `/goals` | `{account_id, name, icon?, target_amount, target_date?, initial_amount?}` | 201 `Goal` |
| GET | `/goals/{id}` | – | `Goal` |
| PATCH | `/goals/{id}` | `{name?, icon?, target_amount?, target_date?}` (`null` tarihi kaldırır) | `Goal` |
| DELETE | `/goals/{id}` | – | 204 |
| GET | `/goals/{id}/contributions` | – | `[{id, amount, note, occurred_at}]` |
| POST | `/goals/{id}/contributions` | `{amount, note?}` (pozitif: ekle, negatif: çek; birikimden fazla çekilemez) | 201 `Goal` |
| DELETE | `/goals/{id}/contributions/{cid}` | – | `Goal` |

`Goal`: `{id, account_id, account_name, currency, name, icon, target_amount, target_date, current, remaining, progress_pct, monthly_needed, account_balance, created_at}`

`current` hedefe ayrılan katkıların toplamıdır (hesap bakiyesini değiştirmez). `icon`: `target, car, home, plane, phone, gift, bag, heart, school`.
— ilerleme bağlı hesabın bakiyesinden hesaplanır; `monthly_needed` hedef tarihe kadar aylık gereken tutar.

## Döviz kurları

| Yöntem | Yol | Gövde | Yanıt |
|---|---|---|---|
| GET | `/rates` | – | `Rates` |

Kaynak: Sun Döviz (`https://online.sundoviz.com/services/apirates.php?app=online`). Sunucu 2 dk önbellekler; kaynak
erişilemezse son başarılı veri `stale: true` ile döner (hiç veri yoksa 500).

`Rates`: `{source, updated_at, fetched_at, stale, cross_mode, online: Channel, desk: Channel}` —
`online` EFT/havale, `desk` gişe kurları. `Channel`: `{rates: {USD|EUR|GBP: {buy, sell}}, cross: {"EUR/USD": {buy, sell}, ...}}`;
`rates` 1 birim dövizin TL karşılığıdır (`buy` = alış, `sell` = satış).

Çevirme (istemci, sundoviz.com hesaplayıcısıyla aynı): döviz → TL alış, TL → döviz satış; döviz → döviz `cross_mode`
ise çapraz kur, değilse `alış(from) / satış(to)`. Toplam bakiyede dövizli hesaplar alış kuruyla TL'ye çevrilir.


## Dekontlar ve karşı taraf

Dekont dosyası sunucuda `RECEIPTS_DIR` (varsayılan `data/receipts`) altında saklanır. Metin istemcide çıkarılır
(PDF: pdf.js, resim: tesseract.js OCR) ve sunucudaki kural tabanlı ayrıştırıcıya (`internal/receipts/parse.go`)
verilir; hiçbir veri üçüncü bir servise gitmez. En fazla 10 MB; PDF, JPG, PNG, WEBP, HEIC.

| Yöntem | Yol | Gövde | Yanıt |
|---|---|---|---|
| POST | `/receipts` | multipart: `file`, `text?` (`?source=share` Android paylaş) | 201 `Receipt` (aynı dosya: 200, `duplicate: true`) |
| GET | `/receipts?status=pending\|used\|dismissed` | – | `[Receipt]` |
| GET | `/receipts/{id}` | – | `Receipt` |
| GET | `/receipts/{id}/file` | – | dosya (sahibi veya bağlı grup işleminin üyeleri) |
| POST | `/receipts/{id}/parse` | `{text}` | `Receipt` |
| PATCH | `/receipts/{id}` | `{status: "pending"\|"dismissed"}` | `Receipt` |
| DELETE | `/receipts/{id}` | – | 204 |
| POST | `/inbox` | **Gelen kutusu anahtarı** (`Authorization: Bearer`); multipart `file` veya ham gövde (`?name=`) | 201 `{ok, message}` |
| POST | `/me/inbox-token` | – | 201 `{token}` (bir kez gösterilir; eskisi geçersiz olur) |
| DELETE | `/me/inbox-token` | – | 204 |
| GET | `/reports/counterparties` | `?type=expense\|income&from=&to=` | `[{key, name, iban, bank, currency, count, total, last_at}]` |

`Receipt`: `{id, mime_type, size_bytes, original_name, source (app|share|inbox), status, transaction_id, parsed, has_text, created_at}`.
`parsed`: `{type, amount, currency, date, time, counterparty_name, counterparty_iban, counterparty_bank, sender_name, sender_iban,
recipient_name, recipient_iban, bank, description, reference, transfer_type}` (bulunamayanlar yok).

İşlemler artık `counterparty_name`, `counterparty_iban`, `counterparty_bank`, `receipt_id` alanlarını döner;
`POST/PATCH /transactions` bu alanları ve `receipt_id` (dekontu bağlar) kabul eder. Banka boşsa IBAN'dan bulunur.
İşlem silinince bağlı dekont gelen kutusuna (`pending`) döner. `GET /transactions?counterparty=<key>` rapordaki kişiye göre süzer.

Gelen kutusu anahtarı sadece `POST /inbox` için geçerlidir (oturum yerine geçmez).

## Tanıtım turları

`GET /me` → `seen_tours: [..]`, `has_inbox_token`. `POST /me/tours {ids: [..]}` görülenleri ekler.
Turlar `wallet-web/src/lib/tours.js` içinde; her yeni özellik yeni bir tur kimliğiyle eklenir.

## Güvenlik ve hesap

- **Giriş kilidi:** aynı kullanıcı adıyla art arda 5 hatalı şifrede 5 dakika kilit (`429`, `Retry-After`, `{error, retry_after}`).
  Hatalı denemede `401 {error, attempts_left}`. Var olmayan kullanıcı adları da aynı şekilde sayılır. Şifre doğrulayan
  diğer uçlar (`/me/password`, `DELETE /me`) da aynı kurala tabidir.
- `POST /me/password` `{current_password, new_password}` → 204; bu cihaz dışındaki oturumlar kapanır.
- `DELETE /me` `{password}` → 204. Hesap kapatılır (`users.deleted_at`); **veriler silinmez** (işlemler, hesaplar,
  grup geçmişi, borçlar kalır). Oturumlar, gelen kutusu anahtarı ve bildirim abonelikleri kaldırılır, bekleyen davetler
  iptal edilir, düzenli ödemeler durdurulur. Kapalı hesapla giriş `403 "bu hesap silinmis"`; kullanıcı adı tekrar alınamaz.

## Tekrar koruması (Idempotency-Key)

Oturumlu tüm değişiklik isteklerine `Idempotency-Key: <8-64 karakter [A-Za-z0-9_-]>` eklenebilir. Aynı anahtarla gelen
tekrar istek işlenmez; ilk yanıt `Idempotent-Replay: true` ile döner (5xx yanıtlar saklanmaz). Anahtar başka bir
yöntem/yol için kullanılırsa `422`; ilk istek hâlâ işleniyorsa `409`. Anahtarlar 7 gün tutulur. Arayüzün çevrimdışı
kuyruğu her isteğe anahtar ekler.

## Grup davetleri

Gruba üye doğrudan eklenmez; davet edilen kabul edince üye olur.

| Yöntem | Yol | Gövde | Yanıt |
|---|---|---|---|
| POST | `/groups/{id}/invites` | `{username}` | 201 `Group` (`invites` dahil); eski `POST /groups/{id}/members` da davet gönderir |
| DELETE | `/groups/{id}/invites/{iid}` | – | `Group` (grubun herhangi bir üyesi geri çekebilir) |
| GET | `/invites` | – | `[Invite]` (bana gelen bekleyenler) |
| POST | `/invites/{id}/accept` | – | `Group` |
| POST | `/invites/{id}/decline` | – | 204 |

`Invite`: `{id, group_id, group_name, inviter_id, inviter_name, invitee_id, invitee_name, created_at}`.
`Group.members[].deleted`: hesabı kapalı üye.

## Transfer, bütçe, arama, dışa aktarma

| Yöntem | Yol | Gövde | Yanıt |
|---|---|---|---|
| POST | `/transfers` | `{from_account_id, to_account_id, amount, to_amount?, description?, occurred_at?}` | 201 kaynak bacak `Transaction` |
| GET | `/budgets?month=YYYY-MM` | – | `[{id, category, currency, amount, spent, remaining, pct}]` |
| POST | `/budgets` | `{category, amount, currency?}` | 201 |
| PATCH | `/budgets/{id}` | `{amount}` | `Budget` |
| DELETE | `/budgets/{id}` | – | 204 |
| GET | `/transactions?q=` | – | açıklama, karşı taraf, hesap adında arar; sayıysa tutar da eşleşir |
| GET | `/export/transactions.csv?from=&to=` | – | CSV (UTF-8 BOM, `;`, `,` ondalık — Excel TR) |

Transfer iki bağlı işlem oluşturur (gider + gelir; `transfer_peer_id`, `transfer_account_id`, `transfer_account_name`).
Farklı para birimlerinde `to_amount` zorunludur. Transferler `summary` toplamlarına ve bütçelere sayılmaz; tutarı
değiştirilemez (`409`), açıklama/tarih iki bacakta birlikte değişir, silinince iki bacak birden silinir.
Bütçe harcaması: o ay, o para biriminde, o kategorideki (transfer olmayan) giderler. %80 ve %100'de bildirim (ayda bir kez).

## Bildirimler

| Yöntem | Yol | Gövde | Yanıt |
|---|---|---|---|
| GET | `/notifications` | – | `{items: [{id, kind, title, body, url, read_at, created_at}], unread}` (son 50) |
| POST | `/notifications/read` | `{ids: []}` (boş: hepsi) | 204 |
| GET | `/push/key` | – | `{enabled, public_key}` |
| POST | `/push/subscribe` | tarayıcının `PushSubscription.toJSON()` çıktısı | 204 |
| POST | `/push/unsubscribe` | `{endpoint}` | 204 |
| POST | `/push/test` | – | 204 |

Bildirim türleri: `invite`, `invite_accepted`, `group_expense` (diğer üyelere, kendi paylarıyla), `recurring` (işlenen
düzenli ödeme), `budget80:YYYY-MM`, `budget:YYYY-MM`. Web Push için `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`,
`VAPID_SUBJECT` gerekir (`wallet-api gen-vapid` üretir); yoksa sadece uygulama içi bildirim yazılır.
