package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "embed"

	"natillera/internal/domain"
)

//go:embed assets/logo.png
var logoBytes []byte

const (
	logoCID    = "logo@natillera"
	resendURL  = "https://api.resend.com/emails"
)

// Mailer define el contrato para enviar correos.
type Mailer interface {
	Send(ctx context.Context, a domain.Aporte) error
}

// ResendMailer implementa Mailer usando la API HTTP de Resend.com.
type ResendMailer struct {
	APIKey  string
	From    string
	Timeout time.Duration
}

// resendRequest es el payload JSON que espera la API de Resend.
type resendRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Text        string             `json:"text"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

// resendAttachment representa un adjunto inline (logo embebido via CID).
type resendAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id,omitempty"`
	Inline      bool   `json:"inline,omitempty"`
}

// Send envía el correo con retry (máximo 2 intentos).
func (m *ResendMailer) Send(ctx context.Context, a domain.Aporte) error {
	const maxRetries = 2
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = m.sendOnce(a)
		if lastErr == nil {
			return nil
		}
		log.Printf("level=warn event=resend_retry attempt=%d/%d id_aporte=%s err=%v", attempt, maxRetries, a.IDAporte, lastErr)

		select {
		case <-ctx.Done():
			return fmt.Errorf("request cancelado en reintento %d: %w", attempt, ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("todos los intentos fallaron: %w", lastErr)
}

// sendOnce realiza un único intento de envío via Resend API.
func (m *ResendMailer) sendOnce(a domain.Aporte) error {
	subject := fmt.Sprintf("Confirmación de aporte - %s", a.Mes)
	html := buildHTMLTemplate(a)
	plain := buildPlainText(a)

	payload := resendRequest{
		From:    m.From,
		To:      []string{a.Correo},
		Subject: subject,
		HTML:    html,
		Text:    plain,
		Attachments: []resendAttachment{
			{
				Filename:    "logo.png",
				Content:     base64.StdEncoding.EncodeToString(logoBytes),
				ContentType: "image/png",
				ContentID:   logoCID,
				Inline:      true,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	client := &http.Client{Timeout: m.Timeout}
	req, err := http.NewRequest(http.MethodPost, resendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http.Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// buildPlainText genera la versión texto plano del correo (fallback).
func buildPlainText(a domain.Aporte) string {
	return fmt.Sprintf(
		"Hola %s,\n\nHemos recibido tu aporte del mes %s.\n\nFecha de pago:    %s\nMonto:            $%.2f\nAporte rifa:      $%.2f\nInterés generado: $%.2f\nSemanas en mora:  %d\nTotal a pagar:    $%.2f\nFecha límite:     %s\n\nGracias por tu compromiso con la Natillera.",
		a.PrimerNombre, a.Mes,
		a.FechaPago, a.Monto, a.AporteRifa, a.InteresGenerado, a.SemanasMora, a.TotalAPagar, a.FechaLimite,
	)
}

// buildHTMLTemplate genera el cuerpo HTML del correo.
func buildHTMLTemplate(a domain.Aporte) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f4f4f4;font-family:Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f4;padding:32px 0;">
    <tr>
      <td align="center">
        <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
          <tr>
            <td style="background:#1a7a4a;padding:24px 40px;text-align:center;">
              <img src="cid:%s" alt="Natillera" style="max-height:64px;max-width:200px;display:block;margin:0 auto;" />
            </td>
          </tr>
          <tr>
            <td style="padding:36px 40px;color:#333333;font-size:15px;line-height:1.7;">
              <p style="margin:0 0 16px;">Hola <strong>%s</strong>,</p>
              <p style="margin:0 0 24px;">Hemos recibido tu aporte correspondiente al mes <strong>%s</strong>.</p>
              <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #e0e0e0;border-radius:6px;overflow:hidden;margin-bottom:24px;">
                <tr style="background:#f9f9f9;">
                  <td style="padding:12px 16px;color:#666;font-size:13px;">Fecha de pago</td>
                  <td style="padding:12px 16px;text-align:right;font-weight:bold;">%s</td>
                </tr>
                <tr>
                  <td style="padding:12px 16px;color:#666;font-size:13px;">Monto</td>
                  <td style="padding:12px 16px;text-align:right;font-weight:bold;">$%.2f</td>
                </tr>
                <tr style="background:#f9f9f9;">
                  <td style="padding:12px 16px;color:#666;font-size:13px;">Aporte rifa</td>
                  <td style="padding:12px 16px;text-align:right;font-weight:bold;">$%.2f</td>
                </tr>
                <tr>
                  <td style="padding:12px 16px;color:#666;font-size:13px;">Interés generado</td>
                  <td style="padding:12px 16px;text-align:right;font-weight:bold;">$%.2f</td>
                </tr>
                <tr style="background:#f9f9f9;">
                  <td style="padding:12px 16px;color:#666;font-size:13px;">Semanas en mora</td>
                  <td style="padding:12px 16px;text-align:right;font-weight:bold;">%d</td>
                </tr>
                <tr style="background:#1a7a4a;">
                  <td style="padding:14px 16px;color:#ffffff;font-weight:bold;">Total a pagar</td>
                  <td style="padding:14px 16px;text-align:right;color:#ffffff;font-weight:bold;font-size:16px;">$%.2f</td>
                </tr>
              </table>
              <p style="margin:0 0 8px;color:#555;">Fecha límite de pago: <strong>%s</strong></p>
              <p style="margin:0;color:#555;">Gracias por tu compromiso con la Natillera.</p>
            </td>
          </tr>
          <tr>
            <td style="background:#f0f0f0;padding:20px 40px;text-align:center;border-top:1px solid #e0e0e0;">
              <img src="cid:%s" alt="Natillera" style="max-height:32px;max-width:100px;display:block;margin:0 auto 8px;" />
              <p style="margin:0;font-size:12px;color:#999;">© 2026 Natillera · Todos los derechos reservados</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		logoCID,
		a.PrimerNombre, a.Mes,
		a.FechaPago,
		a.Monto, a.AporteRifa, a.InteresGenerado, a.SemanasMora, a.TotalAPagar,
		a.FechaLimite,
		logoCID,
	)
}
