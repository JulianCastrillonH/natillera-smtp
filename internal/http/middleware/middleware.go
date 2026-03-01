package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// jsonError escribe una respuesta de error en formato JSON.
func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": msg})
}

// Auth valida que el header Authorization contenga el Bearer token correcto.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || authHeader != "Bearer "+secret {
				log.Printf("level=warn event=unauthorized ip=%s path=%s", r.RemoteAddr, r.URL.Path)
				jsonError(w, http.StatusUnauthorized, "no autorizado")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// JSONContentType rechaza solicitudes cuyo Content-Type no sea application/json.
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				jsonError(w, http.StatusUnsupportedMediaType, "Content-Type debe ser application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Logging registra método, path, status y duración de cada solicitud.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("level=info event=request method=%s path=%s status=%d duration_ms=%d ip=%s",
			r.Method, r.URL.Path, rw.status, time.Since(start).Milliseconds(), r.RemoteAddr)
	})
}

// responseWriter captura el status code para el middleware de logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
