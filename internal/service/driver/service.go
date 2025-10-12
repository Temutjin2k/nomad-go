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
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

/*
Service provides all business logic for driver management,
including registration, session handling, coordinate storage, etc.
*/
type Service struct {
	driverRepo     DriverRepo
	sessionRepo    DriverSessionRepo
	coordinateRepo CoordinateRepo
	addressGetter  AddresGetter
	trm            trm.TxManager
	l              logger.Logger
}

// New returns a new instance of the driver service with all dependencies injected.
func New(driverRepo DriverRepo, sessionRepo DriverSessionRepo, coordinateRepo CoordinateRepo, addressGetter AddresGetter, trm trm.TxManager, l logger.Logger) *Service {
	return &Service{
		driverRepo:     driverRepo,
		sessionRepo:    sessionRepo,
		coordinateRepo: coordinateRepo,
		addressGetter:  addressGetter,
		trm:            trm,
		l:              l,
	}
}

var (
	// validLicenseFmt ensures the license number matches a specific pattern:
	// e.g. "AB1234567" or "AB 123456".
	validLicenseFmt = regexp.MustCompile(`^[A-Z]{2}\s?[0-9]{6,7}$`)
)

// Register handles new driver registration with license validation,
// duplicate checks, and initial driver setup.
func (s *Service) Register(ctx context.Context, newDriver *models.Driver) error {
	licenseNum := strings.TrimSpace(newDriver.LicenseNumber)
	if !validLicenseFmt.MatchString(licenseNum) {
		return wrap.Error(ctx, types.ErrInvalidLicenseFormat)
	}

	fn := func(ctx context.Context) error {
		// Check license uniqueness
		uniq, err := s.driverRepo.IsUnique(ctx, licenseNum)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check license num uniqueness: %w", err))
		}
		if !uniq {
			return wrap.Error(ctx, types.ErrLicenseAlreadyExists)
		}

		newDriver.IsVerified = true

		// Prevent duplicate driver registration
		exist, err := s.driverRepo.IsDriverExist(ctx, newDriver.ID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check driver existence: %w", err))
		}
		if exist {
			return wrap.Error(ctx, types.ErrDriverRegistered)
		}

		// Determine vehicle class (Economy / XL / Premium)
		newDriver.Vehicle.Type = classify(newDriver)
		newDriver.Rating = 5.0
		newDriver.Status = types.StatusDriverOffline

		// Save new driver record
		if err := s.driverRepo.Create(ctx, newDriver); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to create new driver: %w", err))
		}

		return nil
	}

	// Execute inside transaction
	if err := s.trm.Do(ctx, fn); err != nil {
		return err
	}

	return nil
}

// classify determines the vehicle class (Economy, XL, Premium)
// based on car type, brand, and year.
func classify(driver *models.Driver) types.VehicleClass {
	currentYear := time.Now().Year()
	v := driver.Vehicle

	// Premium segment (luxury and business brands)
	premiumBrands := map[string]bool{
		"MERCEDES": true, "BMW": true, "LEXUS": true, "AUDI": true,
		"MASERATI": true, "TESLA": true, "PORSCHE": true, "INFINITI": true,
		"CADILLAC": true, "JAGUAR": true, "RANGE ROVER": true,
		"BENTLEY": true, "ROLLS ROYCE": true,
	}

	makeUpper := strings.ToUpper(v.Make)

	// XL – vans, SUVs, crossovers
	if v.Type == "XL" || v.Type == "VAN" || v.Type == "MINIVAN" ||
		v.Type == "SUV" || v.Type == "CROSSOVER" {
		if v.Year >= currentYear-10 { // not older than 10 years
			return types.ClassXL
		}
	}

	// Premium class – luxury brands under 6 years old
	if premiumBrands[makeUpper] && v.Year >= currentYear-6 {
		return types.ClassPremium
	}

	// Default → Economy
	return types.ClassEconomy
}

// GoOnline puts a driver into "available" mode, creates a session,
// and saves the driver’s current coordinates.
func (s *Service) GoOnline(ctx context.Context, driverID uuid.UUID, latitude, longitude float64) (uuid.UUID, error) {
	var sessionID uuid.UUID

	fn := func(ctx context.Context) error {
		// Check if driver exists in DB
		exist, err := s.driverRepo.IsDriverExist(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check driver existence: %w", err))
		}
		if !exist {
			return wrap.Error(ctx, types.ErrUserNotFound)
		}

		// Change driver status to AVAILABLE
		oldstatus, err := s.driverRepo.ChangeStatus(ctx, driverID, types.StatusDriverAvailable)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to change driver status: %w", err))
		}
		if oldstatus == types.StatusDriverAvailable {
			return types.ErrDriverAlreadyOnline
		}

		// Create a new session for the driver
		sessionID, err = s.sessionRepo.Create(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to create driver session: %w", err))
		}

		// Reverse geocoding: get address by latitude and longitude
		address, err := s.addressGetter.GetAddress(ctx, longitude, latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.coordinateRepo.Create(ctx, driverID, types.Driver, address, latitude, longitude); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to insert new coordinate data: %w", err))
		}

		return nil
	}

	// Execute logic within transaction
	if err := s.trm.Do(ctx, fn); err != nil {
		return uuid.UUID{}, wrap.Error(ctx, err)
	}

	return sessionID, nil
}
