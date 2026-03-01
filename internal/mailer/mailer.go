package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"time"

	_ "embed"

	"natillera/internal/domain"
)

// logoBytes contiene el logo embebido en el binario en tiempo de compilación.
//
//go:embed assets/logo.png
var logoBytes []byte

const logoCID = "logo@natillera"

// Mailer define el contrato para enviar correos.
type Mailer interface {
	Send(ctx context.Context, a domain.Aporte) error
}

// SMTPMailer implementa Mailer usando Gmail SMTP.
type SMTPMailer struct {
	User     string
	Password string
	Timeout  time.Duration
}

// Send envía el correo con retry (máximo 2 intentos).
func (m *SMTPMailer) Send(ctx context.Context, a domain.Aporte) error {
	const maxRetries = 2
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = m.sendOnce(ctx, a)
		if lastErr == nil {
			return nil
		}
		log.Printf("level=warn event=smtp_retry attempt=%d/%d id_aporte=%s err=%v", attempt, maxRetries, a.IDAporte, lastErr)

		select {
		case <-ctx.Done():
			return fmt.Errorf("contexto cancelado en reintento %d: %w", attempt, ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("todos los intentos SMTP fallaron: %w", lastErr)
}

// sendOnce realiza un único intento de envío SMTP respetando el contexto.
func (m *SMTPMailer) sendOnce(ctx context.Context, a domain.Aporte) error {
	subject := fmt.Sprintf("Confirmación de aporte - %s", a.Mes)
	html := buildHTMLTemplate(a)
	plain := buildPlainText(a)
	msg := buildMessage(m.User, a.Correo, subject, html, plain)

	auth := smtp.PlainAuth("", m.User, m.Password, "smtp.gmail.com")

	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		err := smtp.SendMail("smtp.gmail.com:587", auth, m.User, []string{a.Correo}, msg)
		ch <- result{err}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout SMTP: %w", ctx.Err())
	case res := <-ch:
		return res.err
	}
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
// El logo se referencia via CID para que sea renderizado inline por el cliente de correo.
func buildHTMLTemplate(a domain.Aporte) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f4f4f4;font-family:Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f4;padding:32px 0;">
    <tr>
      <td align="center">
        <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">

          <!-- Header con logo -->
          <tr>
            <td style="background:#1a7a4a;padding:24px 40px;text-align:center;">
              <img src="cid:%s" alt="Natillera" style="max-height:64px;max-width:200px;display:block;margin:0 auto;" />
            </td>
          </tr>

          <!-- Cuerpo -->
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

          <!-- Footer con logo pequeño -->
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

// buildMessage construye el mensaje MIME completo con imagen inline via CID.
//
// Estructura MIME:
//
//	multipart/alternative
//	  ├── text/plain          (fallback para clientes sin HTML)
//	  └── multipart/related
//	        ├── text/html     (referencia logo con cid:)
//	        └── image/png     (logo embebido, Content-ID: <logo@natillera>)
func buildMessage(from, to, subject, html, plain string) []byte {
	altBoundary := "natillera_alt_001"
	relBoundary := "natillera_rel_001"

	var buf bytes.Buffer

	// Cabeceras principales
	buf.WriteString("From: Natillera <" + from + ">\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
	buf.WriteString("\r\n")

	// Parte 1: text/plain (fallback)
	buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
	buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(plain)
	buf.WriteString("\r\n\r\n")

	// Parte 2: multipart/related (HTML + imagen inline)
	buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/related; boundary=\"%s\"\r\n", relBoundary))
	buf.WriteString("\r\n")

	// Parte 2a: text/html
	buf.WriteString(fmt.Sprintf("--%s\r\n", relBoundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(html)
	buf.WriteString("\r\n\r\n")

	// Parte 2b: imagen logo inline (CID)
	buf.WriteString(fmt.Sprintf("--%s\r\n", relBoundary))
	buf.WriteString("Content-Type: image/png\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", logoCID))
	buf.WriteString("Content-Disposition: inline; filename=\"logo.png\"\r\n")
	buf.WriteString("\r\n")

	// Codificar logo en base64 con líneas de 76 caracteres (RFC 2045)
	encoded := base64.StdEncoding.EncodeToString(logoBytes)
	for len(encoded) > 76 {
		buf.WriteString(encoded[:76])
		buf.WriteString("\r\n")
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		buf.WriteString(encoded)
		buf.WriteString("\r\n")
	}

	buf.WriteString(fmt.Sprintf("--%s--\r\n", relBoundary))
	buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))

	return buf.Bytes()
}
