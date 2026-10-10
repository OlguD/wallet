package receipts

import "testing"

// Gerçek İş Bankası / Ziraat / Garanti dekont düzenlerinden (2026-10-10, kullanıcının
// paylaştığı örnekler) OCR çıktısına benzer metinler. İki sütunlu düzende OCR
// satırları yan yana birleştirir. Kişi, hesap ve numaralar uydurmadır.

const isbankParaAktarma = `
TÜRKİYE İŞ BANKASI DEKONT TÜRKİYE İŞ BANKASI A.Ş. Genel Müdürlük İstanbul
Müşteri Bilgisi İşlem Bilgisi
KEREM YALIN İşlem Yeri : MOBİL BANKACILIK
İşlem Tarihi : 27.03.2026 / 16:35:12
Müşteri No : 111222333 Referans Numarası : 27.03.2026/447/4217/1
TCKN : 1234567**** Düzenleme Tarihi : 27.03.2026 / 16:35:44
İşlem Özeti
Para Aktarma
Gönderici Hesap : KEREM YALIN Alıcı Hesap : SELİN ÖZTÜRK
TR91 0006 4000 0011 1111 1111 11 TR34 0006 2000 2222 2222 2222 22
YENİMAHALLE/ANKARA T.GARANTİ BANKASI A.Ş.-İBAN MERKEZ SUBESİ
Aktarılan Tutar : 34.000,00 TRY Sorgu Numarası : 0128876
Hesaptan Çekilen : 34.000,00 TRY İşlem Türü : KONUT KİRASI
Açıklama : NİSAN 2026 KİRA BEDELİ
İşleminiz gerçekleştirilmiştir.
`

const ziraatHavale = `
Ziraat Bankası Hesaptan Hesaba Havale
ŞUBE KODU/ADI : 4000/ZİRAAT SÜPER ŞUBE SAYIN
IBAN : TR82 0001 0040 0333 3333 3333 33 CAN DEMİRCİ
HESAP NUMARASI : 4000/33333333-5009 ÖRNEK KÖYÜ NO: 1
VERGİ DAİRESİ : 17200 BİGA ÇANAKKALE
VERGİ KİMLİK NO : 11111111111
İŞLEM TARİHİ : 08/11/2026-22:38:43 - F35544_4003
VALÖR : 08.11.2026
İŞLEM YERİ : ZİRAAT MOBİL
Açıklama : malzeme ücreti
Alacaklı Şube : 1745-ANKARA KAMU KURUMSAL ŞUBE
Alacaklı Hesap : 44444444 5013
Alacaklı IBAN : TR20 0001 0017 4444 4444 4444 44
Alacaklı Adı Soyadı : ÖRNEK TEKNOLOJİ ANONİM ŞİRKETİ
Alacaklı Vergi No : 2222222222
Komisyon : 0,00 TRY
Havale Tutarı : 143,00 TRY
Hesabınızdan 143,00 TL (Yalnız YÜZKIRKÜÇTL) Çekilmiştir.
08/11/2026-22:38:46 INTTHVLD MOBİL
`

const garantiHavale = `
Garanti BBVA Dekont T. Garanti Bankası A.Ş.
HESAPTAN HESABA HAVALE
ŞUBE ADI : İBRAHİMAĞA/ÜSKÜDAR SAYIN
MÜŞTERİ NUMARASI : 55555555 DENİZ ARSLAN
HESAP NUMARASI : 1020/5555555 ÖRNEK MAH. 12. SOK. 4
İŞLEM TARİHİ : 22/11/2026
TC KİMLİK NO : 22222222222 41420 ÇAYIROVA/KOCAELİ
İŞLEM YERİ : MOBİL
DÜZENLEME TARİHİ: 28.11.2026
IBAN: TR94 0006 2001 0200 0055 5555 55
DENİZ ARSLAN SPOR SALONU
ALACAKLI ŞUBE : ŞANLIURFA
ALACAKLI HESAP : 00451 / 6666666 MERT KOÇAK
ALACAKLI IBAN : TR05 0006 2000 4510 0066 6666 66
İŞLEM MEDAI : 0120 / 5555555 IBAN:TR94 0006 2001 0200 0055 5555 55
MASRAF TUTARI : 1,55 TL
BSMV : 0,08 TL
MASRAF TOPLAMI : 1,63 TL
YALNIZ BinDörtYüzElliBeşTL.
SIRA NO : 2026-11-22-16.38.54.589342 TUTAR : - 1.455,00 TL
`

func TestParseIsbankParaAktarma(t *testing.T) {
	p := Parse(isbankParaAktarma)
	check(t, "type", p.Type, "expense")
	check(t, "bank", p.Bank, "İş Bankası")
	if p.Amount != 3400000 {
		t.Errorf("amount = %d, want 3400000", p.Amount)
	}
	check(t, "currency", p.Currency, "TRY")
	check(t, "date", p.Date, "2026-03-27")
	check(t, "time", p.Time, "16:35")
	check(t, "recipient", p.RecipientName, "Selin Öztürk")
	check(t, "sender", p.SenderName, "Kerem Yalın")
	check(t, "recipient iban", p.RecipientIBAN, "TR340006200022222222222222")
	check(t, "sender iban", p.SenderIBAN, "TR910006400000111111111111")
	check(t, "counterparty bank", p.CounterpartyBank, "Garanti BBVA")
	check(t, "desc", p.Description, "NİSAN 2026 KİRA BEDELİ")
}

func TestParseZiraatHavale(t *testing.T) {
	p := Parse(ziraatHavale)
	check(t, "type", p.Type, "expense")
	check(t, "transfer", p.TransferType, "Havale")
	check(t, "bank", p.Bank, "Ziraat Bankası")
	if p.Amount != 14300 {
		t.Errorf("amount = %d, want 14300", p.Amount)
	}
	check(t, "date", p.Date, "2026-11-08")
	check(t, "time", p.Time, "22:38")
	check(t, "recipient", p.RecipientName, "Örnek Teknoloji Anonim Şirketi")
	check(t, "recipient iban", p.RecipientIBAN, "TR200001001744444444444444")
	check(t, "sender iban", p.SenderIBAN, "TR820001004003333333333333")
	check(t, "desc", p.Description, "malzeme ücreti")
}

func TestParseGarantiHavale(t *testing.T) {
	p := Parse(garantiHavale)
	check(t, "type", p.Type, "expense")
	check(t, "transfer", p.TransferType, "Havale")
	check(t, "bank", p.Bank, "Garanti BBVA")
	if p.Amount != 145500 {
		t.Errorf("amount = %d, want 145500", p.Amount)
	}
	check(t, "date", p.Date, "2026-11-22")
	check(t, "recipient", p.RecipientName, "Mert Koçak")
	check(t, "recipient iban", p.RecipientIBAN, "TR050006200045100066666666")
	check(t, "sender iban", p.SenderIBAN, "TR940006200102000055555555")
}
