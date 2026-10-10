package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"wallet-api/internal/accounts"
	"wallet-api/internal/auth"
	"wallet-api/internal/budgets"
	"wallet-api/internal/goals"
	"wallet-api/internal/groups"
	"wallet-api/internal/httpx"
	"wallet-api/internal/idem"
	"wallet-api/internal/ledger"
	"wallet-api/internal/notify"
	"wallet-api/internal/rates"
	"wallet-api/internal/receipts"
	"wallet-api/internal/recurring"
	"wallet-api/internal/reminders"
	"wallet-api/internal/transactions"
	"wallet-api/migrations"
)

func main() {
	// "wallet-api gen-vapid": bildirimler için VAPID anahtar çifti üretir.
	if len(os.Args) > 1 && os.Args[1] == "gen-vapid" {
		priv, pub, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", pub, priv)
		return
	}

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

	if os.Getenv("MIGRATE_ON_START") == "true" {
		if err := migrate(dbURL); err != nil {
			log.Fatal("migration basarisiz: ", err)
		}
	}

	notifier := &notify.Notifier{
		DB:         pool,
		PublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		PrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		Subject:    os.Getenv("VAPID_SUBJECT"),
	}
	if !notifier.PushEnabled() {
		log.Println("VAPID anahtarlari yok: push bildirimleri kapali (uygulama ici bildirimler calisir)")
	}

	authHandler := &auth.Handler{DB: pool, SecureCookie: os.Getenv("COOKIE_SECURE") == "true"}
	// Face ID (passkey): WEBAUTHN_RP_ID alan adı, WEBAUTHN_ORIGINS virgülle ayrılmış tam adresler.
	// Yerelde varsayılan localhost (Vite :5173).
	rpID, origins := os.Getenv("WEBAUTHN_RP_ID"), os.Getenv("WEBAUTHN_ORIGINS")
	if rpID == "" {
		rpID, origins = "localhost", "http://localhost:5173"
	}
	if wa, err := auth.NewWebAuthn(rpID, strings.Split(origins, ",")); err != nil {
		log.Println("WebAuthn ayarlanamadi, Face ID girisi kapali:", err)
	} else {
		authHandler.WebAuthn = wa
	}
	accountHandler := &accounts.Handler{DB: pool}
	txHandler := &transactions.Handler{DB: pool, Loc: loc, Notify: notifier}
	groupHandler := &groups.Handler{DB: pool, Loc: loc, Notify: notifier}
	budgetHandler := &budgets.Handler{DB: pool, Loc: loc, Notify: notifier}
	runner := &recurring.Runner{DB: pool, Loc: loc, Created: func(ctx context.Context, ruleID, txID, owner int) {
		notifyRecurring(ctx, pool, notifier, loc, txID, owner)
	}}
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
	mux.HandleFunc("POST /auth/passkey/begin", authHandler.BeginPasskeyLogin)
	mux.HandleFunc("POST /auth/passkey/finish", authHandler.FinishPasskeyLogin)

	// Oturum gerektiren uçlar.
	idempotent := idem.Middleware(pool)
	protected := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, authHandler.RequireAuth(idempotent(fn)))
	}
	protected("GET /me", authHandler.Me)
	protected("POST /me/tours", authHandler.MarkTours)
	protected("POST /me/password", authHandler.ChangePassword)
	protected("GET /passkeys", authHandler.ListPasskeys)
	protected("POST /passkeys/register/begin", authHandler.BeginPasskeyRegistration)
	protected("POST /passkeys/register/finish", authHandler.FinishPasskeyRegistration)
	protected("DELETE /passkeys/{id}", authHandler.DeletePasskey)
	protected("DELETE /me", authHandler.DeleteAccount)

	protected("GET /notifications", notifier.List)
	protected("GET /events", notifier.Events)
	protected("POST /notifications/read", notifier.MarkRead)
	protected("GET /push/key", notifier.Key)
	protected("POST /push/subscribe", notifier.Subscribe)
	protected("POST /push/unsubscribe", notifier.Unsubscribe)
	protected("POST /push/test", notifier.Test)

	protected("POST /transfers", txHandler.CreateTransfer)
	protected("GET /export/transactions.csv", txHandler.ExportCSV)

	protected("GET /budgets", budgetHandler.List)
	protected("POST /budgets", budgetHandler.Create)
	protected("PATCH /budgets/{id}", budgetHandler.Update)
	protected("DELETE /budgets/{id}", budgetHandler.Delete)
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
	protected("POST /groups/{id}/invites", groupHandler.CreateInvite)
	protected("POST /groups/{id}/members", groupHandler.CreateInvite) // eski istemciler: artık davet gönderir
	protected("DELETE /groups/{id}/invites/{iid}", groupHandler.CancelInvite)
	protected("GET /invites", groupHandler.MyInvites)
	protected("POST /invites/{id}/accept", groupHandler.AcceptInvite)
	protected("POST /invites/{id}/decline", groupHandler.DeclineInvite)
	protected("POST /groups/{id}/leave", groupHandler.Leave)
	protected("GET /groups/{id}/transactions", txHandler.ListByGroup)
	protected("GET /groups/{id}/balances", groupHandler.Balances)
	protected("GET /groups/{id}/summary", groupHandler.Summary)
	protected("GET /groups/{id}/settlements", groupHandler.ListSettlements)
	protected("POST /groups/{id}/settlements", groupHandler.CreateSettlement)
	protected("DELETE /groups/{id}/settlements/{sid}", groupHandler.DeleteSettlement)
	protected("GET /settlements/pending", groupHandler.PendingSettlements)
	protected("POST /settlements/{id}/book", groupHandler.BookSettlement)
	protected("POST /settlements/{id}/dismiss", groupHandler.DismissSettlement)
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
	srv.RegisterOnShutdown(notifier.StopLive)

	runCtx, stopRunner := context.WithCancel(context.Background())
	defer stopRunner()
	runner.Start(runCtx, 15*time.Minute)
	// Kart son ödeme, yarınki düzenli ödeme ve ay başı özet hatırlatmaları.
	(&reminders.Runner{DB: pool, Loc: loc, Notify: notifier}).Start(runCtx, 30*time.Minute)
	go idem.Cleanup(runCtx, pool)

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

// Unwrap http.ResponseController'ın (SSE Flush/SetWriteDeadline) asıl yazıcıya ulaşması için.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

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

// migrate gömülü migration'ları uygular (goose).
func migrate(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, ".")
}

// notifyRecurring düzenli ödeme işlendiğinde sahibine haber verir; grup
// gideriyse diğer üyelere ve bütçe kontrolüne de gider.
func notifyRecurring(ctx context.Context, pool *pgxpool.Pool, n *notify.Notifier, loc *time.Location, txID, owner int) {
	var desc, category *string
	var amount int64
	var currency, typ string
	err := pool.QueryRow(ctx, `
SELECT t.description, t.category, t.amount, a.currency, t.type::text
FROM transactions t JOIN accounts a ON a.id = t.account_id WHERE t.id = $1`, txID).Scan(&desc, &category, &amount, &currency, &typ)
	if err != nil {
		return
	}
	what := "Düzenli ödeme"
	if desc != nil && *desc != "" {
		what = *desc
	} else if category != nil {
		what = budgets.CategoryLabel(*category)
	}
	sym := map[string]string{"TRY": "₺", "USD": "$", "EUR": "€", "GBP": "£"}[currency]
	verb := "işlendi"
	if typ == "income" {
		verb = "hesabına eklendi"
	}
	n.Send(ctx, notify.Notice{
		UserID: owner, Kind: "recurring", RefID: &txID, URL: "/recurring",
		Title: what + " " + verb, Body: ledger.FormatMoney(amount) + " " + sym,
	})
	budgets.Check(ctx, pool, n, loc, txID)
	transactions.NotifyGroupExpense(ctx, pool, n, txID)
}
