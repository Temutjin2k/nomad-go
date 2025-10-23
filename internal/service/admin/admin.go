package admin

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type AdminService struct {
	adminRepo  AdminRepository
	calculator Calculator

	l logger.Logger
}

func NewAdminService(adminRepo AdminRepository, calculator Calculator, l logger.Logger) *AdminService {
	return &AdminService{
		adminRepo:  adminRepo,
		calculator: calculator,
		l:          l,
	}
}

func (s *AdminService) Overview(ctx context.Context) (*models.OverviewResponse, error) {
	return s.adminRepo.GetOverview(ctx)
}

func (s *AdminService) ActiveRides(ctx context.Context, filters models.Filters) (*models.ActiveRidesResponse, error) {
	res, err := s.adminRepo.GetActiveRides(ctx, filters)
	if err != nil {
		return nil, err
	}

	for i, ride := range res.Rides {
		if ride.CurrentDriverLocation.Latitude == 0 || ride.CurrentDriverLocation.Longitude == 0 || ride.DestinationLocation.Latitude == 0 || ride.DestinationLocation.Longitude == 0 {
			s.l.Warn(ctx, "One or more ride locations have zero value",
				"current_driver_latitude", ride.CurrentDriverLocation.Latitude,
				"current_driver_longitude", ride.CurrentDriverLocation.Longitude,
				"destination_latitude", ride.DestinationLocation.Latitude,
				"destination_longitude", ride.DestinationLocation.Longitude,
				"ride_id", ride.RideID,
			)
			continue
		}

		// Calculate remeining distance
		res.Rides[i].DistanceRemainingKm = s.calculator.Distance(
			ride.CurrentDriverLocation,
			ride.DestinationLocation,
		)
	}

	return res, nil
}
