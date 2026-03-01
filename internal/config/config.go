package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contiene toda la configuración del servicio leída desde variables de entorno.
type Config struct {
	Port               string
	GmailUser          string
	GmailAppPassword   string
	WebhookSecret      string
	SMTPTimeoutSeconds int
}

// Load lee y valida las variables de entorno. Falla si falta alguna crítica.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.GmailUser = os.Getenv("GMAIL_USER")
	cfg.GmailAppPassword = os.Getenv("GMAIL_APP_PASSWORD")
	cfg.WebhookSecret = os.Getenv("WEBHOOK_SECRET")

	// Variables críticas — el servicio no puede operar sin ellas
	missing := []string{}
	for key, val := range map[string]string{
		"GMAIL_USER":         cfg.GmailUser,
		"GMAIL_APP_PASSWORD": cfg.GmailAppPassword,
		"WEBHOOK_SECRET":     cfg.WebhookSecret,
	} {
		if val == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("variables de entorno faltantes: %v", missing)
	}

	cfg.Port = os.Getenv("PORT")
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	cfg.SMTPTimeoutSeconds = 10
	if s := os.Getenv("SMTP_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.SMTPTimeoutSeconds = n
		}
	}

	return cfg, nil
}
