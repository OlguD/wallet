import { prefs } from './prefs.svelte.js'

const tr = {
  'app.name': 'cüzdan',
  // gezinme
  'nav.home': 'Özet', 'nav.tx': 'İşlemler', 'nav.tx.b': 'Defter', 'nav.accounts': 'Hesaplar', 'nav.accounts.c': 'Cepler',
  'nav.groups': 'Gruplar', 'nav.add': 'İşlem ekle', 'nav.profile': 'Profil', 'nav.back': 'Geri',
  // ortak
  'common.save': 'Kaydet', 'common.cancel': 'Vazgeç', 'common.close': 'Kapat', 'common.delete': 'Sil', 'common.edit': 'Düzenle',
  'common.all': 'Tümü', 'common.none': 'Yok', 'common.more': 'Daha fazla', 'common.loading': 'Yükleniyor…',
  'common.retry': 'Tekrar dene', 'common.done': 'Tamam', 'common.add': 'Ekle', 'common.create': 'Oluştur',
  'common.income': 'Gelir', 'common.expense': 'Gider', 'common.net': 'Net', 'common.balance': 'Bakiye', 'common.amount': 'Tutar',
  'common.today': 'Bugün', 'common.yesterday': 'Dün', 'common.empty': 'Henüz kayıt yok',
  'common.you': 'sen', 'common.confirm_delete': 'Silinsin mi? Bu işlem geri alınamaz.',
  // giriş
  'auth.login': 'Giriş yap', 'auth.register': 'Hesap oluştur', 'auth.username': 'Kullanıcı adı', 'auth.password': 'Şifre',
  'auth.tagline': 'Kişisel ve ortak harcamaların, tek cepte.', 'auth.logout': 'Çıkış yap',
  'auth.username_hint': '3-32 karakter: harf, rakam, _ . -', 'auth.password_hint': 'En az 8 karakter',
  // özet
  'home.total': 'Toplam bakiye', 'home.accounts': 'Hesaplar', 'home.recent': 'Son işlemler', 'home.recent.b': 'Son kayıtlar',
  'home.open_ledger': 'Defteri aç', 'home.spent_ratio': 'Gelirin %{p}\'si harcandı', 'home.no_income': 'Bu ay henüz gelir yok',
  'home.hello': 'Merhaba {name}', 'home.pockets_total': '{n} cepte toplam', 'home.month_income': '{m} geliri',
  'home.month_expense': '{m} gideri', 'home.this_month': 'Bu ay {v}', 'home.no_accounts': 'İlk hesabını ekleyerek başla',
  'home.add_account': 'Hesap ekle', 'home.other_currencies': 'Diğer: {v}',
  'ledger.date': 'Tarih', 'ledger.desc': 'Açıklama', 'ledger.amount': 'Tutar', 'ledger.balance': 'Bakiye',
  // işlem ekleme
  'add.title': 'Yeni işlem', 'add.title.b': 'Yeni kayıt', 'add.edit_title': 'İşlemi düzenle', 'add.save.b': 'Deftere işle',
  'add.account': 'Hesap', 'add.group': 'Grup', 'add.category': 'Kategori', 'add.note': 'Not', 'add.note_ph': 'Açıklama ekle',
  'add.date': 'Tarih', 'add.repeat': 'Tekrar', 'add.split': 'Paylaşım', 'add.after': '{acc} sonrası',
  'add.need_amount': 'Bir tutar gir', 'add.need_account': 'Önce bir hesap ekle', 'add.saved': 'Kaydedildi',
  'add.deleted': 'İşlem silindi', 'add.repeat_saved': 'Düzenli işlem oluşturuldu',
  'add.locked': 'Bu işlem {name} adına; hesabı değiştirilemez',
  'repeat.none': 'Tekrarlama yok', 'repeat.weekly': 'Her hafta', 'repeat.monthly': 'Her ay', 'repeat.yearly': 'Her yıl',
  'split.equal_all': 'Herkese eşit', 'split.equal_n': '{n} kişiye eşit', 'split.choose': 'Kişileri seç',
  'split.exact': 'Tutar gir', 'split.remaining': 'Kalan: {v}', 'split.sum_error': 'Paylar toplamı tutara eşit olmalı',
  'split.title': 'Nasıl bölünsün?', 'split.mode_equal': 'Eşit', 'split.mode_exact': 'Tutarla',
  // kategoriler
  'cat.groceries': 'Market', 'cat.bills': 'Fatura', 'cat.transport': 'Ulaşım', 'cat.food': 'Yeme-içme', 'cat.rent': 'Kira',
  'cat.subscription': 'Abonelik', 'cat.gift': 'Hediye', 'cat.other': 'Diğer', 'cat.salary': 'Maaş', 'cat.extra': 'Ek gelir',
  'cat.health': 'Sağlık', 'cat.shopping': 'Alışveriş', 'cat.none': 'Kategorisiz', 'cat.settlement': 'Hesaplaşma',
  // işlemler
  'tx.title': 'İşlemler', 'tx.title.b': 'Defter', 'tx.empty': 'Bu ay işlem yok', 'tx.filter_all': 'Tümü',
  'tx.paid_by': '{name} ödedi', 'tx.your_share': 'Payın {v}', 'tx.recurring': 'Düzenli',
  // hesaplar
  'acc.title': 'Hesaplar', 'acc.title.c': 'Cepler', 'acc.new': 'Yeni hesap', 'acc.name': 'Hesap adı', 'acc.kind': 'Tür',
  'acc.currency': 'Para birimi', 'acc.kind.bank': 'Banka', 'acc.kind.cash': 'Nakit', 'acc.kind.card': 'Kart',
  'acc.kind.savings': 'Birikim', 'acc.rename': 'Hesabı düzenle', 'acc.delete_blocked': 'İşlemi olan hesap silinemez',
  'acc.name_ph': 'ör. Ziraat Vadesiz', 'acc.month_in_out': 'Bu ay',
  'card.limit': 'Kart limiti', 'card.opening_debt': 'Şu anki borcu', 'card.opening_hint': 'Kartın bugünkü toplam borcu. Gider toplamlarına sayılmaz.',
  'card.due_day': 'Son ödeme günü (ayın kaçı)', 'card.due_ph': 'ör. 15', 'card.debt': 'Güncel borç', 'card.available': 'Kalan limit',
  'card.available_short': '{v} limit kaldı', 'card.limit_of': 'limit {v}', 'card.no_limit': 'Limit girilmemiş; düzenleyerek ekleyebilirsin.',
  'card.due': 'Son ödeme {d}', 'card.days_left': '{n} gün kaldı', 'card.due_today': 'bugün', 'card.pay': 'Borç öde', 'card.add_spend': 'Harcama ekle',
  'cat.opening': 'Açılış borcu',
  // hedefler
  'goal.title': 'Birikim hedefleri', 'goal.new': 'Yeni hedef', 'goal.name': 'Hedef adı', 'goal.target': 'Hedef tutar',
  'goal.date': 'Hedef tarih', 'goal.no_date': 'Tarih yok', 'goal.monthly': 'Ayda {v} biriktir', 'goal.done': 'Hedefe ulaşıldı 🎉',
  'goal.left': '{v} kaldı', 'goal.empty': 'Araba, tatil, yeni telefon… Bir hedef koy, biriktirdikçe ekle.',
  'goal.name_ph': 'ör. Araba', 'goal.account': 'Para hangi hesapta duruyor?', 'goal.icon': 'Simge',
  'goal.initial': 'Şu ana kadar biriktirdiğin (opsiyonel)', 'goal.saved': '{v} birikti', 'goal.of': '{v} hedefin',
  'goal.deposit': 'Para ekle', 'goal.withdraw': 'Para çek', 'goal.history': 'Hareketler', 'goal.no_history': 'Henüz hareket yok.',
  'goal.note_ph': 'Not (opsiyonel)', 'goal.added': 'Hedefe {v} eklendi', 'goal.taken': 'Hedeften {v} çekildi',
  'goal.too_much': 'Hedefte bu kadar birikim yok', 'goal.days_left': '{n} gün kaldı', 'goal.overdue': 'Hedef tarihi geçti',
  'goal.account_note': '{acc} bakiyesi: {v}', 'goal.currency': 'Hangi para biriminde?',
  'goal.no_account': '{c} hesabın yok. Hedefin parası bir {c} hesapta durmalı.', 'goal.create_account': '{c} birikim hesabı aç',
  'goal.auto_account': 'Birikim {c}', 'goal.monthly_hint': 'Hedef tarihe yetişmek için ayda {v}',
  // düzenli
  'rec.title': 'Düzenli ödemeler', 'rec.empty': 'Kira, aidat, fatura, abonelik… Bir kez ekle, her ay otomatik işlensin.',
  'rec.next': 'Sıradaki: {d}', 'rec.paused': 'Duraklatıldı', 'rec.pause': 'Duraklat', 'rec.resume': 'Devam ettir',
  'rec.new': 'Düzenli ödeme ekle', 'rec.edit': 'Düzenli ödemeyi düzenle', 'rec.monthly_total': 'Aylık sabit gider',
  'rec.monthly_income': 'Aylık sabit gelir', 'rec.end': 'Bitiş tarihi', 'rec.no_end': 'Bitiş yok',
  'rec.future_only': 'Değişiklik sıradaki tekrarlardan itibaren geçerli olur; geçmiş işlemler değişmez.',
  'rec.saved': 'Düzenli ödeme güncellendi', 'rec.apply_future': 'Sonraki tekrarlara da uygula',
  'rec.from_tx': 'Bu işlemi düzenli yap', 'rec.how': 'Nasıl çalışır? Vadesi gelen ödeme o gün kendiliğinden işlenir ve bakiyene yansır.',
  'rec.see_all': 'Düzenli ödemeler', 'rec.count': '{n} düzenli ödeme',
  // güvenlik / hesap
  'err.locked': 'Çok fazla hatalı deneme. {n} dakika sonra tekrar dene.', 'err.attempts_left': '{n} deneme hakkın kaldı',
  'pw.title': 'Şifre değiştir', 'pw.current': 'Mevcut şifre', 'pw.new': 'Yeni şifre', 'pw.new2': 'Yeni şifre (tekrar)',
  'pw.mismatch': 'Yeni şifreler aynı değil', 'pw.done': 'Şifren değişti. Diğer cihazlardaki oturumlar kapatıldı.',
  'pw.wrong_current': 'Mevcut şifre hatalı', 'pw.same': 'Yeni şifre eskisiyle aynı olamaz', 'pw.hint': 'En az 8 karakter.',
  'del.title': 'Hesabı sil', 'del.body': 'Hesabın kapatılır ve bir daha giriş yapamazsın. Kayıtların silinmez: grup arkadaşlarının hesapları bozulmasın diye işlemlerin, borç ve hesaplaşma geçmişin yerinde kalır. Düzenli ödemelerin durdurulur.',
  'del.confirm': 'Onaylamak için şifreni gir', 'del.button': 'Hesabımı kalıcı olarak kapat', 'del.gone': 'Bu hesap kapatılmış',
  'del.sure': 'Emin misin? Bu işlem geri alınamaz.', 'set.security': 'Güvenlik', 'set.data': 'Veriler',
  // gelen kutusu / bildirim
  'inbox.title': 'Gelen kutusu', 'inbox.invites': 'Grup davetleri', 'inbox.receipts': 'Bekleyen dekontlar', 'inbox.notifications': 'Bildirimler',
  'inbox.empty': 'Gelen kutun boş. Davetler, dekontlar ve bildirimler burada görünür.', 'inbox.read_all': 'Tümünü okundu say',
  'inv.from': '{name} seni davet etti', 'inv.accept': 'Katıl', 'inv.decline': 'Reddet', 'inv.joined': '"{g}" grubuna katıldın',
  'inv.sent': 'Davet gönderildi; kabul edince gruba katılacak', 'inv.already': 'Bu kişi zaten davet edildi', 'inv.pending': 'Davet bekleniyor',
  'inv.cancel': 'Daveti geri çek', 'inv.invite': 'Davet et', 'grp.deleted_member': '{name} (hesap kapalı)',
  'push.title': 'Bildirimler', 'push.on': 'Bildirimleri aç', 'push.off': 'Bildirimleri kapat', 'push.enabled': 'Bu cihazda bildirimler açık',
  'push.denied': 'Bildirim izni reddedilmiş; telefon ayarlarından izin ver.', 'push.ios_hint': 'iPhone\'da bildirim için uygulamayı ana ekrana ekleyip oradan açman gerekir (iOS 16.4+).',
  'push.unsupported': 'Bu tarayıcı bildirimleri desteklemiyor.', 'push.server_off': 'Bildirimler sunucuda henüz ayarlanmadı.', 'push.test': 'Deneme bildirimi gönder',
  'push.what': 'Grup daveti, ortak harcama, işlenen düzenli ödeme ve bütçe uyarılarında haber verir.',
  // transfer
  'tr.title': 'Hesaplar arası transfer', 'tr.tab': 'Transfer', 'tr.from': 'Nereden', 'tr.to': 'Nereye', 'tr.amount': 'Tutar',
  'tr.to_amount': 'Hedefe geçen tutar ({c})', 'tr.rate': 'Sun Döviz kuru: 1 {from} = {v} {to}', 'tr.saved': 'Transfer yapıldı',
  'tr.locked': 'Transferin tutarı değiştirilemez; silip yeniden oluştur', 'tr.same': 'Kaynak ve hedef hesap aynı olamaz',
  'tr.hint': 'Transferler gelir/gider toplamlarına ve bütçelere sayılmaz.', 'tr.deleted': 'Transfer silindi', 'tr.need_two': 'Transfer için en az iki hesap gerekir',
  // bütçe
  'bud.title': 'Bütçeler', 'bud.new': 'Bütçe ekle', 'bud.edit': 'Bütçeyi düzenle', 'bud.category': 'Kategori', 'bud.amount': 'Aylık limit',
  'bud.spent': '{v} harcandı', 'bud.left': '{v} kaldı', 'bud.over': '{v} aşıldı', 'bud.empty': 'Market, fatura, yemek… Kategorilere aylık limit koy; %80\'e gelince ve aşınca haber verelim.',
  'bud.exists': 'Bu kategori için zaten bütçe var', 'bud.home': 'Bu ayki bütçeler',
  // arama / dışa aktarma
  'tx.search': 'Ara: açıklama, alıcı, hesap, tutar', 'tx.export': 'Excel (CSV) indir', 'tx.no_results': 'Sonuç yok',
  // çevrimdışı
  'off.banner': 'Çevrimdışısın', 'off.pending_n': '{n} değişiklik gönderilmeyi bekliyor', 'off.syncing': 'Gönderiliyor…',
  'off.queued': 'İnternet yok: kaydedildi, bağlantı gelince gönderilecek', 'off.synced': '{n} değişiklik gönderildi',
  'off.failed': 'Bir değişiklik sunucuda kabul edilmedi: {e}', 'off.pending': 'Bekliyor', 'off.needs_net': 'Bu işlem için internet gerekli',
  'off.logout_warn': '{n} değişiklik henüz gönderilmedi. Çıkarsan silinecek. Yine de çıkılsın mı?', 'off.retry': 'Şimdi gönder',
  // tanıtım turu
  'tour.skip': 'Geç', 'tour.prev': 'Geri', 'tour.next': 'İleri', 'tour.finish': 'Başla', 'tour.replay': 'Tanıtımı tekrar izle',
  'tour.welcome.title': 'Cüzdan\'a hoş geldin 👋', 'tour.welcome.body': 'Kısa bir turla nerede ne yapabileceğini gösterelim. Bir dakika sürmez.',
  'tour.total.title': 'Toplam bakiyen', 'tour.total.body': 'Tüm hesaplarının toplamı burada. Dövizli hesaplar Sun Döviz kuruyla TL\'ye çevrilip eklenir.',
  'tour.add.title': 'Gelir ve gider ekle', 'tour.add.body': 'Buradan harcama ya da gelir eklersin: tutar, hesap, kategori. Grup seçersen ev arkadaşlarınla bölüşülür; "Tekrar" ile her ay otomatik işlenir.',
  'tour.receipt.title': 'Dekontla otomatik doldur', 'tour.receipt.body': 'Ekleme ekranında "Dekont ekle"ye dokun, banka dekontunu (PDF veya ekran görüntüsü) seç. Alıcı, IBAN, banka, tutar ve tarih kendiliğinden dolar.',
  'tour.share.title': 'Banka uygulamasından gönder', 'tour.share.body': 'Dekontu doğrudan paylaş menüsünden gönderebilirsin. iPhone için kurulum buradaki Ayarlar\'da: "iPhone\'dan dekont gönder". Android\'de paylaş menüsünde "Cüzdan" çıkar.',
  'tour.goals.title': 'Birikim hedefleri', 'tour.goals.body': 'Araba, tatil, telefon… Hedef koy, biriktirdikçe para ekle; ayda ne kadar ayırman gerektiğini hesaplarız.',
  'tour.payees.title': 'Kime gitti?', 'tour.payees.body': 'Dekontlu ve alıcısı yazılı işlemler kişi ve firmaya göre toplanır: kime ne kadar gönderdin, kimden ne geldi.',
  'tour.rates.title': 'Döviz kurları', 'tour.rates.body': 'Sun Döviz\'in güncel alış/satış kurları ve çevirici burada.',
  'tour.recurring.title': 'Düzenli ödemeler', 'tour.recurring.body': 'Kira, aidat, fatura… Bir kez ekle, her ay kendiliğinden işlensin. Tutar değişirse buradan güncelle.',
  'tour.tx.title': 'İşlemler', 'tour.tx.body': 'Tüm hareketlerin; aya, hesaba ve kategoriye göre süzebilirsin. Bir işleme dokunup düzenleyebilirsin.',
  'tour.accounts.title': 'Hesaplar', 'tour.accounts.body': 'Banka, nakit, kart hesapların; her birinin bakiyesi ve bu ayki hareketi.',
  'tour.groups.title': 'Gruplar', 'tour.groups.body': 'Ev arkadaşlarınla ortak harcamalar: kim ne ödedi, kim kime borçlu, tek dokunuşla hesaplaşma.',
  'tour.update.title': 'Yenilikler ✨', 'tour.update.body': 'Uygulamaya yeni özellikler geldi. Kısaca nerede olduklarını gösterelim.',
  'tour.inbox.title': 'Gelen kutusu', 'tour.inbox.body': 'Grup davetleri, banka uygulamasından gönderdiğin dekontlar ve bildirimler burada. Rozet bekleyenleri gösterir.',
  'tour.invite.title': 'Gruba davet', 'tour.invite.body': 'Grubun içinde "Ekle"ye dokunup kullanıcı adıyla davet gönder. Davet, karşı tarafın gelen kutusuna düşer; kabul edince gruba katılır.',
  'tour.budgets.title': 'Bütçeler', 'tour.budgets.body': 'Kategorilere aylık limit koy (market, fatura…). %80\'e gelince ve aşınca haber veririz.',
  'tour.transfer.title': 'Hesaplar arası transfer', 'tour.transfer.body': '+ ile açılan ekranda üstteki "Transfer"e dokun: bankadan nakde, karttan hesaba para aktar. Gelir/gider toplamlarını şişirmez; dövizli hesaplarda kur önerilir.',
  'tour.search.title': 'Arama ve Excel', 'tour.search.body': 'İşlemler sayfasında açıklama, alıcı, hesap ya da tutarla ara. Yanındaki indirme düğmesiyle ayı Excel (CSV) olarak al.',
  'tour.security.title': 'Bildirim ve güvenlik', 'tour.security.body': 'Ayarlar\'dan bildirimleri aç, şifreni değiştir. 5 hatalı girişte hesap 5 dakika kilitlenir.',
  'tour.offline.title': 'İnternetsiz de çalışır', 'tour.offline.body': 'Bağlantın yokken eklediğin ve değiştirdiğin kayıtlar telefonda bekler; internet gelince kendiliğinden gönderilir.',
  'tour.cards.title': 'Kredi kartı borçları', 'tour.cards.body': 'Hesaplar\'da + ile "Kart" türünde hesap aç: limitini, şu anki borcunu ve son ödeme gününü yaz. Kartın sayfasında güncel borç, kalan limit ve "Borç öde" düğmesi var.',
  'tour.goalfx.title': 'Dövizli hedef', 'tour.goalfx.body': 'Birikim hedefi açarken para birimini seç (ör. sterlinle alınacak araba için GBP). O para biriminde hesabın yoksa tek dokunuşla açılır.',
  'tour.done.title': 'Hazırsın!', 'tour.done.body': 'İlk işini eklemek için alttaki + düğmesine dokun. Bu tanıtımı Ayarlar\'dan tekrar izleyebilirsin.',
  // dekont
  'rc.title': 'Dekont', 'rc.add': 'Dekont ekle', 'rc.attached': 'Dekont eklendi', 'rc.reading': 'Dekont okunuyor…',
  'rc.uploading': 'Yükleniyor…', 'rc.read_ok': 'Dekont okundu, bilgileri kontrol et', 'rc.unreadable': 'Dekont okunamadı; bilgileri elle gir',
  'rc.already_used': 'Bu dekont zaten bir işleme eklenmiş', 'rc.recipient': 'Alıcı (kime gitti)', 'rc.sender': 'Gönderen (kimden geldi)',
  'rc.name_ph': 'Ad soyad / firma', 'rc.view': 'Dekontu gör', 'rc.inbox': 'Gelen dekontlar', 'rc.inbox_n': '{n} dekont işlenmeyi bekliyor',
  'rc.inbox_hint': 'Kestirme veya paylaş menüsüyle gönderdiğin dekontlar burada. Dokun, işleme çevir.', 'rc.dismiss': 'Yok say',
  'rc.process': 'İşleme çevir', 'rc.shared': 'Paylaşılan dekont alındı', 'rc.unknown': 'Okunamayan dekont',
  // kestirme
  'sc.title': 'iPhone\'dan dekont gönder', 'sc.intro': 'Banka uygulamasında dekontu paylaşırken "Cüzdan\'a Gönder" seçeneği çıksın. Bir kez kurman yeterli.',
  'sc.create': 'Anahtar oluştur', 'sc.rotate': 'Yeni anahtar oluştur', 'sc.revoke': 'Anahtarı kaldır', 'sc.has': 'Anahtar oluşturulmuş. Kaybettiysen yenisini oluştur (eskisi çalışmaz).',
  'sc.token_once': 'Bu anahtarı şimdi kopyala; bir daha gösterilmez.', 'sc.copy': 'Kopyala', 'sc.copied': 'Kopyalandı',
  'sc.url': 'Adres', 'sc.token': 'Anahtar',
  'sc.step1': 'Kestirmeler uygulamasını aç → sağ üstte + → adını "Cüzdan\'a Gönder" yap.',
  'sc.step2': 'Ayarlar (ⓘ) → "Paylaşma Ekranında Göster"i aç; alınacak tür: Dosyalar, PDF\'ler, Görüntüler.',
  'sc.step3': '"URL İçeriğini Al" eylemini ekle: URL = yukarıdaki adres, Yöntem = POST.',
  'sc.step4': 'Başlıklar: Authorization = "Bearer " + anahtar. İstek Gövdesi = Form; alan adı "file", tür Dosya, değer = Kestirme Girdisi.',
  'sc.step5': '"Bildirim Göster" eylemi ekle (isteğe bağlı) ve kaydet. Artık dekontu paylaş → Cüzdan\'a Gönder.',
  'sc.android': 'Android: uygulamayı ana ekrana ekledikten sonra paylaş menüsünde "Cüzdan" doğrudan çıkar.',
  'sc.https': 'Not: Kestirmenin çalışması için sunucunun telefondan erişilebilir olması gerekir (ev ağı veya HTTPS alan adı).',
  // kime gitti
  'cp.title': 'Kime gitti?', 'cp.title_in': 'Kimden geldi?', 'cp.out': 'Giden', 'cp.in': 'Gelen', 'cp.empty': 'Dekont eklediğin veya alıcı yazdığın işlemler burada kişi ve firmaya göre toplanır.',
  'cp.count': '{n} işlem', 'cp.last': 'Son: {d}', 'cp.all_time': 'Tüm zamanlar', 'cp.this_month': 'Bu ay', 'cp.unnamed': 'İsimsiz',
  // kurlar
  'fx.title': 'Döviz kurları', 'fx.source': 'Kaynak: Sun Döviz', 'fx.buy': 'Alış', 'fx.sell': 'Satış',
  'fx.online': 'EFT / Havale', 'fx.desk': 'Gişe', 'fx.updated': 'Güncelleme: {d}', 'fx.stale': 'Kaynağa ulaşılamadı, son bilinen kurlar gösteriliyor.',
  'fx.unavailable': 'Kurlar şu an alınamıyor.', 'fx.calc': 'Hesapla', 'fx.amount': 'Tutar', 'fx.result': 'Sonuç',
  'fx.rate_line': '1 {from} = {v} {to}', 'fx.swap': 'Yer değiştir', 'fx.incl': '{v} dahil · Sun Döviz alış kuru',
  'fx.approx': '≈ {v}', 'fx.cross': 'Çapraz kurlar', 'fx.hint': 'Döviz → TL alış, TL → döviz satış kuruyla hesaplanır.',
  // gruplar
  'grp.title': 'Gruplar', 'grp.new': 'Yeni grup', 'grp.name': 'Grup adı', 'grp.name_ph': 'ör. Ev', 'grp.members': '{n} üye',
  'grp.empty': 'Ev arkadaşlarınla ortak harcamaları takip etmek için bir grup kur.',
  'grp.tab.tx': 'Harcamalar', 'grp.tab.debts': 'Borçlar', 'grp.tab.summary': 'Özet',
  'grp.add_member': 'Gruba davet et', 'grp.member_username': 'Kullanıcı adı', 'grp.member_added': 'Üye eklendi',
  'grp.leave': 'Gruptan ayrıl', 'grp.leave_confirm': 'Gruptan ayrılırsan grubun harcamalarını göremezsin.',
  'grp.leave_blocked': 'Açık borcun ya da alacağın var; önce hesaplaşın.', 'grp.rename': 'Grubu düzenle',
  'grp.you_get': 'Alacağın {v}', 'grp.you_owe': 'Borcun {v}', 'grp.settled': 'Hesaplar denk',
  'grp.owes': '{from} → {to}', 'grp.mark_paid': 'Ödendi', 'grp.settlements': 'Hesaplaşmalar',
  'grp.no_debts': 'Herkes ödeşmiş. 🎉', 'grp.total_spent': 'Toplam harcama', 'grp.paid': 'Ödedi', 'grp.share': 'Payı',
  'grp.recurring': 'Ortak düzenli giderler', 'grp.member_you': '{name} (sen)', 'grp.net': 'Net durum',
  'settle.title': 'Ödeme kaydet', 'settle.from': 'Ödeyen', 'settle.to': 'Alan', 'settle.to_account': 'Hesabıma işle',
  'settle.saved': 'Ödeme kaydedildi', 'settle.note': 'Not',
  // ayarlar
  'set.title': 'Ayarlar', 'set.theme': 'Tema', 'set.lang': 'Dil', 'set.account': 'Hesap', 'set.install': 'Ana ekrana ekle',
  'set.install_hint': 'Safari\'de Paylaş ↑ düğmesine dokun, ardından "Ana Ekrana Ekle"yi seç. İkon seçili temanın renginde olur.',
  'set.installed': 'Uygulama ana ekrandan çalışıyor', 'set.version': 'Sürüm {v}',
  'theme.a': 'Gece', 'theme.b': 'Defter', 'theme.c': 'Cepler',
  'theme.a.desc': 'Koyu, sade, rakam odaklı', 'theme.b.desc': 'Kağıt, serif, muhasebe defteri', 'theme.c.desc': 'Renkli cepler, sıcak tonlar',
  // ay
  'month.pick': 'Ay seç',
  // hatalar
  'err.network': 'Sunucuya ulaşılamadı', 'err.generic': 'Bir şeyler ters gitti', 'err.login': 'Kullanıcı adı veya şifre hatalı',
  'err.taken': 'Bu kullanıcı adı alınmış', 'err.username': 'Kullanıcı adı 3-32 karakter olmalı (harf, rakam, _ . -)',
  'err.password': 'Şifre 8-72 karakter olmalı', 'err.not_found': 'Bulunamadı', 'err.account_exists': 'Bu isimde bir hesabın var',
  'err.user_not_found': 'Kullanıcı bulunamadı', 'err.already_member': 'Kullanıcı zaten grupta',
}

const en = {
  'app.name': 'cüzdan',
  'nav.home': 'Home', 'nav.tx': 'Activity', 'nav.tx.b': 'Ledger', 'nav.accounts': 'Accounts', 'nav.accounts.c': 'Pockets',
  'nav.groups': 'Groups', 'nav.add': 'Add transaction', 'nav.profile': 'Profile', 'nav.back': 'Back',
  'common.save': 'Save', 'common.cancel': 'Cancel', 'common.close': 'Close', 'common.delete': 'Delete', 'common.edit': 'Edit',
  'common.all': 'All', 'common.none': 'None', 'common.more': 'Load more', 'common.loading': 'Loading…',
  'common.retry': 'Retry', 'common.done': 'Done', 'common.add': 'Add', 'common.create': 'Create',
  'common.income': 'Income', 'common.expense': 'Expense', 'common.net': 'Net', 'common.balance': 'Balance', 'common.amount': 'Amount',
  'common.today': 'Today', 'common.yesterday': 'Yesterday', 'common.empty': 'Nothing here yet',
  'common.you': 'you', 'common.confirm_delete': 'Delete this? This cannot be undone.',
  'auth.login': 'Log in', 'auth.register': 'Create account', 'auth.username': 'Username', 'auth.password': 'Password',
  'auth.tagline': 'Personal and shared spending, in one pocket.', 'auth.logout': 'Log out',
  'auth.username_hint': '3-32 chars: letters, digits, _ . -', 'auth.password_hint': 'At least 8 characters',
  'home.total': 'Total balance', 'home.accounts': 'Accounts', 'home.recent': 'Recent', 'home.recent.b': 'Recent entries',
  'home.open_ledger': 'Open ledger', 'home.spent_ratio': '{p}% of income spent', 'home.no_income': 'No income yet this month',
  'home.hello': 'Hi {name}', 'home.pockets_total': 'Total in {n} pockets', 'home.month_income': '{m} income',
  'home.month_expense': '{m} spending', 'home.this_month': 'This month {v}', 'home.no_accounts': 'Add your first account to begin',
  'home.add_account': 'Add account', 'home.other_currencies': 'Other: {v}',
  'ledger.date': 'Date', 'ledger.desc': 'Description', 'ledger.amount': 'Amount', 'ledger.balance': 'Balance',
  'add.title': 'New transaction', 'add.title.b': 'New entry', 'add.edit_title': 'Edit transaction', 'add.save.b': 'Write to ledger',
  'add.account': 'Account', 'add.group': 'Group', 'add.category': 'Category', 'add.note': 'Note', 'add.note_ph': 'Add a note',
  'add.date': 'Date', 'add.repeat': 'Repeat', 'add.split': 'Split', 'add.after': 'After {acc}',
  'add.need_amount': 'Enter an amount', 'add.need_account': 'Add an account first', 'add.saved': 'Saved',
  'add.deleted': 'Transaction deleted', 'add.repeat_saved': 'Recurring transaction created',
  'add.locked': 'Paid by {name}; account cannot be changed',
  'repeat.none': 'Does not repeat', 'repeat.weekly': 'Every week', 'repeat.monthly': 'Every month', 'repeat.yearly': 'Every year',
  'split.equal_all': 'Equally, everyone', 'split.equal_n': 'Equally, {n} people', 'split.choose': 'Choose people',
  'split.exact': 'Exact amounts', 'split.remaining': 'Remaining: {v}', 'split.sum_error': 'Shares must add up to the amount',
  'split.title': 'How to split?', 'split.mode_equal': 'Equal', 'split.mode_exact': 'Exact',
  'cat.groceries': 'Groceries', 'cat.bills': 'Bills', 'cat.transport': 'Transport', 'cat.food': 'Food & drink', 'cat.rent': 'Rent',
  'cat.subscription': 'Subscriptions', 'cat.gift': 'Gift', 'cat.other': 'Other', 'cat.salary': 'Salary', 'cat.extra': 'Side income',
  'cat.health': 'Health', 'cat.shopping': 'Shopping', 'cat.none': 'Uncategorized', 'cat.settlement': 'Settlement',
  'tx.title': 'Activity', 'tx.title.b': 'Ledger', 'tx.empty': 'No transactions this month', 'tx.filter_all': 'All',
  'tx.paid_by': 'Paid by {name}', 'tx.your_share': 'Your share {v}', 'tx.recurring': 'Recurring',
  'acc.title': 'Accounts', 'acc.title.c': 'Pockets', 'acc.new': 'New account', 'acc.name': 'Account name', 'acc.kind': 'Type',
  'acc.currency': 'Currency', 'acc.kind.bank': 'Bank', 'acc.kind.cash': 'Cash', 'acc.kind.card': 'Card',
  'acc.kind.savings': 'Savings', 'acc.rename': 'Edit account', 'acc.delete_blocked': 'Accounts with transactions cannot be deleted',
  'acc.name_ph': 'e.g. Checking', 'acc.month_in_out': 'This month',
  'card.limit': 'Credit limit', 'card.opening_debt': 'Current balance owed', 'card.opening_hint': 'What the card owes today. Not counted as spending.',
  'card.due_day': 'Payment due day (of the month)', 'card.due_ph': 'e.g. 15', 'card.debt': 'Balance owed', 'card.available': 'Available',
  'card.available_short': '{v} available', 'card.limit_of': 'limit {v}', 'card.no_limit': 'No limit set; edit the card to add one.',
  'card.due': 'Due {d}', 'card.days_left': '{n} days left', 'card.due_today': 'today', 'card.pay': 'Pay off', 'card.add_spend': 'Add spending',
  'cat.opening': 'Opening balance',
  'goal.title': 'Savings goals', 'goal.new': 'New goal', 'goal.name': 'Goal name', 'goal.target': 'Target amount',
  'goal.date': 'Target date', 'goal.no_date': 'No date', 'goal.monthly': 'Save {v} a month', 'goal.done': 'Goal reached 🎉',
  'goal.left': '{v} to go', 'goal.empty': 'A car, a holiday, a new phone… Set a goal and add to it as you save.',
  'goal.name_ph': 'e.g. Car', 'goal.account': 'Which account holds the money?', 'goal.icon': 'Icon',
  'goal.initial': 'Already saved (optional)', 'goal.saved': '{v} saved', 'goal.of': 'of {v}',
  'goal.deposit': 'Add money', 'goal.withdraw': 'Withdraw', 'goal.history': 'History', 'goal.no_history': 'No activity yet.',
  'goal.note_ph': 'Note (optional)', 'goal.added': 'Added {v} to goal', 'goal.taken': 'Took {v} from goal',
  'goal.too_much': 'Not that much saved in this goal', 'goal.days_left': '{n} days left', 'goal.overdue': 'Target date passed',
  'goal.account_note': '{acc} balance: {v}', 'goal.currency': 'Which currency?',
  'goal.no_account': 'You have no {c} account. The goal\'s money should sit in a {c} account.', 'goal.create_account': 'Open a {c} savings account',
  'goal.auto_account': 'Savings {c}', 'goal.monthly_hint': '{v} a month to reach it on time',
  'rec.title': 'Recurring payments', 'rec.empty': 'Rent, dues, bills, subscriptions… Add once, recorded automatically every month.',
  'rec.next': 'Next: {d}', 'rec.paused': 'Paused', 'rec.pause': 'Pause', 'rec.resume': 'Resume',
  'rec.new': 'Add recurring payment', 'rec.edit': 'Edit recurring payment', 'rec.monthly_total': 'Monthly fixed costs',
  'rec.monthly_income': 'Monthly fixed income', 'rec.end': 'End date', 'rec.no_end': 'No end',
  'rec.future_only': 'Changes apply from the next occurrence; past transactions stay as they are.',
  'rec.saved': 'Recurring payment updated', 'rec.apply_future': 'Also apply to future occurrences',
  'rec.from_tx': 'Make this recurring', 'rec.how': 'How it works: each payment is recorded automatically on its due day and hits your balance.',
  'rec.see_all': 'Recurring payments', 'rec.count': '{n} recurring payments',
  'err.locked': 'Too many failed attempts. Try again in {n} min.', 'err.attempts_left': '{n} attempts left',
  'pw.title': 'Change password', 'pw.current': 'Current password', 'pw.new': 'New password', 'pw.new2': 'New password (again)',
  'pw.mismatch': 'New passwords do not match', 'pw.done': 'Password changed. Other devices were signed out.',
  'pw.wrong_current': 'Current password is wrong', 'pw.same': 'New password must differ from the old one', 'pw.hint': 'At least 8 characters.',
  'del.title': 'Delete account', 'del.body': 'Your account is closed and you can no longer sign in. Your records are kept so your groups stay consistent: transactions, debts and settlements remain. Recurring payments are stopped.',
  'del.confirm': 'Enter your password to confirm', 'del.button': 'Permanently close my account', 'del.gone': 'This account has been closed',
  'del.sure': 'Are you sure? This cannot be undone.', 'set.security': 'Security', 'set.data': 'Data',
  'inbox.title': 'Inbox', 'inbox.invites': 'Group invites', 'inbox.receipts': 'Pending receipts', 'inbox.notifications': 'Notifications',
  'inbox.empty': 'Your inbox is empty. Invites, receipts and notifications show up here.', 'inbox.read_all': 'Mark all read',
  'inv.from': '{name} invited you', 'inv.accept': 'Join', 'inv.decline': 'Decline', 'inv.joined': 'You joined "{g}"',
  'inv.sent': 'Invite sent; they join once they accept', 'inv.already': 'Already invited', 'inv.pending': 'Invite pending',
  'inv.cancel': 'Cancel invite', 'inv.invite': 'Invite', 'grp.deleted_member': '{name} (account closed)',
  'push.title': 'Notifications', 'push.on': 'Turn on notifications', 'push.off': 'Turn off notifications', 'push.enabled': 'Notifications are on for this device',
  'push.denied': 'Notification permission was denied; allow it in your phone settings.', 'push.ios_hint': 'On iPhone, add the app to your home screen and open it from there (iOS 16.4+).',
  'push.unsupported': 'This browser does not support notifications.', 'push.server_off': 'Notifications are not configured on the server yet.', 'push.test': 'Send a test notification',
  'push.what': 'Get notified about group invites, shared expenses, recurring payments and budget alerts.',
  'tr.title': 'Transfer between accounts', 'tr.tab': 'Transfer', 'tr.from': 'From', 'tr.to': 'To', 'tr.amount': 'Amount',
  'tr.to_amount': 'Amount received ({c})', 'tr.rate': 'Sun Döviz rate: 1 {from} = {v} {to}', 'tr.saved': 'Transfer done',
  'tr.locked': 'A transfer amount cannot be changed; delete and recreate it', 'tr.same': 'Source and target must differ',
  'tr.hint': 'Transfers are not counted in income/expense totals or budgets.', 'tr.deleted': 'Transfer deleted', 'tr.need_two': 'You need at least two accounts to transfer',
  'bud.title': 'Budgets', 'bud.new': 'Add budget', 'bud.edit': 'Edit budget', 'bud.category': 'Category', 'bud.amount': 'Monthly limit',
  'bud.spent': '{v} spent', 'bud.left': '{v} left', 'bud.over': '{v} over', 'bud.empty': 'Groceries, bills, food… Set monthly limits per category; we warn you at 80% and when you go over.',
  'bud.exists': 'There is already a budget for this category', 'bud.home': 'Budgets this month',
  'tx.search': 'Search: note, recipient, account, amount', 'tx.export': 'Download Excel (CSV)', 'tx.no_results': 'No results',
  'off.banner': 'You are offline', 'off.pending_n': '{n} changes waiting to be sent', 'off.syncing': 'Sending…',
  'off.queued': 'No connection: saved, will be sent when you are back online', 'off.synced': '{n} changes sent',
  'off.failed': 'A change was rejected by the server: {e}', 'off.pending': 'Pending', 'off.needs_net': 'This needs an internet connection',
  'off.logout_warn': '{n} changes have not been sent yet and will be lost if you sign out. Sign out anyway?', 'off.retry': 'Send now',
  'tour.skip': 'Skip', 'tour.prev': 'Back', 'tour.next': 'Next', 'tour.finish': 'Start', 'tour.replay': 'Replay the intro',
  'tour.welcome.title': 'Welcome to Wallet 👋', 'tour.welcome.body': 'A quick tour of where to do what. It takes less than a minute.',
  'tour.total.title': 'Your total balance', 'tour.total.body': 'All your accounts added up. Foreign-currency accounts are converted to TRY at Sun Döviz rates.',
  'tour.add.title': 'Add income and expenses', 'tour.add.body': 'Add a spending or income here: amount, account, category. Pick a group to split with housemates; "Repeat" records it automatically every month.',
  'tour.receipt.title': 'Fill in from a receipt', 'tour.receipt.body': 'In the add screen tap "Add receipt" and pick your bank receipt (PDF or screenshot). Recipient, IBAN, bank, amount and date fill in automatically.',
  'tour.share.title': 'Send from your banking app', 'tour.share.body': 'Send receipts straight from the share menu. iPhone setup is in Settings here: "Send receipts from iPhone". On Android "Wallet" shows up in the share menu.',
  'tour.goals.title': 'Savings goals', 'tour.goals.body': 'A car, a holiday, a phone… Set a goal and add money as you save; we work out the monthly amount.',
  'tour.payees.title': 'Paid to', 'tour.payees.body': 'Transactions with receipts or recipients are grouped by person and company: who you paid and who paid you.',
  'tour.rates.title': 'Exchange rates', 'tour.rates.body': 'Current Sun Döviz buy/sell rates and a converter.',
  'tour.recurring.title': 'Recurring payments', 'tour.recurring.body': 'Rent, dues, bills… Add once and it is recorded every month. Update it here if the amount changes.',
  'tour.tx.title': 'Transactions', 'tour.tx.body': 'Everything you recorded; filter by month, account and category. Tap one to edit it.',
  'tour.accounts.title': 'Accounts', 'tour.accounts.body': 'Your bank, cash and card accounts with balances and this month\'s activity.',
  'tour.groups.title': 'Groups', 'tour.groups.body': 'Shared expenses with housemates: who paid what, who owes whom, settle up in one tap.',
  'tour.update.title': 'What\'s new ✨', 'tour.update.body': 'New features have arrived. Here is where to find them.',
  'tour.inbox.title': 'Inbox', 'tour.inbox.body': 'Group invites, receipts sent from your banking app and notifications live here. The badge shows what is waiting.',
  'tour.invite.title': 'Invite to a group', 'tour.invite.body': 'In a group tap "Add" and invite by username. The invite lands in their inbox; they join once they accept.',
  'tour.budgets.title': 'Budgets', 'tour.budgets.body': 'Set monthly limits per category (groceries, bills…). We warn you at 80% and when you go over.',
  'tour.transfer.title': 'Transfers between accounts', 'tour.transfer.body': 'In the + screen tap "Transfer" at the top: bank to cash, card to account. It does not inflate income/expense; rates are suggested for foreign accounts.',
  'tour.search.title': 'Search and Excel', 'tour.search.body': 'Search transactions by note, recipient, account or amount. The download button next to it exports the month as Excel (CSV).',
  'tour.security.title': 'Notifications and security', 'tour.security.body': 'Turn on notifications and change your password in Settings. After 5 wrong passwords the account locks for 5 minutes.',
  'tour.offline.title': 'Works offline', 'tour.offline.body': 'Entries you add or change without a connection wait on your phone and are sent automatically once you are back online.',
  'tour.cards.title': 'Credit card balances', 'tour.cards.body': 'In Accounts tap + and choose "Card": enter the limit, what it owes now and the due day. The card page shows the balance owed, available limit and a "Pay off" button.',
  'tour.goalfx.title': 'Goals in any currency', 'tour.goalfx.body': 'Pick a currency when creating a savings goal (e.g. GBP for a car bought in pounds). If you have no account in it, open one with a tap.',
  'tour.done.title': 'You\'re all set!', 'tour.done.body': 'Tap the + button below to add your first entry. You can replay this intro from Settings.',
  'rc.title': 'Receipt', 'rc.add': 'Add receipt', 'rc.attached': 'Receipt attached', 'rc.reading': 'Reading receipt…',
  'rc.uploading': 'Uploading…', 'rc.read_ok': 'Receipt read, please check the details', 'rc.unreadable': 'Could not read the receipt; fill in manually',
  'rc.already_used': 'This receipt is already attached to a transaction', 'rc.recipient': 'Recipient (paid to)', 'rc.sender': 'Sender (received from)',
  'rc.name_ph': 'Name / company', 'rc.view': 'View receipt', 'rc.inbox': 'Incoming receipts', 'rc.inbox_n': '{n} receipts waiting',
  'rc.inbox_hint': 'Receipts sent via the Shortcut or share menu land here. Tap to turn into a transaction.', 'rc.dismiss': 'Dismiss',
  'rc.process': 'Create transaction', 'rc.shared': 'Shared receipt received', 'rc.unknown': 'Unreadable receipt',
  'sc.title': 'Send receipts from iPhone', 'sc.intro': 'Get a "Send to Wallet" option when sharing a receipt from your banking app. One-time setup.',
  'sc.create': 'Create key', 'sc.rotate': 'Create new key', 'sc.revoke': 'Remove key', 'sc.has': 'A key exists. If you lost it, create a new one (the old one stops working).',
  'sc.token_once': 'Copy this key now; it will not be shown again.', 'sc.copy': 'Copy', 'sc.copied': 'Copied',
  'sc.url': 'URL', 'sc.token': 'Key',
  'sc.step1': 'Open Shortcuts → + at top right → name it "Send to Wallet".',
  'sc.step2': 'Settings (ⓘ) → turn on "Show in Share Sheet"; input types: Files, PDFs, Images.',
  'sc.step3': 'Add "Get Contents of URL": URL = the address above, Method = POST.',
  'sc.step4': 'Headers: Authorization = "Bearer " + key. Request Body = Form; field "file", type File, value = Shortcut Input.',
  'sc.step5': 'Optionally add "Show Notification" and save. Now share a receipt → Send to Wallet.',
  'sc.android': 'Android: after adding the app to your home screen, "Wallet" appears directly in the share menu.',
  'sc.https': 'Note: the server must be reachable from your phone (home network or an HTTPS domain).',
  'cp.title': 'Paid to', 'cp.title_in': 'Received from', 'cp.out': 'Outgoing', 'cp.in': 'Incoming', 'cp.empty': 'Transactions with a receipt or recipient are grouped by person and company here.',
  'cp.count': '{n} transactions', 'cp.last': 'Last: {d}', 'cp.all_time': 'All time', 'cp.this_month': 'This month', 'cp.unnamed': 'Unnamed',
  'fx.title': 'Exchange rates', 'fx.source': 'Source: Sun Döviz', 'fx.buy': 'Buy', 'fx.sell': 'Sell',
  'fx.online': 'Transfer', 'fx.desk': 'Desk (cash)', 'fx.updated': 'Updated: {d}', 'fx.stale': 'Source unreachable, showing last known rates.',
  'fx.unavailable': 'Rates are unavailable right now.', 'fx.calc': 'Convert', 'fx.amount': 'Amount', 'fx.result': 'Result',
  'fx.rate_line': '1 {from} = {v} {to}', 'fx.swap': 'Swap', 'fx.incl': 'incl. {v} · Sun Döviz buy rate',
  'fx.approx': '≈ {v}', 'fx.cross': 'Cross rates', 'fx.hint': 'Foreign → TRY uses the buy rate, TRY → foreign the sell rate.',
  'grp.title': 'Groups', 'grp.new': 'New group', 'grp.name': 'Group name', 'grp.name_ph': 'e.g. Home', 'grp.members': '{n} members',
  'grp.empty': 'Create a group to track shared spending with your housemates.',
  'grp.tab.tx': 'Expenses', 'grp.tab.debts': 'Balances', 'grp.tab.summary': 'Summary',
  'grp.add_member': 'Invite to group', 'grp.member_username': 'Username', 'grp.member_added': 'Member added',
  'grp.leave': 'Leave group', 'grp.leave_confirm': 'If you leave, you will no longer see this group\'s expenses.',
  'grp.leave_blocked': 'You have an open balance; settle up first.', 'grp.rename': 'Edit group',
  'grp.you_get': 'You get {v}', 'grp.you_owe': 'You owe {v}', 'grp.settled': 'All settled',
  'grp.owes': '{from} → {to}', 'grp.mark_paid': 'Paid', 'grp.settlements': 'Settlements',
  'grp.no_debts': 'Everyone is square. 🎉', 'grp.total_spent': 'Total spent', 'grp.paid': 'Paid', 'grp.share': 'Share',
  'grp.recurring': 'Shared recurring', 'grp.member_you': '{name} (you)', 'grp.net': 'Balances',
  'settle.title': 'Record payment', 'settle.from': 'From', 'settle.to': 'To', 'settle.to_account': 'Record in my account',
  'settle.saved': 'Payment recorded', 'settle.note': 'Note',
  'set.title': 'Settings', 'set.theme': 'Theme', 'set.lang': 'Language', 'set.account': 'Account', 'set.install': 'Add to Home Screen',
  'set.install_hint': 'In Safari tap Share ↑, then "Add to Home Screen". The icon uses the selected theme\'s colors.',
  'set.installed': 'Running from the Home Screen', 'set.version': 'Version {v}',
  'theme.a': 'Night', 'theme.b': 'Ledger', 'theme.c': 'Pockets',
  'theme.a.desc': 'Dark, minimal, numbers first', 'theme.b.desc': 'Paper, serif, bookkeeping', 'theme.c.desc': 'Colorful pockets, warm tones',
  'month.pick': 'Pick a month',
  'err.network': 'Could not reach the server', 'err.generic': 'Something went wrong', 'err.login': 'Wrong username or password',
  'err.taken': 'That username is taken', 'err.username': 'Username must be 3-32 chars (letters, digits, _ . -)',
  'err.password': 'Password must be 8-72 characters', 'err.not_found': 'Not found', 'err.account_exists': 'You already have an account with this name',
  'err.user_not_found': 'User not found', 'err.already_member': 'User is already in the group',
}

export const dicts = { tr, en }

/** t('home.hello', {name}) — prefs.lang'e bağlı olduğu için şablonlarda reaktiftir. */
export function t(key, params) {
  let s = dicts[prefs.lang][key] ?? dicts.tr[key] ?? key
  if (params) for (const [k, v] of Object.entries(params)) s = s.replaceAll(`{${k}}`, v)
  return s
}

/** Tema varyantı olan anahtar: t2('nav.tx', 'b') → 'nav.tx.b' varsa onu kullanır. */
export function tv(key, theme) {
  const k = `${key}.${theme}`
  return dicts[prefs.lang][k] || dicts.tr[k] ? t(k) : t(key)
}

// Sunucu hata mesajları Türkçe döner; bilinenleri dile çeviriyoruz.
const serverErrors = {
  'kullanici adi veya sifre hatali': 'err.login',
  'bu kullanici adi alinmis': 'err.taken',
  'sifre 8-72 karakter arasinda olmali': 'err.password',
  'bu isimde bir hesabiniz zaten var': 'err.account_exists',
  'kullanici bulunamadi': 'err.user_not_found',
  'kullanici zaten grupta': 'err.already_member',
  'islemi olan hesap silinemez': 'acc.delete_blocked',
  'grupta acik borcunuz veya alacaginiz var; ayrilmadan once hesaplasin': 'grp.leave_blocked',
  'paylarin toplami islem tutarina esit olmali': 'split.sum_error',
  'hedefte bu kadar birikim yok': 'goal.too_much',
  'dekont bulunamadi veya baska bir isleme bagli': 'rc.already_used',
  'mevcut sifre hatali': 'pw.wrong_current',
  'yeni sifre eskisiyle ayni olamaz': 'pw.same',
  'bu hesap silinmis': 'del.gone',
  'bu kullanici zaten davet edildi': 'inv.already',
  'transferin tutari degistirilemez; silip yeniden olusturun': 'tr.locked',
  'kaynak ve hedef hesap ayni olamaz': 'tr.same',
  'bu kategori icin zaten butce var': 'bud.exists',
  'bildirimler sunucuda kapali': 'push.server_off',
  network: 'off.needs_net',
}

export function errorText(err) {
  const msg = err?.message || ''
  if (err?.status === 429) return t('err.locked', { n: Math.max(1, Math.ceil((err.data?.retry_after || 300) / 60)) })
  if (err?.data?.attempts_left !== undefined && err.data.attempts_left > 0) {
    const base = serverErrors[msg] ? t(serverErrors[msg]) : msg
    return `${base} · ${t('err.attempts_left', { n: err.data.attempts_left })}`
  }
  if (serverErrors[msg]) return t(serverErrors[msg])
  if (msg.startsWith('kullanici adi 3-32')) return t('err.username')
  if (err?.status === 404) return t('err.not_found')
  if (err?.status >= 400 && err?.status < 500 && msg && prefs.lang === 'tr') return msg
  return t('err.generic')
}

const loc = () => (prefs.lang === 'en' ? 'en-US' : 'tr-TR')

export function monthLabel(d) {
  const s = new Intl.DateTimeFormat(loc(), { month: 'long', year: 'numeric' }).format(d)
  return s.charAt(0).toUpperCase() + s.slice(1)
}
export const monthName = (d) => {
  const s = new Intl.DateTimeFormat(loc(), { month: 'long' }).format(d)
  return s.charAt(0).toUpperCase() + s.slice(1)
}

const sameDay = (a, b) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()

export function dayLabel(date) {
  const d = new Date(date)
  const now = new Date()
  const y = new Date(now)
  y.setDate(now.getDate() - 1)
  if (sameDay(d, now)) return t('common.today')
  if (sameDay(d, y)) return t('common.yesterday')
  return new Intl.DateTimeFormat(loc(), { day: 'numeric', month: 'short' }).format(d)
}

export function dayHeader(date) {
  const d = new Date(date)
  const s = new Intl.DateTimeFormat(loc(), { day: 'numeric', month: 'long', weekday: 'long' }).format(d)
  return s.charAt(0).toUpperCase() + s.slice(1)
}

export const timeLabel = (date) => new Intl.DateTimeFormat(loc(), { hour: '2-digit', minute: '2-digit' }).format(new Date(date))
const p2 = (n) => String(n).padStart(2, '0')
export function shortDate(date) {
  const d = new Date(date)
  return prefs.lang === 'en' ? `${p2(d.getMonth() + 1)}/${p2(d.getDate())}` : `${p2(d.getDate())}.${p2(d.getMonth() + 1)}`
}
export function longDate(date) {
  const d = new Date(date)
  return prefs.lang === 'en' ? `${p2(d.getMonth() + 1)}/${p2(d.getDate())}/${d.getFullYear()}` : `${p2(d.getDate())}.${p2(d.getMonth() + 1)}.${d.getFullYear()}`
}
export const fullDate = (date) => new Intl.DateTimeFormat(loc(), { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(date))
