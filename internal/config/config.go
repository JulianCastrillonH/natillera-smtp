package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contiene toda la configuración del servicio leída desde variables de entorno.
type Config struct {
	Port               string
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPassword       string
	WebhookSecret      string
	SMTPTimeoutSeconds int
}

// Load lee y valida las variables de entorno. Falla si falta alguna crítica.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.SMTPUser = os.Getenv("SMTP_USER")
	cfg.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	cfg.WebhookSecret = os.Getenv("WEBHOOK_SECRET")

	missing := []string{}
	for key, val := range map[string]string{
		"SMTP_USER":     cfg.SMTPUser,
		"SMTP_PASSWORD": cfg.SMTPPassword,
		"WEBHOOK_SECRET": cfg.WebhookSecret,
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

	// Defaults: Outlook STARTTLS
	cfg.SMTPHost = os.Getenv("SMTP_HOST")
	if cfg.SMTPHost == "" {
		cfg.SMTPHost = "smtp-mail.outlook.com"
	}

	cfg.SMTPPort = 587
	if s := os.Getenv("SMTP_PORT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.SMTPPort = n
		}
	}

	cfg.SMTPTimeoutSeconds = 30
	if s := os.Getenv("SMTP_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.SMTPTimeoutSeconds = n
		}
	}

	return cfg, nil
}
