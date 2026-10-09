package receipts

import "strings"

// TR IBAN'da 5-9. haneler banka kodudur (EFT kodu).
var bankCodes = map[string]string{
	"00010": "Ziraat Bankası",
	"00012": "Halkbank",
	"00015": "VakıfBank",
	"00032": "TEB",
	"00046": "Akbank",
	"00059": "Şekerbank",
	"00062": "Garanti BBVA",
	"00064": "İş Bankası",
	"00067": "Yapı Kredi",
	"00092": "Citibank",
	"00099": "ING",
	"00103": "Fibabanka",
	"00111": "QNB",
	"00123": "HSBC",
	"00124": "Alternatif Bank",
	"00125": "Burgan Bank",
	"00134": "DenizBank",
	"00135": "Anadolubank",
	"00143": "Aktif Bank",
	"00146": "Odeabank",
	"00203": "Albaraka Türk",
	"00205": "Kuveyt Türk",
	"00206": "Türkiye Finans",
	"00209": "Ziraat Katılım",
	"00210": "Vakıf Katılım",
	"00211": "Emlak Katılım",
}

// BankByIBAN IBAN'daki banka kodundan banka adını bulur.
func BankByIBAN(iban string) string {
	if len(iban) < 9 || !strings.HasPrefix(iban, "TR") {
		return ""
	}
	return bankCodes[iban[4:9]]
}

// Metinde banka adı arama; daha özel olanlar önce (Ziraat Katılım > Ziraat).
var bankKeywords = []struct{ key, name string }{
	{"ziraat katilim", "Ziraat Katılım"},
	{"vakif katilim", "Vakıf Katılım"},
	{"emlak katilim", "Emlak Katılım"},
	{"ziraat", "Ziraat Bankası"},
	{"halkbank", "Halkbank"},
	{"halk bankasi", "Halkbank"},
	{"vakifbank", "VakıfBank"},
	{"vakiflar bankasi", "VakıfBank"},
	{"turk ekonomi bankasi", "TEB"},
	{"teb ", "TEB"},
	{"akbank", "Akbank"},
	{"sekerbank", "Şekerbank"},
	{"garanti", "Garanti BBVA"},
	{"is bankasi", "İş Bankası"},
	{"isbank", "İş Bankası"},
	{"yapi kredi", "Yapı Kredi"},
	{"yapi ve kredi", "Yapı Kredi"},
	{"ing bank", "ING"},
	{"fibabanka", "Fibabanka"},
	{"enpara", "Enpara (QNB)"},
	{"qnb", "QNB"},
	{"finansbank", "QNB"},
	{"hsbc", "HSBC"},
	{"alternatif bank", "Alternatif Bank"},
	{"burgan", "Burgan Bank"},
	{"denizbank", "DenizBank"},
	{"anadolubank", "Anadolubank"},
	{"aktif bank", "Aktif Bank"},
	{"odeabank", "Odeabank"},
	{"albaraka", "Albaraka Türk"},
	{"kuveyt turk", "Kuveyt Türk"},
	{"turkiye finans", "Türkiye Finans"},
	{"papara", "Papara"},
	{"ininal", "ininal"},
	// KKTC bankaları
	{"creditwest", "Creditwest Bank"},
	{"near east bank", "Near East Bank"},
	{"koop bank", "Koopbank"},
	{"koopbank", "Koopbank"},
	{"kibris vakiflar", "Kıbrıs Vakıflar Bankası"},
	{"kibris iktisat", "Kıbrıs İktisat Bankası"},
	{"iktisat bankasi", "Kıbrıs İktisat Bankası"},
	{"turkish bank", "Turkish Bank"},
	{"limasol", "Limasol Türk Kooperatif Bankası"},
	{"universal bank", "Universal Bank"},
	{"nova bank", "Nova Bank"},
	{"asbank", "Asbank"},
	{"capital bank", "Capital Bank"},
}

// detectBank katlanmış (fold) metinde en önce geçen banka adını döner;
// dekontu düzenleyen banka genelde başlıktadır.
func detectBank(folded string) string {
	best, bestAt := "", -1
	for _, k := range bankKeywords {
		i := strings.Index(folded, k.key)
		if i >= 0 && (bestAt < 0 || i < bestAt) {
			best, bestAt = k.name, i
		}
	}
	return best
}
