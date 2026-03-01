package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"natillera/internal/domain"
	"natillera/internal/service"
)

// aporteRequest es la representación JSON del payload recibido desde AppSheet.
// Los tags json coinciden exactamente con los nombres de columna en AppSheet.
type aporteRequest struct {
	IDAporte        string  `json:"ID Aporte"`
	IDSocio         string  `json:"ID Socio"`
	PrimerNombre    string  `json:"PrimerNombre"`
	Correo          string  `json:"Correo"`
	Mes             string  `json:"Mes"`
	Monto           float64 `json:"Monto"`
	FechaPago       string  `json:"FechaPago"`
	SemanasMora     int     `json:"SemanasMora"`
	InteresGenerado float64 `json:"InteresGenerado"`
	TotalAPagar     float64 `json:"TotalAPagar"`
	AporteRifa      float64 `json:"AporteRifa"`
	FechaLimite     string  `json:"FechaLimite"`
}

// AporteHandler gestiona las solicitudes HTTP del webhook de aportes.
type AporteHandler struct {
	service        *service.AporteService
	smtpTimeoutSec int
}

// NewAporteHandler crea una nueva instancia de AporteHandler.
func NewAporteHandler(svc *service.AporteService, smtpTimeoutSec int) *AporteHandler {
	return &AporteHandler{service: svc, smtpTimeoutSec: smtpTimeoutSec}
}

// HandleWebhook procesa el POST /webhook/aporte proveniente de AppSheet.
func (h *AporteHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, "error", "método no permitido")
		return
	}

	// Limitar tamaño del body a 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req aporteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, "error", "payload inválido: "+err.Error())
		return
	}

	aporte := domain.Aporte{
		IDAporte:        req.IDAporte,
		IDSocio:         req.IDSocio,
		PrimerNombre:    req.PrimerNombre,
		Correo:          req.Correo,
		Mes:             req.Mes,
		Monto:           req.Monto,
		FechaPago:       req.FechaPago,
		SemanasMora:     req.SemanasMora,
		InteresGenerado: req.InteresGenerado,
		TotalAPagar:     req.TotalAPagar,
		AporteRifa:      req.AporteRifa,
		FechaLimite:     req.FechaLimite,
	}

	timeout := time.Duration(h.smtpTimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	if err := h.service.ProcesarAporte(ctx, aporte); err != nil {
		// Errores de validación de dominio → 400; otros → 500
		if strings.HasPrefix(err.Error(), "validación fallida:") {
			jsonResponse(w, http.StatusBadRequest, "error", err.Error())
		} else {
			// DEBUG: exponer error real para diagnóstico — remover en producción
			jsonResponse(w, http.StatusInternalServerError, "error", err.Error())
		}
		return
	}

	jsonResponse(w, http.StatusOK, "success", "correo enviado correctamente")
}

// HandleHealth verifica que las variables de entorno críticas estén presentes.
func HandleHealth(envVars []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonResponse(w, http.StatusMethodNotAllowed, "error", "método no permitido")
			return
		}

		checks := make(map[string]string, len(envVars))
		allOk := true
		for _, key := range envVars {
			if os.Getenv(key) != "" {
				checks[key] = "ok"
			} else {
				checks[key] = "missing"
				allOk = false
			}
		}

		statusCode := http.StatusOK
		statusMsg := "ok"
		if !allOk {
			statusCode = http.StatusServiceUnavailable
			statusMsg = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": statusMsg,
			"env":    checks,
		})
	}
}

// jsonResponse escribe una respuesta JSON estructurada.
func jsonResponse(w http.ResponseWriter, status int, s, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"status": s, "message": msg})
}
