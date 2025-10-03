package drivergo

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type Service struct {
	driverRepo Repo
	l          logger.Logger
}

func New(driverRepo Repo, l logger.Logger) *Service {
	return &Service{
		driverRepo: driverRepo,
		l:          l,
	}
}

var (
	validLicenseFmt = regexp.MustCompile(`^[A-Z]{2}\s?[0-9]{6,7}$`)
)

func (s *Service) Register(ctx context.Context, newDriver *models.Driver) error {
	// Проверяю существует ли такой водитель и является ли он водителем
	exist, err := s.driverRepo.IsDriverExist(ctx, newDriver.ID)
	if err != nil {
		s.l.Error(ctx, "Failed to check user role", err)
		return err
	}
	if exist {
		s.l.Warn(ctx, "Driver already registered")
		return types.ErrDriverRegistered
	}

	// Валидация и проверка водительской лицензии по стандартам КЗ
	licenseNum := strings.TrimSpace(newDriver.LicenseNumber)
	if validLicenseFmt.MatchString(licenseNum) {
		uniq, err := s.driverRepo.IsUnique(ctx, licenseNum)
		if err != nil {
			s.l.Error(ctx, "Failed to check license num uniqueness", err)
			return err
		}
		if !uniq {
			return types.ErrLicenseAlreadyExists
		}
		newDriver.IsVerified = true
	} else {

		return types.ErrInvalidLicenseFormat
	}

	// Вычисление рейтинга по виду и другим свойствам машины
	newDriver.Vehicle.Type = classify(newDriver)

	// Дефолты по типу рейтинга и тд
	newDriver.Rating = 5.0
	newDriver.Status = types.OfflineStatus

	// Сохраняю запись в БД
	if err := s.driverRepo.Create(ctx, newDriver); err != nil {
		s.l.Error(ctx, "Failed to create new driver", err)
		return err
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
