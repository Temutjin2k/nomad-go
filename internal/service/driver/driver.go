package drivergo

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	ridecalc "github.com/Temutjin2k/ride-hail-system/internal/service/calculator"
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
	repos repos
	logic logic
	infra infra
	l     logger.Logger
}

type logic struct {
	calculate ridecalc.Calculator
}

type infra struct {
	communicator  DriverCommunicator
	addressGetter GeoCoder
	publisher     Publisher
	trm           trm.TxManager
}

type repos struct {
	driver     DriverRepo
	session    DriverSessionRepo
	ride       RideGetter
	user       UserRepo
	coordinate CoordinateRepo
	eventRepo  RideEventRepository
}

// New returns a new instance of the driver service with all dependencies injected.
func New(
	driverRepo DriverRepo,
	sessionRepo DriverSessionRepo,
	coordinateRepo CoordinateRepo,
	userRepo UserRepo,
	rideRepo RideGetter,
	addressGetter GeoCoder,
	publisher Publisher,
	calculate ridecalc.Calculator,
	communicator DriverCommunicator,
	trm trm.TxManager,
	eventRepo RideEventRepository,
	l logger.Logger) *Service {
	return &Service{
		repos: repos{
			driver:     driverRepo,
			session:    sessionRepo,
			coordinate: coordinateRepo,
			user:       userRepo,
			ride:       rideRepo,
		},
		logic: logic{
			calculate: calculate,
		},
		infra: infra{
			addressGetter: addressGetter,
			publisher:     publisher,
			communicator:  communicator,
			trm:           trm,
		},
		l: l,
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
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action: "register_driver",
		UserID: newDriver.ID.String(),
	})

	licenseNum := strings.TrimSpace(newDriver.LicenseNumber)
	if !validLicenseFmt.MatchString(licenseNum) {
		s.l.Debug(ctx, "Invalid license format", "license", newDriver.LicenseNumber)
		return wrap.Error(ctx, types.ErrInvalidLicenseFormat)
	}

	fn := func(ctx context.Context) error {
		// Check license uniqueness
		uniq, err := s.repos.driver.IsLicenseExists(ctx, licenseNum)
		if err != nil {
			return fmt.Errorf("failed to check license num uniqueness: %w", err)
		}
		if !uniq {
			return types.ErrLicenseAlreadyExists
		}

		newDriver.IsVerified = true

		// Prevent duplicate driver registration
		exist, err := s.repos.driver.IsDriverExist(ctx, newDriver.ID)
		if err != nil {
			return fmt.Errorf("failed to check driver existence: %w", err)
		}
		if exist {
			return types.ErrDriverRegistered
		}

		// Determine vehicle class (Economy / XL / Premium)
		newDriver.Vehicle.Type = classify(newDriver)
		newDriver.Rating = 5.0
		newDriver.Status = types.StatusDriverOffline

		// Save new driver record
		if err := s.repos.driver.Create(ctx, newDriver); err != nil {
			return fmt.Errorf("failed to create new driver: %w", err)
		}

		if _, err = s.repos.user.ChangeRole(ctx, newDriver.ID, types.RoleDriver); err != nil {
			return fmt.Errorf("failed to change user role to driver: %w", err)
		}

		return nil
	}

	// Execute inside transaction
	if err := s.infra.trm.Do(ctx, fn); err != nil {
		return wrap.Error(ctx, err)
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
func (s *Service) GoOnline(ctx context.Context, driverID uuid.UUID, location models.Location) (uuid.UUID, error) {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:   "go_online_driver",
		DriverID: driverID.String(),
	})

	var sessionID uuid.UUID
	fn := func(ctx context.Context) error {
		now := time.Now()

		// Check if driver exists in DB
		exist, err := s.repos.driver.IsDriverExist(ctx, driverID)
		if err != nil {
			return fmt.Errorf("failed to check driver existence: %w", err)
		}
		if !exist {
			return types.ErrUserNotFound
		}

		// Change driver status to AVAILABLE
		oldstatus, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverAvailable)
		if err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}
		if oldstatus != types.StatusDriverOffline {
			return types.ErrDriverAlreadyOnline
		}

		// Create a new session for the driver
		sessionID, err = s.repos.session.Create(ctx, driverID)
		if err != nil {
			return fmt.Errorf("failed to create driver session: %w", err)
		}

		// Reverse geocoding: get address by latitude and longitude
		location.Address, err = s.infra.addressGetter.GetAddress(ctx, location.Longitude, location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.repos.coordinate.CreateCoordinate(ctx, driverID, types.Driver, location, now); err != nil {
			return fmt.Errorf("failed to insert new coordinate data: %w", err)
		}

		// Publish driver status
		if err := s.infra.publisher.PublishDriverStatus(
			ctx,
			models.DriverStatusUpdateMessage{
				DriverID:  driverID,
				Status:    types.StatusDriverAvailable.String(),
				Timestamp: now,
				RideID:    nil,
			}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}

		return nil
	}

	// Execute logic within transaction
	if err := s.infra.trm.Do(ctx, fn); err != nil {
		return uuid.UUID{}, wrap.Error(ctx, err)
	}

	return sessionID, nil
}

func (s *Service) GoOffline(ctx context.Context, driverID uuid.UUID) (models.SessionSummary, error) {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:   "go_offline_driver",
		DriverID: driverID.String(),
	})

	var summary models.SessionSummary
	fn := func(ctx context.Context) error {
		now := time.Now()

		// Check if driver exists in DB
		exist, err := s.repos.driver.IsDriverExist(ctx, driverID)
		if err != nil {
			return fmt.Errorf("failed to check driver existence: %w", err)
		}
		if !exist {
			return types.ErrUserNotFound
		}

		// Change driver status to OFFLINE
		oldstatus, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverOffline)
		if err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}

		if oldstatus != types.StatusDriverAvailable {
			if oldstatus == types.StatusDriverOffline {
				return types.ErrDriverAlreadyOffline
			} else {
				return types.ErrDriverMustBeAvailable
			}
		}

		// Get driver`s session summary
		summary, err = s.repos.session.GetSummary(ctx, driverID)
		if err != nil {
			return fmt.Errorf("failed to get session summary: %w", err)
		}

		// Refresh driver total ride summary
		if err := s.repos.driver.UpdateStats(ctx, driverID, summary.RidesCompleted, summary.Earnings); err != nil {
			return fmt.Errorf("failed to update driver stats: %w", err)
		}

		// Publish driver status
		if err := s.infra.publisher.PublishDriverStatus(
			ctx,
			models.DriverStatusUpdateMessage{
				DriverID:  driverID,
				Status:    types.StatusDriverOffline.String(),
				Timestamp: now,
				RideID:    nil,
			}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}

		return nil
	}

	// Execute logic within transaction
	if err := s.infra.trm.Do(ctx, fn); err != nil {
		return models.SessionSummary{}, wrap.Error(ctx, err)
	}

	return summary, nil
}

func (s *Service) StartRide(ctx context.Context, startTime time.Time, driverID, rideID uuid.UUID, location models.Location) error {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:   "start_ride",
		RideID:   rideID.String(),
		DriverID: driverID.String(),
	})

	fn := func(ctx context.Context) error {
		// Get driver data
		driver, err := s.repos.driver.Get(ctx, driverID)
		if err != nil {
			return fmt.Errorf("failed to get driver data: %w", err)
		}

		// Ensure driver is EN_ROUTE to pickup location
		if driver.Status != types.StatusDriverEnRoute {
			return types.ErrDriverMustBeEnRoute
		}

		// Get driver last coordinates
		lastcord, err := s.repos.coordinate.GetDriverLastCoordinate(ctx, driverID)
		if err != nil {
			return fmt.Errorf("failed to get driver last coordinate: %w", err)
		}
		if lastcord.Latitude == 0 && lastcord.Longitude == 0 {
			return types.ErrDriverLocationNotFound
		}

		// Get ride data
		ride, err := s.repos.ride.Get(ctx, rideID)
		if err != nil {
			return fmt.Errorf("failed to get ride data: %w", err)
		}

		// Validate ride status and driver assignment
		if ride.Status != types.StatusArrived.String() {
			return types.ErrRideNotArrived
		}

		// Ensure the ride is assigned to the driver starting it
		if ride.DriverID != nil && *ride.DriverID != driverID {
			return types.ErrRideDriverMismatch
		}

		// Change driver status in database
		if _, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverBusy); err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}

		// Get address by geocoding
		location.Address, err = s.infra.addressGetter.GetAddress(ctx, location.Longitude, location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.repos.coordinate.CreateCoordinate(ctx, driverID, types.Driver, location, time.Now()); err != nil {
			return fmt.Errorf("failed to insert new coordinate data: %w", err)
		}

		// Publish driver status
		if err := s.infra.publisher.PublishDriverStatus(
			ctx,
			models.DriverStatusUpdateMessage{
				DriverID:  driverID,
				Status:    types.StatusInProgress.String(),
				Timestamp: startTime,
				RideID:    &rideID,
			}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}

		return nil
	}

	if err := s.infra.trm.Do(ctx, fn); err != nil {
		return wrap.Error(ctx, err)
	}

	return nil
}

type CompleteRideData struct {
	CompleteTime      time.Time
	DriverID          uuid.UUID
	ActualDurationMin int
	ActualDistanceKm  float64
	Location          models.Location
}

func (s *Service) CompleteRide(ctx context.Context, rideID uuid.UUID, data CompleteRideData) (earnings float64, err error) {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:   "complete_ride",
		RideID:   rideID.String(),
		DriverID: data.DriverID.String(),
	})

	fn := func(ctx context.Context) error {
		// Get Ride data
		ride, err := s.repos.ride.Get(ctx, rideID)
		if err != nil {
			return fmt.Errorf("failed to get ride data: %w", err)
		}
		earnings = ride.EstimatedFare

		// Ride status must be IN_PROGRESS
		if ride.Status != types.StatusInProgress.String() {
			return types.ErrRideNotInProgress
		}

		// Get Driver data
		driver, err := s.repos.driver.Get(ctx, data.DriverID)
		if err != nil {
			return fmt.Errorf("failed to get driver data: %w", err)
		}

		// Driver status must be BUSY
		if driver.Status != types.StatusDriverBusy {
			return types.ErrDriverMustBeBusy
		}

		// Get address by geocoding
		data.Location.Address, err = s.infra.addressGetter.GetAddress(ctx, data.Location.Longitude, data.Location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.repos.coordinate.CreateCoordinate(ctx, data.DriverID, types.Driver, data.Location, data.CompleteTime); err != nil {
			return fmt.Errorf("failed to insert new coordinate data: %w", err)
		}

		// Change driver status to AVAILABLE
		if _, err := s.repos.driver.ChangeStatus(ctx, data.DriverID, types.StatusDriverAvailable); err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}

		// Update driver session: total rides, earnings
		if err := s.repos.session.Update(ctx, data.DriverID, 1, earnings); err != nil {
			return fmt.Errorf("failed to update driver stats: %w", err)
		}

		// Publish driver status update
		if err := s.infra.publisher.PublishDriverStatus(
			ctx,
			models.DriverStatusUpdateMessage{
				DriverID:  data.DriverID,
				Status:    types.StatusCompleted.String(),
				Timestamp: data.CompleteTime,
				RideID:    &rideID,
			}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}

		return nil
	}

	if err := s.infra.trm.Do(ctx, fn); err != nil {
		return 0, wrap.Error(ctx, err)
	}

	return earnings, nil
}

func (s *Service) UpdateLocation(ctx context.Context, data models.RideLocationUpdate) (coordinateID uuid.UUID, err error) {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:   "update_driver_location",
		DriverID: data.DriverID.String(),
		RideID:   data.RideID.String(),
	})

	fn := func(ctx context.Context) error {
		// Check if driver exists in DB
		exist, err := s.repos.driver.IsDriverExist(ctx, data.DriverID)
		if err != nil {
			return fmt.Errorf("failed to check driver existence: %w", err)
		}
		if !exist {
			return types.ErrUserNotFound
		}

		// Get address by geocoding
		data.Location.Address, err = s.infra.addressGetter.GetAddress(ctx, data.Location.Longitude, data.Location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		coordinateID, err = s.repos.coordinate.CreateCoordinate(ctx, data.DriverID, types.Driver, data.Location, data.TimeStamp)
		if err != nil {
			return fmt.Errorf("failed to insert new coordinate data: %w", err)
		}

		if _, err := s.repos.coordinate.CreateLocationHistory(ctx, coordinateID, data.DriverID, data.RideID, data.Location, data.AccuracyMeters, data.SpeedKmh, data.HeadingDegrees); err != nil {
			return fmt.Errorf("failed to create location history: %w", err)
		}

		if err := s.infra.publisher.PublishLocationUpdate(ctx, data); err != nil {
			return fmt.Errorf("failed to publish location update: %w", err)
		}

		return nil
	}

	if err := s.infra.trm.Do(ctx, fn); err != nil {
		return uuid.UUID{}, wrap.Error(ctx, err)
	}

	// записываем ивент
	eventData, _ := json.Marshal(data) // non fatal event so just ignore error
	if err := s.repos.eventRepo.CreateEvent(ctx, *data.RideID, types.EventLocationUpdated, eventData); err != nil {
		s.l.Warn(ctx, "failed to create ride event", "event_type", types.EventLocationUpdated, "error", err.Error())
	}

	return coordinateID, nil
}

func (s *Service) IsExist(ctx context.Context, driverID uuid.UUID) (bool, error) {
	return s.repos.driver.IsDriverExist(ctx, driverID)
}
