package domain

import (
	"fmt"
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Aporte representa el modelo de negocio de un aporte mensual de un socio.
type Aporte struct {
	IDAporte        string
	IDSocio         string
	PrimerNombre    string
	Correo          string
	Mes             string
	Monto           float64
	FechaPago       string
	SemanasMora     int
	InteresGenerado float64
	TotalAPagar     float64
	AporteRifa      float64
	FechaLimite     string
}

// Validate verifica que el aporte tenga todos los campos requeridos y valores válidos.
func (a Aporte) Validate() error {
	if a.IDAporte == "" {
		return fmt.Errorf("ID Aporte es requerido")
	}
	if a.IDSocio == "" {
		return fmt.Errorf("ID Socio es requerido")
	}
	if a.Mes == "" {
		return fmt.Errorf("Mes es requerido")
	}
	if a.Correo == "" {
		return fmt.Errorf("Correo es requerido")
	}
	if !emailRegex.MatchString(a.Correo) {
		return fmt.Errorf("Correo no tiene un formato válido")
	}
	if a.Monto < 0 {
		return fmt.Errorf("Monto debe ser >= 0")
	}
	if a.TotalAPagar < 0 {
		return fmt.Errorf("TotalAPagar debe ser >= 0")
	}
	return nil
}
