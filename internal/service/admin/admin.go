package admin

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type AdminService struct {
	adminRepo AdminRepository
	l         logger.Logger
}

func NewAdminService(adminRepo AdminRepository, l logger.Logger) *AdminService {
	return &AdminService{
		adminRepo: adminRepo,
		l:         l,
	}
}

func (s *AdminService) GetOverview(ctx context.Context) (*models.OverviewResponse, error) {
	return s.adminRepo.GetOverview(ctx)
}

func (s *AdminService) GetActiveRides(ctx context.Context) (*models.ActiveRidesResponse, error) {
	res, err := s.adminRepo.GetActiveRides(ctx)
	if err != nil {
		return nil, err
	}

	if len(res.Rides) == 0 {
		return nil, types.ErrRideNotFound
	}

	for i, ride := range res.Rides {
		// Calculate DistanceRemainingKm using Haversine formula
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
		res.Rides[i].DistanceRemainingKm = HaversineDistance(
			ride.CurrentDriverLocation.Latitude, ride.CurrentDriverLocation.Longitude,
			ride.DestinationLocation.Latitude, ride.DestinationLocation.Longitude,
		)
	}

	return res, nil
}
