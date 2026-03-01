package main

import (
	"log"
	"net/http"
	"time"

	"natillera/internal/config"
	"natillera/internal/http/handlers"
	"natillera/internal/http/middleware"
	"natillera/internal/mailer"
	"natillera/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("level=fatal event=config_error err=%v", err)
	}

	smtpMailer := &mailer.SMTPMailer{
		User:     cfg.GmailUser,
		Password: cfg.GmailAppPassword,
		Timeout:  time.Duration(cfg.SMTPTimeoutSeconds) * time.Second,
	}

	aporteService := service.NewAporteService(smtpMailer)
	aporteHandler := handlers.NewAporteHandler(aporteService, cfg.SMTPTimeoutSeconds)

	envVars := []string{"GMAIL_USER", "GMAIL_APP_PASSWORD", "WEBHOOK_SECRET"}

	// Pipeline de middlewares para el webhook: Logging → Auth → JSONContentType → Handler
	webhookChain := middleware.Logging(
		middleware.Auth(cfg.WebhookSecret)(
			middleware.JSONContentType(
				http.HandlerFunc(aporteHandler.HandleWebhook),
			),
		),
	)

	mux := http.NewServeMux()
	mux.Handle("/webhook/aporte", webhookChain)
	mux.HandleFunc("/health", handlers.HandleHealth(envVars))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("level=info event=server_start addr=%s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("level=fatal event=server_error err=%v", err)
	}
}
