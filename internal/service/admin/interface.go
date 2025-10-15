package admin

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
)

type AdminRepository interface {
	GetOverview(ctx context.Context) (*models.OverviewResponse, error)
	GetActiveRides(ctx context.Context) (*models.ActiveRidesResponse, error)
}
