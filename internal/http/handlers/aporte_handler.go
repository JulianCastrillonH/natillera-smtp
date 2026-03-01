package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"natillera/internal/domain"
	"natillera/internal/service"
)

// aporteRequest acepta todos los campos numéricos como string para tolerar
// el formato colombiano que AppSheet puede enviar (ej: "220.000,00").
type aporteRequest struct {
	IDAporte        string `json:"id_aporte"`
	IDSocio         string `json:"id_socio"`
	PrimerNombre    string `json:"primer_nombre"`
	Correo          string `json:"correo"`
	Mes             string `json:"mes"`
	Monto           string `json:"monto"`
	FechaPago       string `json:"fecha_pago"`
	SemanasMora     string `json:"semanas_mora"`
	InteresGenerado string `json:"interes_generado"`
	TotalAPagar     string `json:"total_a_pagar"`
	AporteRifa      string `json:"aporte_rifa"`
	FechaLimite     string `json:"fecha_limite"`
}

// parseCOP convierte un string numérico en formato colombiano o estándar a float64.
// Soporta: "220.000,50" → 220000.50 | "220000.50" → 220000.50 | "220000" → 220000
func parseCOP(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		// Formato colombiano: 220.000,50 → quitar punto, reemplazar coma por punto
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if strings.Contains(s, ",") {
		// Solo coma como decimal: 0,50 → 0.50
		s = strings.ReplaceAll(s, ",", ".")
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseInt convierte un string a int tolerando formato colombiano.
func parseInt(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.Atoi(s)
	return v
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
		Monto:           parseCOP(req.Monto),
		FechaPago:       req.FechaPago,
		SemanasMora:     parseInt(req.SemanasMora),
		InteresGenerado: parseCOP(req.InteresGenerado),
		TotalAPagar:     parseCOP(req.TotalAPagar),
		AporteRifa:      parseCOP(req.AporteRifa),
		FechaLimite:     req.FechaLimite,
	}

	if err := h.service.ProcesarAporte(r.Context(), aporte); err != nil {
		if strings.HasPrefix(err.Error(), "validación fallida:") {
			jsonResponse(w, http.StatusBadRequest, "error", err.Error())
		} else {
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
