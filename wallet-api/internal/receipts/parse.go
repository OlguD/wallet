// Package receipts banka dekontlarını (EFT/FAST/havale) saklar ve okur.
// Dosyanın metni istemcide çıkarılır (PDF: pdf.js, görüntü: OCR); burada
// metinden alıcı, IBAN, banka, tutar, tarih gibi alanlar kurallarla bulunur.
// Hiçbir veri üçüncü bir servise gönderilmez.
package receipts

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Parsed dekonttan okunan alanlar; bulunamayanlar boş kalır.
type Parsed struct {
	// Type: expense (giden) veya income (gelen).
	Type             string `json:"type"`
	Amount           int64  `json:"amount,omitempty"` // kuruş
	Currency         string `json:"currency,omitempty"`
	Date             string `json:"date,omitempty"` // YYYY-AA-GG
	Time             string `json:"time,omitempty"` // SS:DD
	CounterpartyName string `json:"counterparty_name,omitempty"`
	CounterpartyIBAN string `json:"counterparty_iban,omitempty"`
	CounterpartyBank string `json:"counterparty_bank,omitempty"`
	SenderName       string `json:"sender_name,omitempty"`
	SenderIBAN       string `json:"sender_iban,omitempty"`
	RecipientName    string `json:"recipient_name,omitempty"`
	RecipientIBAN    string `json:"recipient_iban,omitempty"`
	Bank             string `json:"bank,omitempty"` // dekontu düzenleyen banka
	Description      string `json:"description,omitempty"`
	Reference        string `json:"reference,omitempty"`
	TransferType     string `json:"transfer_type,omitempty"` // FAST, EFT, Havale, SWIFT
}

// fold karşılaştırma için Türkçe harfleri ASCII küçük harfe indirir.
// Her rune tek rune'a eşlenir; böylece indeksler orijinal metinle aynı kalır.
func fold(s string) []rune {
	out := []rune(s)
	for i, r := range out {
		switch r {
		case 'İ', 'I', 'ı':
			r = 'i'
		case 'Ğ', 'ğ':
			r = 'g'
		case 'Ü', 'ü':
			r = 'u'
		case 'Ş', 'ş':
			r = 's'
		case 'Ö', 'ö':
			r = 'o'
		case 'Ç', 'ç':
			r = 'c'
		case 'Â', 'â':
			r = 'a'
		default:
			r = unicode.ToLower(r)
		}
		out[i] = r
	}
	return out
}

func foldStr(s string) string { return string(fold(s)) }

var spaceRe = regexp.MustCompile(`[ \t\x{00A0}]+`)

func lines(text string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n") {
		l = strings.TrimSpace(spaceRe.ReplaceAllString(l, " "))
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

const sepChars = " :;-–—=>/|\t*•"

// field bir etiketin değerini arar: "Etiket : değer" veya etiket satırı
// boşsa sonraki satır. skip: etiketten hemen sonra gelirse eşleşmeyi
// geçersiz kılan kelimeler (örn. "alıcı" + "iban").
type field struct {
	labels []string
	skip   []string
	reject []string // satırda geçerse yok say (masraf, komisyon...)
}

type hit struct {
	value string
	line  int
}

func (f field) find(ls []string) []hit {
	var hits []hit
	for i, l := range ls {
		fl := fold(l)
		for _, label := range f.labels {
			lr := []rune(label)
			pos := runeIndex(fl, lr)
			// Etiket satır başında ya da önünde ayraç olmalı ("... | Alıcı: X").
			if pos < 0 || (pos > 0 && !strings.ContainsRune(sepChars, fl[pos-1])) {
				continue
			}
			end := pos + len(lr)
			// Kelimenin devamı olmasın ("tutar" ≠ "tutarı" hariç tutulur, aşağıda ayrıca var).
			if end < len(fl) && unicode.IsLetter(fl[end]) {
				continue
			}
			rest := strings.TrimLeft(string([]rune(l)[end:]), sepChars)
			// pdf.js sütun aralarını " | " ile birleştirir: değer sonraki sütunda biter.
			if j := strings.Index(rest, "|"); j >= 0 {
				rest = strings.TrimSpace(rest[:j])
			}
			if hasPrefixWord(foldStr(rest), f.skip) {
				continue
			}
			if rest == "" && i+1 < len(ls) && !looksLikeLabel(ls[i+1]) {
				rest = ls[i+1]
			}
			// Red kelimeleri (masraf, BSMV...) sadece bu etiketin sütununda aranır.
			seg := pos
			for seg > 0 && fl[seg-1] != '|' {
				seg--
			}
			if containsAny(string(fl[seg:end])+" "+foldStr(rest), f.reject) {
				continue
			}
			if rest != "" {
				hits = append(hits, hit{rest, i})
			}
			break
		}
	}
	return hits
}

func runeIndex(s, sub []rune) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if string(s[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func hasPrefixWord(s string, words []string) bool {
	for _, w := range words {
		if strings.HasPrefix(s, w) {
			return true
		}
	}
	return false
}

// looksLikeLabel: "Alıcı IBAN:" gibi kendisi etiket olan satır.
func looksLikeLabel(l string) bool {
	return strings.HasSuffix(strings.TrimSpace(l), ":")
}

var (
	nameSkip = []string{"iban", "banka", "hesap", "sube", "tc", "t.c", "vkn", "vergi", "no", "numara", "adres", "il ", "kimlik", "musteri no", "telefon"}

	recipientName = field{
		labels: []string{
			"alici adi soyadi/unvani", "alici adi soyadi / unvani", "alici ad soyad/unvan", "alici adi/unvani", "alici ad/unvan",
			"alici adi soyadi", "alici ad soyad", "alici unvani", "alici adi", "alici hesap sahibi", "alici",
			"alacakli adi soyadi", "alacakli adi", "alacakli", "lehtar adi soyadi", "lehtar adi", "lehtar",
			"gonderilen kisi", "karsi taraf adi", "karsi taraf", "kime",
		},
		skip: nameSkip,
	}
	senderName = field{
		labels: []string{
			"gonderen adi soyadi/unvani", "gonderen adi soyadi", "gonderen ad soyad", "gonderen adi/unvani", "gonderen unvani",
			"gonderen adi", "gonderen", "gonderici hesap", "gonderici adi", "gonderici", "borclu adi soyadi", "borclu adi", "borclu", "hesap sahibi adi soyadi", "hesap sahibi",
			"musteri adi soyadi", "musteri adi", "musteri unvani", "kimden",
		},
		skip: nameSkip,
	}
	// Bazı bankalar (İş, Garanti) alıcı adını "Alıcı Hesap" / "Alacaklı Hesap"
	// satırına hesap numarasının arkasına yazar: "00451 / 6873376 AHMET YILMAZ".
	recipientAccountName = field{labels: []string{"alici hesap", "alacakli hesap"}}
	recipientIBAN        = field{labels: []string{"alici iban", "alici hesap iban", "alici hesap no/iban", "alacakli iban", "lehtar iban", "alici hesap", "karsi iban"}}
	senderIBAN           = field{labels: []string{"gonderen iban", "gonderen hesap iban", "borclu iban", "gonderen hesap", "hesap iban", "borclu hesap"}}
	recipientBank        = field{labels: []string{"alici banka adi", "alici bankasi", "alici banka", "alacakli banka", "lehtar banka", "karsi banka"}, skip: []string{"kodu", "sube"}}
	amountField          = field{
		labels: []string{
			"islem tutari", "transfer tutari", "gonderilen tutar", "havale tutari", "eft tutari", "fast tutari",
			"odeme tutari", "odenen tutar", "toplam tutar", "tutar", "tutari", "miktar",
		},
		reject: []string{"masraf", "ucret", "komisyon", "bsmv", "vergi", "limit", "bakiye", "kesinti"},
	}
	dateField = field{labels: []string{"islem tarihi ve saati", "islem tarihi/saati", "islem tarihi", "islem zamani", "tarih/saat", "tarih", "duzenleme tarihi", "valor"}}
	descField = field{
		labels: []string{"islem aciklamasi", "odeme aciklamasi", "transfer aciklamasi", "aciklama", "aciklamasi", "odeme amaci"},
	}
	refField = field{labels: []string{"islem referans no", "referans numarasi", "referans no", "referans", "sorgu numarasi", "sorgu no", "islem no", "islem numarasi", "dekont no", "fis no", "islem sira no"}}
)

// IBAN: TR + 24 hane; maskeli (****) olabilir, boşluklu yazılabilir.
var ibanRe = regexp.MustCompile(`(?i)TR\s?[0-9]{2}(?:\s?[0-9*xX•]){22}`)

func findIBAN(s string) string {
	m := ibanRe.FindString(s)
	if m == "" {
		return ""
	}
	return strings.ToUpper(strings.Map(func(r rune) rune {
		if r == ' ' {
			return -1
		}
		if r == '•' || r == 'x' || r == 'X' {
			return '*'
		}
		return r
	}, m))
}

// ValidIBAN tam (maskesiz) bir TR IBAN'ın mod-97 kontrolü.
func ValidIBAN(iban string) bool {
	if len(iban) != 26 || !strings.HasPrefix(iban, "TR") {
		return false
	}
	re := iban[4:] + iban[:4]
	var sb strings.Builder
	for _, r := range re {
		switch {
		case r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			sb.WriteString(strconv.Itoa(int(r-'A') + 10))
		default:
			return false
		}
	}
	n, ok := new(big.Int).SetString(sb.String(), 10)
	return ok && new(big.Int).Mod(n, big.NewInt(97)).Int64() == 1
}

// Para: "1.234,56 TL", "TL 1.234,56", "1,234.56 TRY", "₺250", "250,00".
var moneyRe = regexp.MustCompile(`(?i)(TL|TRY|₺|USD|EUR|GBP|\$|€|£)?\s*([0-9]{1,3}(?:[.,' ][0-9]{3})+(?:[.,][0-9]{1,2})?|[0-9]+(?:[.,][0-9]{1,2})?)\s*(TL|TRY|₺|USD|EUR|GBP|\$|€|£)?`)

var currencyCodes = map[string]string{"TL": "TRY", "TRY": "TRY", "₺": "TRY", "USD": "USD", "$": "USD", "EUR": "EUR", "€": "EUR", "GBP": "GBP", "£": "GBP"}

// parseMoney bir metindeki ilk anlamlı tutarı kuruş olarak döner.
// needCurrency: etiketsiz aramada para birimi işareti zorunlu.
func parseMoney(s string, needCurrency bool) (int64, string, bool) {
	for _, m := range moneyRe.FindAllStringSubmatch(s, -1) {
		cur := strings.ToUpper(m[1])
		if cur == "" {
			cur = strings.ToUpper(m[3])
		}
		if needCurrency && cur == "" {
			continue
		}
		k, ok := moneyToKurus(m[2])
		if !ok || k <= 0 {
			continue
		}
		return k, currencyCodes[cur], true
	}
	return 0, "", false
}

func moneyToKurus(s string) (int64, bool) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "'", "")
	lastDot, lastComma := strings.LastIndex(s, "."), strings.LastIndex(s, ",")
	dec := -1
	if i := max(lastDot, lastComma); i >= 0 && len(s)-i-1 <= 2 {
		dec = i // son ayraçtan sonra 1-2 hane: ondalık
	}
	var intPart, frac string
	if dec >= 0 {
		intPart, frac = s[:dec], s[dec+1:]
	} else {
		intPart = s
	}
	intPart = strings.NewReplacer(".", "", ",", "").Replace(intPart)
	frac = (frac + "00")[:2]
	i, err1 := strconv.ParseInt(intPart, 10, 64)
	f, err2 := strconv.ParseInt(frac, 10, 64)
	if err1 != nil || err2 != nil || i > 10_000_000_000 {
		return 0, false
	}
	return i*100 + f, true
}

var (
	dmyRe = regexp.MustCompile(`\b([0-3]?[0-9])[./-]([01]?[0-9])[./-](20[0-9]{2})(?:[ T,/-]+([0-2]?[0-9])[:.]([0-5][0-9])(?:[:.]([0-5][0-9]))?)?`)
	ymdRe = regexp.MustCompile(`\b(20[0-9]{2})-([01][0-9])-([0-3][0-9])(?:[ T]([0-2][0-9]):([0-5][0-9]))?`)
)

func parseDate(s string) (date, clock string, ok bool) {
	var y, m, d, hh, mm string
	if g := dmyRe.FindStringSubmatch(s); g != nil {
		d, m, y, hh, mm = g[1], g[2], g[3], g[4], g[5]
	} else if g := ymdRe.FindStringSubmatch(s); g != nil {
		y, m, d, hh, mm = g[1], g[2], g[3], g[4], g[5]
	} else {
		return "", "", false
	}
	yi, _ := strconv.Atoi(y)
	mi, _ := strconv.Atoi(m)
	di, _ := strconv.Atoi(d)
	t := time.Date(yi, time.Month(mi), di, 0, 0, 0, 0, time.UTC)
	if t.Day() != di || int(t.Month()) != mi {
		return "", "", false
	}
	date = t.Format("2006-01-02")
	if hh != "" {
		h, _ := strconv.Atoi(hh)
		if h < 24 {
			clock = strconv.Itoa(100 + h)[1:] + ":" + mm
		}
	}
	return date, clock, true
}

var letterRe = regexp.MustCompile(`\p{L}`)

// accountNoRe hesap numarası önekini atar: "00451 / 6873376 " → "".
var accountNoRe = regexp.MustCompile(`^[0-9 /*.-]+`)

// cleanName dekonttaki isim değerini sadeleştirir; isim değilse "".
func cleanName(s string) string {
	if i := ibanRe.FindStringIndex(s); i != nil {
		s = s[:i[0]]
	}
	// Aynı satırda başka bir etiket devam ediyorsa kes ("AHMET YILMAZ  IBAN: ...").
	fs := foldStr(s)
	for _, cut := range []string{" iban", " hesap no", " banka", " tc kimlik", " vkn", " tutar", " tarih", " sube", " alici", " alacakli"} {
		if i := strings.Index(fs, cut); i > 0 {
			s = string([]rune(s)[:len([]rune(fs[:i]))])
			fs = foldStr(s)
		}
	}
	s = strings.Trim(s, sepChars+",")
	if utf8Len(s) < 2 || utf8Len(s) > 80 || !letterRe.MatchString(s) {
		return ""
	}
	return titleTR(s)
}

func utf8Len(s string) int { return len([]rune(s)) }

// titleTR TAMAMI BÜYÜK isimleri Türkçe kurallarıyla "Ahmet Yılmaz" yapar.
// Şirket kısaltmaları (A.Ş., LTD., ŞTİ.) korunur; zaten karışık yazılmışsa dokunmaz.
func titleTR(s string) string {
	if strings.ToUpper(s) != s && strings.ToLower(s) != s {
		return s
	}
	keep := map[string]bool{"A.Ş.": true, "AŞ": true, "A.S.": true, "LTD.": true, "LTD": true, "ŞTİ.": true, "ŞTİ": true, "STİ.": true, "TİC.": true, "SAN.": true, "VE": false}
	words := strings.Fields(s)
	for i, w := range words {
		if keep[w] {
			continue
		}
		rs := []rune(w)
		for j, r := range rs {
			if j == 0 {
				rs[j] = upperTR(r)
			} else {
				rs[j] = lowerTR(r)
			}
		}
		words[i] = string(rs)
	}
	return strings.Join(words, " ")
}

func upperTR(r rune) rune {
	switch r {
	case 'i':
		return 'İ'
	case 'ı':
		return 'I'
	}
	return unicode.ToUpper(r)
}

func lowerTR(r rune) rune {
	switch r {
	case 'I':
		return 'ı'
	case 'İ':
		return 'i'
	}
	return unicode.ToLower(r)
}

// Parse dekont metnini okur.
func Parse(text string) Parsed {
	ls := lines(text)
	all := strings.Join(ls, "\n")
	fall := foldStr(all)
	p := Parsed{Type: "expense"}

	// Yön: gelen transfer dekontları.
	if containsAny(fall, []string{"gelen eft", "gelen fast", "gelen havale", "hesabiniza gelen", "gelen transfer", "alacak dekontu"}) {
		p.Type = "income"
	}
	switch {
	case strings.Contains(fall, "swift"):
		p.TransferType = "SWIFT"
	case regexp.MustCompile(`\bfast\b`).MatchString(fall):
		p.TransferType = "FAST"
	case regexp.MustCompile(`\beft\b`).MatchString(fall):
		p.TransferType = "EFT"
	case strings.Contains(fall, "havale"):
		p.TransferType = "Havale"
	}

	for _, h := range recipientName.find(ls) {
		if n := cleanName(h.value); n != "" {
			p.RecipientName = n
			break
		}
	}
	if p.RecipientName == "" {
		for _, h := range recipientAccountName.find(ls) {
			if n := cleanName(accountNoRe.ReplaceAllString(h.value, "")); n != "" {
				p.RecipientName = n
				break
			}
		}
	}
	for _, h := range senderName.find(ls) {
		if n := cleanName(h.value); n != "" {
			p.SenderName = n
			break
		}
	}

	// IBAN'lar: önce etiketle, sonra metindeki sıraya göre (genelde önce gönderen).
	for _, h := range recipientIBAN.find(ls) {
		if v := findIBAN(h.value); v != "" {
			p.RecipientIBAN = v
			break
		}
	}
	for _, h := range senderIBAN.find(ls) {
		if v := findIBAN(h.value); v != "" && v != p.RecipientIBAN {
			p.SenderIBAN = v
			break
		}
	}
	// Etiketsiz IBAN'lar: alıcı/gönderen adının geçtiği satıra yakınlıkla eşle.
	var loose []string
	for _, m := range ibanRe.FindAllString(all, -1) {
		v := findIBAN(m)
		if v != p.RecipientIBAN && v != p.SenderIBAN && !contains(loose, v) {
			loose = append(loose, v)
		}
	}
	for _, v := range loose {
		switch {
		case p.RecipientIBAN == "" && (p.SenderIBAN != "" || len(loose) == 1 && !strings.Contains(v, "*")):
			p.RecipientIBAN = v
		case p.SenderIBAN == "" && p.RecipientIBAN != "":
			p.SenderIBAN = v
		case p.SenderIBAN == "":
			p.SenderIBAN = v
		}
	}

	for _, h := range recipientBank.find(ls) {
		if b := cleanBank(h.value); b != "" {
			p.CounterpartyBank = b
			break
		}
	}

	for _, h := range amountField.find(ls) {
		if k, cur, ok := parseMoney(h.value, false); ok {
			p.Amount, p.Currency = k, cur
			break
		}
	}
	if p.Amount == 0 {
		for _, l := range ls {
			if containsAny(foldStr(l), amountField.reject) {
				continue
			}
			if k, cur, ok := parseMoney(l, true); ok && !ibanRe.MatchString(l) {
				p.Amount, p.Currency = k, cur
				break
			}
		}
	}
	if p.Amount > 0 && p.Currency == "" {
		p.Currency = "TRY"
	}

	for _, h := range dateField.find(ls) {
		if d, c, ok := parseDate(h.value); ok {
			p.Date, p.Time = d, c
			break
		}
	}
	if p.Date == "" {
		if d, c, ok := parseDate(all); ok {
			p.Date, p.Time = d, c
		}
	}

	for _, h := range descField.find(ls) {
		v := strings.Trim(h.value, sepChars)
		if v != "" && !looksLikeLabel(v) {
			p.Description = truncate(v, 200)
			break
		}
	}
	for _, h := range refField.find(ls) {
		if f := strings.Fields(h.value); len(f) > 0 {
			p.Reference = truncate(strings.Trim(f[0], sepChars), 64)
			break
		}
	}

	p.Bank = detectBank(fall)
	if p.Bank == "" && p.SenderIBAN != "" {
		p.Bank = BankByIBAN(p.SenderIBAN)
	}

	// Karşı taraf: giderde alıcı, gelirde gönderen.
	if p.Type == "income" {
		p.CounterpartyName, p.CounterpartyIBAN = p.SenderName, p.SenderIBAN
		p.CounterpartyBank = ""
	} else {
		p.CounterpartyName, p.CounterpartyIBAN = p.RecipientName, p.RecipientIBAN
	}
	if p.CounterpartyBank == "" && p.CounterpartyIBAN != "" {
		p.CounterpartyBank = BankByIBAN(p.CounterpartyIBAN)
	}
	return p
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func cleanBank(s string) string {
	if b := detectBank(foldStr(s)); b != "" {
		return b
	}
	s = strings.Trim(s, sepChars)
	if utf8Len(s) < 3 || utf8Len(s) > 64 || !letterRe.MatchString(s) {
		return ""
	}
	return titleTR(s)
}
