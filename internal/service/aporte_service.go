package service

import (
	"context"
	"fmt"
	"log"

	"natillera/internal/domain"
	"natillera/internal/mailer"
)

// AporteService orquesta la lógica de negocio para procesar un aporte mensual.
type AporteService struct {
	mailer mailer.Mailer
}

// NewAporteService crea una nueva instancia de AporteService.
func NewAporteService(m mailer.Mailer) *AporteService {
	return &AporteService{mailer: m}
}

// ProcesarAporte valida el aporte y delega el envío del correo al mailer.
func (s *AporteService) ProcesarAporte(ctx context.Context, a domain.Aporte) error {
	if err := a.Validate(); err != nil {
		log.Printf("level=warn event=validation_failed id_aporte=%s err=%v", a.IDAporte, err)
		return fmt.Errorf("validación fallida: %w", err)
	}

	if err := s.mailer.Send(ctx, a); err != nil {
		log.Printf("level=error event=email_failed id_aporte=%s id_socio=%s email=%s err=%v",
			a.IDAporte, a.IDSocio, a.Correo, err)
		return fmt.Errorf("error enviando correo: %w", err)
	}

	log.Printf("level=info event=aporte_procesado id_aporte=%s id_socio=%s email=%s mes=%s",
		a.IDAporte, a.IDSocio, a.Correo, a.Mes)
	return nil
}
