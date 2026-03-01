package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contiene toda la configuración del servicio leída desde variables de entorno.
type Config struct {
	Port               string
	ResendAPIKey       string
	ResendFrom         string
	WebhookSecret      string
	SMTPTimeoutSeconds int
}

// Load lee y valida las variables de entorno. Falla si falta alguna crítica.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.ResendAPIKey = os.Getenv("RESEND_API_KEY")
	cfg.WebhookSecret = os.Getenv("WEBHOOK_SECRET")

	missing := []string{}
	for key, val := range map[string]string{
		"RESEND_API_KEY": cfg.ResendAPIKey,
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

	// Para pruebas sin dominio verificado usar: onboarding@resend.dev
	cfg.ResendFrom = os.Getenv("RESEND_FROM")
	if cfg.ResendFrom == "" {
		cfg.ResendFrom = "Natillera <onboarding@resend.dev>"
	}

	cfg.SMTPTimeoutSeconds = 30
	if s := os.Getenv("SMTP_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.SMTPTimeoutSeconds = n
		}
	}

	return cfg, nil
}
