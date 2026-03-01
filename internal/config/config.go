package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contiene toda la configuración del servicio leída desde variables de entorno.
type Config struct {
	Port               string
	BrevoAPIKey        string
	BrevoSenderEmail   string
	BrevoSenderName    string
	WebhookSecret      string
	SMTPTimeoutSeconds int
}

// Load lee y valida las variables de entorno. Falla si falta alguna crítica.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.BrevoAPIKey = os.Getenv("BREVO_API_KEY")
	cfg.BrevoSenderEmail = os.Getenv("BREVO_SENDER_EMAIL")
	cfg.WebhookSecret = os.Getenv("WEBHOOK_SECRET")

	missing := []string{}
	for key, val := range map[string]string{
		"BREVO_API_KEY":      cfg.BrevoAPIKey,
		"BREVO_SENDER_EMAIL": cfg.BrevoSenderEmail,
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

	cfg.BrevoSenderName = os.Getenv("BREVO_SENDER_NAME")
	if cfg.BrevoSenderName == "" {
		cfg.BrevoSenderName = "Natillera"
	}

	cfg.SMTPTimeoutSeconds = 30
	if s := os.Getenv("SMTP_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.SMTPTimeoutSeconds = n
		}
	}

	return cfg, nil
}
