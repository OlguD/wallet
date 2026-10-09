package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"wallet-api/internal/accounts"
	"wallet-api/internal/auth"
	"wallet-api/internal/goals"
	"wallet-api/internal/groups"
	"wallet-api/internal/httpx"
	"wallet-api/internal/rates"
	"wallet-api/internal/receipts"
	"wallet-api/internal/recurring"
	"wallet-api/internal/transactions"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env dosyasi bulunamadi, sistem ortam degiskenleri kullanilacak")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL tanimli degil")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	tz := os.Getenv("APP_TIMEZONE")
	if tz == "" {
		tz = "Europe/Istanbul"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Fatal("APP_TIMEZONE gecersiz: ", err)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("havuz olusturulamadi: ", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("veritabani baglantisi basarisiz: ", err)
	}
	log.Println("veritabani baglandı")

	authHandler := &auth.Handler{DB: pool, SecureCookie: os.Getenv("COOKIE_SECURE") == "true"}
	accountHandler := &accounts.Handler{DB: pool}
	txHandler := &transactions.Handler{DB: pool, Loc: loc}
	groupHandler := &groups.Handler{DB: pool, Loc: loc}
	runner := &recurring.Runner{DB: pool, Loc: loc}
	recurringHandler := &recurring.Handler{DB: pool, Loc: loc, Runner: runner}
	goalHandler := &goals.Handler{DB: pool, Loc: loc}
	rateHandler := rates.New(loc)
	receiptsDir := os.Getenv("RECEIPTS_DIR")
	if receiptsDir == "" {
		receiptsDir = "data/receipts"
	}
	receiptHandler := &receipts.Handler{DB: pool, Dir: receiptsDir}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	// Oturum gerektiren uçlar.
	protected := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, authHandler.RequireAuth(fn))
	}
	protected("GET /me", authHandler.Me)
	protected("POST /me/tours", authHandler.MarkTours)
	protected("POST /me/inbox-token", authHandler.RotateInboxToken)
	protected("DELETE /me/inbox-token", authHandler.RevokeInboxToken)

	// iOS Kestirme: oturum değil, gelen kutusu anahtarı ile dekont yükleme.
	mux.Handle("POST /inbox", authHandler.RequireInboxToken(http.HandlerFunc(receiptHandler.Inbox)))
	protected("GET /receipts", receiptHandler.List)
	protected("POST /receipts", receiptHandler.Upload)
	protected("GET /receipts/{id}", receiptHandler.Get)
	protected("GET /receipts/{id}/file", receiptHandler.File)
	protected("POST /receipts/{id}/parse", receiptHandler.ParseText)
	protected("PATCH /receipts/{id}", receiptHandler.Update)
	protected("DELETE /receipts/{id}", receiptHandler.Delete)
	protected("GET /reports/counterparties", txHandler.Counterparties)

	protected("GET /accounts", accountHandler.List)
	protected("POST /accounts", accountHandler.Create)
	protected("GET /accounts/{id}", accountHandler.Get)
	protected("PATCH /accounts/{id}", accountHandler.Update)
	protected("DELETE /accounts/{id}", accountHandler.Delete)
	protected("GET /accounts/{id}/transactions", txHandler.ListByAccount)

	protected("GET /transactions", txHandler.List)
	protected("POST /transactions", txHandler.Create)
	protected("GET /transactions/{id}", txHandler.Get)
	protected("PATCH /transactions/{id}", txHandler.Update)
	protected("DELETE /transactions/{id}", txHandler.Delete)
	protected("GET /summary", txHandler.Summary)

	protected("GET /groups", groupHandler.List)
	protected("POST /groups", groupHandler.Create)
	protected("GET /groups/{id}", groupHandler.Get)
	protected("PATCH /groups/{id}", groupHandler.Rename)
	protected("POST /groups/{id}/members", groupHandler.AddMember)
	protected("POST /groups/{id}/leave", groupHandler.Leave)
	protected("GET /groups/{id}/transactions", txHandler.ListByGroup)
	protected("GET /groups/{id}/balances", groupHandler.Balances)
	protected("GET /groups/{id}/summary", groupHandler.Summary)
	protected("GET /groups/{id}/settlements", groupHandler.ListSettlements)
	protected("POST /groups/{id}/settlements", groupHandler.CreateSettlement)
	protected("DELETE /groups/{id}/settlements/{sid}", groupHandler.DeleteSettlement)
	protected("GET /groups/{id}/recurring", recurringHandler.ListByGroup)

	protected("GET /recurring", recurringHandler.List)
	protected("POST /recurring", recurringHandler.Create)
	protected("GET /recurring/{id}", recurringHandler.Get)
	protected("PATCH /recurring/{id}", recurringHandler.Update)
	protected("DELETE /recurring/{id}", recurringHandler.Delete)

	protected("GET /goals", goalHandler.List)
	protected("POST /goals", goalHandler.Create)
	protected("GET /goals/{id}", goalHandler.Get)
	protected("PATCH /goals/{id}", goalHandler.Update)
	protected("DELETE /goals/{id}", goalHandler.Delete)
	protected("GET /rates", rateHandler.List)

	protected("GET /goals/{id}/contributions", goalHandler.ListContributions)
	protected("POST /goals/{id}/contributions", goalHandler.AddContribution)
	protected("DELETE /goals/{id}/contributions/{cid}", goalHandler.DeleteContribution)

	srv := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second, // dekont yüklemeleri (10 MB) yavaş mobil bağlantıda
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	runCtx, stopRunner := context.WithCancel(context.Background())
	defer stopRunner()
	runner.Start(runCtx, 15*time.Minute)

	go func() {
		log.Printf("sunucu %s'de basladi", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("sunucu kapaniyor")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("kapanma hatasi:", err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}
