package drivergo

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
)

type Service struct {
	driverRepo Repo
	trm        trm.TxManager
	l          logger.Logger
}

func New(driverRepo Repo, trm trm.TxManager, l logger.Logger) *Service {
	return &Service{
		driverRepo: driverRepo,
		trm:        trm,
		l:          l,
	}
}

var (
	validLicenseFmt = regexp.MustCompile(`^[A-Z]{2}\s?[0-9]{6,7}$`)
)

func (s *Service) Register(ctx context.Context, newDriver *models.Driver) error {
	licenseNum := strings.TrimSpace(newDriver.LicenseNumber)
	if !validLicenseFmt.MatchString(licenseNum) {
		return wrap.Error(ctx, types.ErrInvalidLicenseFormat)
	}

	fn := func(ctx context.Context) error {
		uniq, err := s.driverRepo.IsUnique(ctx, licenseNum)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check license num uniqueness: %w", err))
		}
		if !uniq {
			return wrap.Error(ctx, types.ErrLicenseAlreadyExists)
		}

		newDriver.IsVerified = true

		exist, err := s.driverRepo.IsDriverExist(ctx, newDriver.ID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check user role: %w", err))
		}
		if exist {
			return wrap.Error(ctx, types.ErrDriverRegistered)
		}

		newDriver.Vehicle.Type = classify(newDriver)
		newDriver.Rating = 5.0
		newDriver.Status = types.OfflineStatus

		if err := s.driverRepo.Create(ctx, newDriver); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to create new driver: %w", err))
		}

		return nil
	}

	if err := s.trm.Do(ctx, fn); err != nil {
		return wrap.Error(ctx, err)
	}

	return nil
}

func classify(driver *models.Driver) types.VehicleClass {
	currentYear := time.Now().Year()
	v := driver.Vehicle

	// Премиум-сегмент (бизнес и люкс)
	premiumBrands := map[string]bool{
		"MERCEDES": true, "BMW": true, "LEXUS": true, "AUDI": true,
		"MASERATI": true, "TESLA": true, "PORSCHE": true, "INFINITI": true,
		"CADILLAC": true, "JAGUAR": true, "RANGE ROVER": true,
		"BENTLEY": true, "ROLLS ROYCE": true,
	}

	makeUpper := strings.ToUpper(v.Make)

	// XL — минивэны, большие SUV и микроавтобусы
	if v.Type == "XL" || v.Type == "VAN" || v.Type == "MINIVAN" ||
		v.Type == "SUV" || v.Type == "CROSSOVER" {
		if v.Year >= currentYear-10 { // не старше 10 лет
			return types.XLClass
		}
	}

	// Премиум
	if premiumBrands[makeUpper] && v.Year >= currentYear-6 {
		return types.PremiumClass
	}

	// Всё остальное → Эконом
	return types.EconomyClass
}
