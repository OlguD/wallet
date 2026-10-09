package receipts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet-api/internal/auth"
	"wallet-api/internal/httpx"
)

// MaxFileBytes yüklenebilecek en büyük dekont.
const MaxFileBytes = 10 << 20

const maxTextBytes = 64 << 10

type Handler struct {
	DB *pgxpool.Pool
	// Dir dosyaların saklandığı klasör (RECEIPTS_DIR).
	Dir string
}

type Receipt struct {
	ID            int       `json:"id"`
	MimeType      string    `json:"mime_type"`
	SizeBytes     int       `json:"size_bytes"`
	OriginalName  *string   `json:"original_name"`
	Source        string    `json:"source"`
	Status        string    `json:"status"`
	TransactionID *int      `json:"transaction_id"`
	Parsed        *Parsed   `json:"parsed"`
	HasText       bool      `json:"has_text"`
	CreatedAt     time.Time `json:"created_at"`
	// Duplicate: aynı dosya daha önce yüklenmişse true (mevcut kayıt döner).
	Duplicate bool `json:"duplicate,omitempty"`
}

const selectReceipt = `
SELECT id, mime_type, size_bytes, original_name, source, status, transaction_id, parsed, text IS NOT NULL, created_at
FROM receipts `

func scanReceipt(row pgx.CollectableRow) (Receipt, error) {
	var r Receipt
	var parsed []byte
	err := row.Scan(&r.ID, &r.MimeType, &r.SizeBytes, &r.OriginalName, &r.Source, &r.Status, &r.TransactionID,
		&parsed, &r.HasText, &r.CreatedAt)
	if err == nil && parsed != nil {
		r.Parsed = &Parsed{}
		err = json.Unmarshal(parsed, r.Parsed)
	}
	return r, err
}

var allowed = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/heic":      ".heic",
	"image/heif":      ".heif",
}

// sniff içerikten tür belirler; HEIC'i tarayıcı tanımadığı için uzantı/başlığa bakılır.
func sniff(data []byte, declared, name string) string {
	t := http.DetectContentType(data)
	if i := strings.Index(t, ";"); i >= 0 {
		t = t[:i]
	}
	if _, ok := allowed[t]; ok {
		return t
	}
	if len(data) > 12 && string(data[4:8]) == "ftyp" {
		switch string(data[8:12]) {
		case "heic", "heix", "mif1", "msf1", "heif":
			return "image/heic"
		}
	}
	ext := strings.ToLower(filepath.Ext(name))
	for m, e := range allowed {
		if e == ext || m == declared {
			if m == "application/pdf" && !strings.HasPrefix(string(data), "%PDF") {
				continue
			}
			return m
		}
	}
	return ""
}

type upload struct {
	data []byte
	mime string
	name string
	text string
}

// readUpload çok parçalı formdan ("file" ve opsiyonel "text") ya da ham
// gövdeden (iOS Kestirme "Dosya" gövdesi) dosyayı okur.
func readUpload(w http.ResponseWriter, r *http.Request) (*upload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileBytes+maxTextBytes+64<<10)
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	u := &upload{}
	if ct == "multipart/form-data" {
		if err := r.ParseMultipartForm(MaxFileBytes); err != nil {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "dosya en fazla 10 MB olabilir")
			return nil, false
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "file alani gerekli")
			return nil, false
		}
		defer f.Close()
		if u.data, err = io.ReadAll(f); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "dosya okunamadi")
			return nil, false
		}
		u.name = hdr.Filename
		u.mime = hdr.Header.Get("Content-Type")
		u.text = r.FormValue("text")
	} else {
		var err error
		if u.data, err = io.ReadAll(r.Body); err != nil {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "dosya en fazla 10 MB olabilir")
			return nil, false
		}
		u.mime = ct
		u.name = r.URL.Query().Get("name")
	}
	if len(u.data) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "dosya bos")
		return nil, false
	}
	if len(u.data) > MaxFileBytes {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "dosya en fazla 10 MB olabilir")
		return nil, false
	}
	u.mime = sniff(u.data, u.mime, u.name)
	if u.mime == "" {
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "sadece PDF veya resim (JPG, PNG, HEIC) yuklenebilir")
		return nil, false
	}
	if len(u.text) > maxTextBytes || !utf8.ValidString(u.text) {
		u.text = ""
	}
	if utf8.RuneCountInString(u.name) > 200 {
		u.name = string([]rune(u.name)[:200])
	}
	return u, true
}

func randomKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *Handler) path(key, mimeType string) string {
	return filepath.Join(h.Dir, key+allowed[mimeType])
}

// store dosyayı diske, kaydı veritabanına yazar. Aynı kullanıcı aynı dosyayı
// tekrar yüklerse yeni kayıt açılmaz, mevcut kayıt döner.
func (h *Handler) store(ctx context.Context, userID int, u *upload, source string) (int, bool, error) {
	sum := sha256.Sum256(u.data)
	digest := hex.EncodeToString(sum[:])

	var existing int
	err := h.DB.QueryRow(ctx, "SELECT id FROM receipts WHERE user_id = $1 AND sha256 = $2", userID, digest).Scan(&existing)
	if err == nil {
		return existing, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}

	key, err := randomKey()
	if err != nil {
		return 0, false, err
	}
	if err := os.MkdirAll(h.Dir, 0o750); err != nil {
		return 0, false, err
	}
	p := h.path(key, u.mime)
	if err := os.WriteFile(p, u.data, 0o640); err != nil {
		return 0, false, err
	}

	var text *string
	var parsed []byte
	if strings.TrimSpace(u.text) != "" {
		text = &u.text
		parsed, _ = json.Marshal(Parse(u.text))
	}
	var name *string
	if u.name != "" {
		name = &u.name
	}
	var id int
	err = h.DB.QueryRow(ctx, `
INSERT INTO receipts (user_id, storage_key, mime_type, size_bytes, sha256, original_name, source, text, parsed)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, sha256) DO NOTHING
RETURNING id`,
		userID, key, u.mime, len(u.data), digest, name, source, text, parsed,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Eşzamanlı aynı yükleme: diğerini kullan.
		os.Remove(p)
		err = h.DB.QueryRow(ctx, "SELECT id FROM receipts WHERE user_id = $1 AND sha256 = $2", userID, digest).Scan(&id)
		return id, true, err
	}
	if err != nil {
		os.Remove(p)
		return 0, false, err
	}
	return id, false, nil
}

func (h *Handler) writeOne(w http.ResponseWriter, r *http.Request, id, status int, dup bool) {
	rows, err := h.DB.Query(r.Context(), selectReceipt+"WHERE id = $1 AND user_id = $2", id, auth.UserID(r.Context()))
	if err != nil {
		httpx.ServerError(w, "receipt get", err)
		return
	}
	rec, err := pgx.CollectOneRow(rows, scanReceipt)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "dekont bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "receipt scan", err)
		return
	}
	rec.Duplicate = dup
	httpx.WriteJSON(w, status, rec)
}

// Upload uygulama içinden dekont yükler (file + istemcide çıkarılmış text).
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	u, ok := readUpload(w, r)
	if !ok {
		return
	}
	source := "app"
	if r.URL.Query().Get("source") == "share" {
		source = "share"
	}
	id, dup, err := h.store(r.Context(), auth.UserID(r.Context()), u, source)
	if err != nil {
		httpx.ServerError(w, "receipt store", err)
		return
	}
	if dup && u.text != "" {
		h.saveText(r.Context(), id, u.text)
	}
	status := http.StatusCreated
	if dup {
		status = http.StatusOK
	}
	h.writeOne(w, r, id, status, dup)
}

// Inbox iOS Kestirme / dış gönderim içindir: gelen kutusu anahtarıyla
// dosyayı kabul eder; dekont uygulama açılınca okunur.
func (h *Handler) Inbox(w http.ResponseWriter, r *http.Request) {
	u, ok := readUpload(w, r)
	if !ok {
		return
	}
	_, dup, err := h.store(r.Context(), auth.UserID(r.Context()), u, "inbox")
	if err != nil {
		httpx.ServerError(w, "inbox store", err)
		return
	}
	msg := "Dekont Cüzdan'a gönderildi. Uygulamayı açınca işleme çevirebilirsin."
	if dup {
		msg = "Bu dekont zaten Cüzdan'da."
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "message": msg})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "used" && status != "dismissed" {
		httpx.WriteError(w, http.StatusBadRequest, "status pending, used veya dismissed olmali")
		return
	}
	rows, err := h.DB.Query(r.Context(), selectReceipt+"WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT 100",
		auth.UserID(r.Context()), status)
	if err != nil {
		httpx.ServerError(w, "receipts list", err)
		return
	}
	list, err := pgx.CollectRows(rows, scanReceipt)
	if err != nil {
		httpx.ServerError(w, "receipts scan", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	h.writeOne(w, r, id, http.StatusOK, false)
}

// File dekont dosyasını döner. Sahibi veya bağlı olduğu grup işleminin
// üyeleri görebilir.
func (h *Handler) File(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var key, mimeType string
	var name *string
	err := h.DB.QueryRow(r.Context(), `
SELECT rc.storage_key, rc.mime_type, rc.original_name FROM receipts rc
LEFT JOIN transactions t ON t.id = rc.transaction_id
WHERE rc.id = $1 AND (rc.user_id = $2 OR EXISTS (
  SELECT 1 FROM group_members gm WHERE gm.group_id = t.group_id AND gm.user_id = $2))`,
		id, auth.UserID(r.Context()),
	).Scan(&key, &mimeType, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "dekont bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "receipt file", err)
		return
	}
	f, err := os.Open(h.path(key, mimeType))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "dekont dosyasi bulunamadi")
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	filename := "dekont-" + strconv.Itoa(id) + allowed[mimeType]
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, filename, st.ModTime(), f)
}

func (h *Handler) saveText(ctx context.Context, id int, text string) error {
	parsed, _ := json.Marshal(Parse(text))
	_, err := h.DB.Exec(ctx, "UPDATE receipts SET text = $2, parsed = $3, updated_at = now() WHERE id = $1", id, text, parsed)
	return err
}

type parseRequest struct {
	Text string `json:"text"`
}

// ParseText istemcinin çıkardığı metni kaydeder ve okur (gelen kutusundaki
// dekontlar uygulamada açılınca).
func (h *Handler) ParseText(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req parseRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Text) == "" || len(req.Text) > maxTextBytes {
		httpx.WriteError(w, http.StatusBadRequest, "metin bos veya cok uzun")
		return
	}
	var exists bool
	err := h.DB.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM receipts WHERE id = $1 AND user_id = $2)",
		id, auth.UserID(r.Context())).Scan(&exists)
	if err != nil {
		httpx.ServerError(w, "receipt parse check", err)
		return
	}
	if !exists {
		httpx.WriteError(w, http.StatusNotFound, "dekont bulunamadi")
		return
	}
	if err := h.saveText(r.Context(), id, req.Text); err != nil {
		httpx.ServerError(w, "receipt parse save", err)
		return
	}
	h.writeOne(w, r, id, http.StatusOK, false)
}

type updateRequest struct {
	Status string `json:"status"`
}

// Update gelen kutusundaki dekontu yok sayar ya da geri alır.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var req updateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Status != "pending" && req.Status != "dismissed" {
		httpx.WriteError(w, http.StatusBadRequest, "status pending veya dismissed olmali")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
UPDATE receipts SET status = $3, updated_at = now()
WHERE id = $1 AND user_id = $2 AND transaction_id IS NULL`, id, auth.UserID(r.Context()), req.Status)
	if err != nil {
		httpx.ServerError(w, "receipt update", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "dekont bulunamadi veya bir isleme bagli")
		return
	}
	h.writeOne(w, r, id, http.StatusOK, false)
}

// Delete dekontu ve dosyasını siler (bağlı işlem yerinde kalır).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id")
	if !ok {
		return
	}
	var key, mimeType string
	err := h.DB.QueryRow(r.Context(), "DELETE FROM receipts WHERE id = $1 AND user_id = $2 RETURNING storage_key, mime_type",
		id, auth.UserID(r.Context())).Scan(&key, &mimeType)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "dekont bulunamadi")
		return
	}
	if err != nil {
		httpx.ServerError(w, "receipt delete", err)
		return
	}
	os.Remove(h.path(key, mimeType))
	w.WriteHeader(http.StatusNoContent)
}

// Attach dekontu bir işleme bağlar (işlem oluşturma/düzenleme içinde çağrılır).
func Attach(ctx context.Context, tx pgx.Tx, userID, receiptID, transactionID int) error {
	tag, err := tx.Exec(ctx, `
UPDATE receipts SET transaction_id = $3, status = 'used', updated_at = now()
WHERE id = $1 AND user_id = $2 AND (transaction_id IS NULL OR transaction_id = $3)`,
		receiptID, userID, transactionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.BadRequest("dekont bulunamadi veya baska bir isleme bagli")
	}
	return nil
}
