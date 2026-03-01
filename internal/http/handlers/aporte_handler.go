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

// flexFloat acepta tanto número JSON como string con formato colombiano.
// Soporta: 220000 | "220000" | "220.000,50" | "220.000"
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	// Intentar como número JSON directo
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexFloat(n)
		return nil
	}
	// Intentar como string
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*f = flexFloat(parseCOP(s))
	return nil
}

// flexInt acepta tanto número JSON como string.
type flexInt int

func (i *flexInt) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*i = flexInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ""))
	n, _ = strconv.Atoi(s)
	*i = flexInt(n)
	return nil
}

// parseCOP convierte un string numérico en formato colombiano o estándar a float64.
// Reglas:
//   - "1.200.000,50" → 1200000.50  (punto=miles, coma=decimal)
//   - "200.000"      → 200000       (3 dígitos tras el punto → separador de miles)
//   - "200.5"        → 200.5        (1-2 dígitos → separador decimal)
//   - "0,50"         → 0.50         (solo coma → decimal)
func parseCOP(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		// Formato colombiano completo: 1.200.000,50
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if strings.Contains(s, ",") {
		// Solo coma como decimal: 0,50
		s = strings.ReplaceAll(s, ",", ".")
	} else if strings.Contains(s, ".") {
		// Solo punto: si el último segmento tiene 3 dígitos → miles
		parts := strings.Split(s, ".")
		if len(parts[len(parts)-1]) == 3 {
			s = strings.ReplaceAll(s, ".", "")
		}
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// aporteRequest recibe el payload de AppSheet.
// Los campos numéricos usan tipos flex que aceptan número o string.
type aporteRequest struct {
	IDAporte        string    `json:"id_aporte"`
	IDSocio         string    `json:"id_socio"`
	PrimerNombre    string    `json:"primer_nombre"`
	Correo          string    `json:"correo"`
	Mes             string    `json:"mes"`
	Monto           flexFloat `json:"monto"`
	FechaPago       string    `json:"fecha_pago"`
	SemanasMora     flexInt   `json:"semanas_mora"`
	InteresGenerado flexFloat `json:"interes_generado"`
	TotalAPagar     flexFloat `json:"total_a_pagar"`
	AporteRifa      flexFloat `json:"aporte_rifa"`
	FechaLimite     string    `json:"fecha_limite"`
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
		Monto:           float64(req.Monto),
		FechaPago:       req.FechaPago,
		SemanasMora:     int(req.SemanasMora),
		InteresGenerado: float64(req.InteresGenerado),
		TotalAPagar:     float64(req.TotalAPagar),
		AporteRifa:      float64(req.AporteRifa),
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
