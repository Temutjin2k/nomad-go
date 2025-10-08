package admin

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
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

func (s *AdminService) GetOverview(ctx context.Context) (any, error) {
	return s.adminRepo.GetOverview(ctx)
}

func (s *AdminService) GetActiveRides(ctx context.Context) (*models.ActiveRidesResponse, error) {
	res, err := s.adminRepo.GetActiveRides(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}
