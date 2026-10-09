// Package httpx JSON cevap/istek yardımcılarını içerir.
package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

const maxBodyBytes = 1 << 20

type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("json encode:", err)
	}
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, errorResponse{Error: msg})
}

// ServerError asıl hatayı loglar, kullanıcıya genel bir mesaj döner.
func ServerError(w http.ResponseWriter, where string, err error) {
	log.Printf("%s: %v", where, err)
	WriteError(w, http.StatusInternalServerError, "sunucu hatasi")
}

// DecodeJSON gövdeyi okur; hata durumunda 400 yazar ve false döner.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, "istek cok buyuk")
			return false
		}
		WriteError(w, http.StatusBadRequest, "gecersiz JSON")
		return false
	}
	return true
}

// PathID URL'deki {name} parametresini pozitif int olarak okur; hata durumunda 404 yazar.
func PathID(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil || id <= 0 {
		WriteError(w, http.StatusNotFound, "bulunamadi")
		return 0, false
	}
	return id, true
}

// Pagination ?limit=&offset= parametrelerini okur.
func Pagination(r *http.Request) (limit, offset int) {
	limit = 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = min(v, 200)
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		offset = v
	}
	return limit, offset
}

// Error kullanıcıya gösterilebilecek, HTTP durum kodu taşıyan hata.
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

func BadRequest(msg string) error { return &Error{http.StatusBadRequest, msg} }
func NotFound(msg string) error   { return &Error{http.StatusNotFound, msg} }
func Conflict(msg string) error   { return &Error{http.StatusConflict, msg} }
func Forbidden(msg string) error  { return &Error{http.StatusForbidden, msg} }

// Fail *Error ise mesajını ilgili durum koduyla yazar; değilse 500 döner ve loglar.
func Fail(w http.ResponseWriter, where string, err error) {
	var e *Error
	if errors.As(err, &e) {
		WriteError(w, e.Status, e.Msg)
		return
	}
	ServerError(w, where, err)
}

// Optional PATCH isteklerinde "alan hiç gönderilmedi" ile "null gönderildi"yi ayırır.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// Date JSON'da "YYYY-AA-GG" olarak taşınan takvim günü.
type Date struct{ time.Time }

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(time.DateOnly))
}

// DateRange ?from=&to= (YYYY-AA-GG, to hariç) okur. Varsayılan: loc'a göre içinde bulunulan ay.
func DateRange(r *http.Request, loc *time.Location) (from, to time.Time, err error) {
	now := time.Now().In(loc)
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	to = from.AddDate(0, 1, 0)
	q := r.URL.Query()
	if v := q.Get("from"); v != "" {
		if from, err = time.ParseInLocation(time.DateOnly, v, loc); err != nil {
			return from, to, BadRequest("from YYYY-AA-GG formatinda olmali")
		}
	}
	if v := q.Get("to"); v != "" {
		if to, err = time.ParseInLocation(time.DateOnly, v, loc); err != nil {
			return from, to, BadRequest("to YYYY-AA-GG formatinda olmali")
		}
	}
	if !to.After(from) {
		return from, to, BadRequest("to, from'dan sonra olmali")
	}
	return from, to, nil
}

// Today loc'a göre bugünün tarihini UTC gece yarısı olarak döner (DB date alanlarıyla uyumlu).
func Today(loc *time.Location) time.Time {
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
