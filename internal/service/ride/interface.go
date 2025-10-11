package ride

import (
	"context"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideRepo interface {
	Create(ctx context.Context, ride *models.Ride) (*models.Ride, error)
	Update(ctx context.Context, ride *models.Ride) error
	FindByID(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
	
	// для генерации уникального номера поездки (ride_number)
	CountByDate(ctx context.Context, date time.Time) (int, error)
}