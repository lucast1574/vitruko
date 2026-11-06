package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lucast1574/vitruko/internal/extractapi"
	"github.com/lucast1574/vitruko/internal/platform"
	"github.com/lucast1574/vitruko/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	api := extractapi.New()
	if os.Getenv("DATABASE_URL") == "" {
		slog.Error("DATABASE_URL is required; refusing unauthenticated startup")
		os.Exit(1)
	}
	geo := api.Handler()
	site := web.Handler()
	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/openapi.json" || strings.HasPrefix(r.URL.Path, "/v1/") {
			geo.ServeHTTP(w, r)
		} else {
			site.ServeHTTP(w, r)
		}
	})
	startup, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	app, err := platform.New(startup, platform.Config{DatabaseURL: os.Getenv("DATABASE_URL"), FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"), FirebaseAPIKey: os.Getenv("FIREBASE_API_KEY"), FirebaseAppID: os.Getenv("FIREBASE_APP_ID"), AdminEmail: os.Getenv("ADMIN_EMAIL"), AdminPassword: os.Getenv("ADMIN_PASSWORD"), AppURL: os.Getenv("APP_URL"), SiteURL: os.Getenv("SITE_URL"), StripeKey: os.Getenv("STRIPE_SECRET_KEY"), StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"), StripePortalConfig: os.Getenv("STRIPE_PORTAL_CONFIG"), StripePrices: map[string]string{"launch": os.Getenv("STRIPE_PRICE_LAUNCH"), "scale": os.Getenv("STRIPE_PRICE_SCALE")}, ResendKey: os.Getenv("RESEND_API_KEY"), MailFrom: os.Getenv("MAIL_FROM")}, combined)
	cancelStartup()
	if err != nil {
		slog.Error("platform startup failed", "error", err)
		os.Exit(1)
	}
	defer app.Close()
	server := &http.Server{Addr: ":" + port, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go app.Maintain(ctx)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("Vitruko API listening", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
