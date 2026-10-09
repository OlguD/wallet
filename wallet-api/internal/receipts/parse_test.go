package receipts

import "testing"

// Örnek metinler gerçek dekont düzenlerine benzer şekilde elle yazılmıştır
// (kişi ve hesaplar uydurmadır; IBAN'lar mod-97 geçerlidir).

const garantiFAST = `
GARANTİ BBVA
FAST GİDEN TRANSFER DEKONTU
İşlem Tarihi : 09.10.2026 14:32:11
Gönderen Adı Soyadı : OLGU DEĞİRMENCİ
Gönderen IBAN : TR66 0006 2000 0012 3456 7890 12
Alıcı Adı Soyadı : AYŞE IŞIK
Alıcı IBAN : TR27 0001 0000 0987 6543 2100 01
Alıcı Banka : T.C. ZİRAAT BANKASI A.Ş.
Tutar : 1.250,50 TL
İşlem Masrafı : 0,00 TL
Açıklama : Ekim kirası
Sorgu No : 8812345
`

// Etiket ve değer ayrı satırlarda (pdf.js çıktısında sık görülür), maskeli gönderen IBAN.
const isbankEFT = `
Türkiye İş Bankası A.Ş.
EFT Dekontu
Gönderen
MEHMET KAYA
Gönderen Hesap
TR12 0006 4000 **** **** **** 45
Alıcı
ELEKTRİK DAĞITIM A.Ş.
Alıcı IBAN
TR550004600000000001234567
İşlem Tutarı
TL 3.480,00
BSMV
TL 1,20
İşlem Tarihi
2026-10-01 09:05
Açıklama
Fatura ödemesi
Referans No
REF-2026-77
`

const incoming = `
Akbank
GELEN HAVALE
Tarih: 05/10/2026
Gönderen: ALİ VELİ
Gönderen IBAN: TR75 0006 4000 0111 2223 3344 45
Alıcı: OLGU DEĞİRMENCİ
Tutar: 500 TL
`

func TestParseGarantiFAST(t *testing.T) {
	p := Parse(garantiFAST)
	check(t, "type", p.Type, "expense")
	check(t, "transfer", p.TransferType, "FAST")
	check(t, "bank", p.Bank, "Garanti BBVA")
	check(t, "sender", p.SenderName, "Olgu Değirmenci")
	check(t, "sender iban", p.SenderIBAN, "TR660006200000123456789012")
	check(t, "cp name", p.CounterpartyName, "Ayşe Işık")
	check(t, "cp iban", p.CounterpartyIBAN, "TR270001000009876543210001")
	check(t, "cp bank", p.CounterpartyBank, "Ziraat Bankası")
	check(t, "date", p.Date, "2026-10-09")
	check(t, "time", p.Time, "14:32")
	check(t, "desc", p.Description, "Ekim kirası")
	check(t, "ref", p.Reference, "8812345")
	check(t, "currency", p.Currency, "TRY")
	if p.Amount != 125050 {
		t.Errorf("amount = %d, want 125050", p.Amount)
	}
}

func TestParseSplitLinesAndMasked(t *testing.T) {
	p := Parse(isbankEFT)
	check(t, "transfer", p.TransferType, "EFT")
	check(t, "bank", p.Bank, "İş Bankası")
	check(t, "sender", p.SenderName, "Mehmet Kaya")
	check(t, "sender iban", p.SenderIBAN, "TR1200064000************45")
	check(t, "cp name", p.CounterpartyName, "Elektrik Dağıtım A.Ş.")
	check(t, "cp iban", p.CounterpartyIBAN, "TR550004600000000001234567")
	check(t, "cp bank", p.CounterpartyBank, "Akbank")
	check(t, "date", p.Date, "2026-10-01")
	check(t, "time", p.Time, "09:05")
	check(t, "desc", p.Description, "Fatura ödemesi")
	check(t, "ref", p.Reference, "REF-2026-77")
	if p.Amount != 348000 {
		t.Errorf("amount = %d, want 348000 (BSMV satırı alınmamalı)", p.Amount)
	}
}

func TestParseIncoming(t *testing.T) {
	p := Parse(incoming)
	check(t, "type", p.Type, "income")
	check(t, "transfer", p.TransferType, "Havale")
	check(t, "cp name", p.CounterpartyName, "Ali Veli")
	check(t, "cp iban", p.CounterpartyIBAN, "TR750006400001112223334445")
	check(t, "cp bank", p.CounterpartyBank, "İş Bankası")
	check(t, "date", p.Date, "2026-10-05")
	if p.Amount != 50000 {
		t.Errorf("amount = %d, want 50000", p.Amount)
	}
}

// pdf.js tablo düzeni: sütunlar " | " ile ayrılmış, iki alan aynı satırda.
const columns = `
YAPI KREDİ
FAST Ödeme Dekontu
Gönderen | AHMET DEMİR | Alıcı | ZEYNEP KARA
Alıcı IBAN | TR66 0006 2000 0012 3456 7890 12 | İşlem Tarihi | 03.10.2026
Tutar | 75,00 TL | Masraf | 0,00 TL
`

func TestParseColumns(t *testing.T) {
	p := Parse(columns)
	check(t, "bank", p.Bank, "Yapı Kredi")
	check(t, "sender", p.SenderName, "Ahmet Demir")
	check(t, "cp name", p.CounterpartyName, "Zeynep Kara")
	check(t, "cp iban", p.CounterpartyIBAN, "TR660006200000123456789012")
	check(t, "cp bank", p.CounterpartyBank, "Garanti BBVA")
	check(t, "date", p.Date, "2026-10-03")
	if p.Amount != 7500 {
		t.Errorf("amount = %d, want 7500", p.Amount)
	}
}

func TestMoney(t *testing.T) {
	cases := map[string]int64{"1.234,56": 123456, "1,234.56": 123456, "1.234": 123400, "250": 25000, "12,5": 1250, "1 250,00": 125000}
	for in, want := range cases {
		if got, ok := moneyToKurus(in); !ok || got != want {
			t.Errorf("moneyToKurus(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestValidIBAN(t *testing.T) {
	if !ValidIBAN("TR660006200000123456789012") {
		t.Error("valid IBAN rejected")
	}
	if ValidIBAN("TR660006200000123456789013") {
		t.Error("invalid IBAN accepted")
	}
}

func check(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}
