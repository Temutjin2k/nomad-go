package admin

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
)

type AdminRepository interface {
	GetOverview(ctx context.Context) (*models.OverviewResponse, error)
	GetActiveRides(ctx context.Context, filters models.Filters) (*models.ActiveRidesResponse, error)
}

type Calculator interface {
	Distance(p1, p2 models.Location) float64
}
