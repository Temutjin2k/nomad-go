package ride

import (
	"context"
	"fmt"
	"time"

	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

// создать уникальный номер поездки
func (s *RideService) generateRideNumber(ctx context.Context) (string, error) {
	datePart := time.Now().Format("20060102")

	count, err := s.repo.CountByDate(ctx)
	if err != nil {
		return "", wrap.Error(ctx, err)
	}
	nextSequence := count + 1
	return fmt.Sprintf("RIDE_%s_%03d", datePart, nextSequence), nil
}
