package drivergo

import (
	"context"
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
	repos         repos
	publisher     Publisher
	addressGetter GeoCoder
	calculate     *ridecalc.Calculator
	trm           trm.TxManager
	l             logger.Logger
}

type repos struct {
	driver     DriverRepo
	session    DriverSessionRepo
	ride       RideRepo
	user       UserRepo
	coordinate CoordinateRepo
}

// New returns a new instance of the driver service with all dependencies injected.
func New(driverRepo DriverRepo, sessionRepo DriverSessionRepo, coordinateRepo CoordinateRepo, userRepo UserRepo, rideRepo RideRepo, addressGetter GeoCoder, publisher Publisher, calculate *ridecalc.Calculator, trm trm.TxManager, l logger.Logger) *Service {
	return &Service{
		repos: repos{
			driver:     driverRepo,
			session:    sessionRepo,
			coordinate: coordinateRepo,
			user:       userRepo,
			ride:       rideRepo,
		},
		addressGetter: addressGetter,
		publisher:     publisher,
		calculate:     calculate,
		trm:           trm,
		l:             l,
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
		ctx = wrap.WithAction(ctx, "register_driver")

		// Check license uniqueness
		uniq, err := s.repos.driver.IsUnique(ctx, licenseNum)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check license num uniqueness: %w", err))
		}
		if !uniq {
			return wrap.Error(ctx, types.ErrLicenseAlreadyExists)
		}

		newDriver.IsVerified = true

		// Prevent duplicate driver registration
		exist, err := s.repos.driver.IsDriverExist(ctx, newDriver.ID)
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
		if err := s.repos.driver.Create(ctx, newDriver); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to create new driver: %w", err))
		}

		if _, err = s.repos.user.ChangeRole(ctx, newDriver.ID, types.RoleDriver); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to change user role to driver: %w", err))
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
func (s *Service) GoOnline(ctx context.Context, driverID uuid.UUID, location models.Location) (uuid.UUID, error) {
	var sessionID uuid.UUID

	fn := func(ctx context.Context) error {
		ctx = wrap.WithAction(ctx, "go_online_driver")

		// Check if driver exists in DB
		exist, err := s.repos.driver.IsDriverExist(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check driver existence: %w", err))
		}
		if !exist {
			return wrap.Error(ctx, types.ErrUserNotFound)
		}

		// Change driver status to AVAILABLE
		oldstatus, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverAvailable)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to change driver status: %w", err))
		}
		if oldstatus != types.StatusDriverOffline {
			return types.ErrDriverAlreadyOnline
		}

		// Create a new session for the driver
		sessionID, err = s.repos.session.Create(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to create driver session: %w", err))
		}

		// Reverse geocoding: get address by latitude and longitude
		address, err := s.addressGetter.GetAddress(ctx, location.Longitude, location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.repos.coordinate.Create(ctx, driverID, types.Driver, address, location.Latitude, location.Longitude); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to insert new coordinate data: %w", err))
		}

		return nil
	}

	// Execute logic within transaction
	if err := s.trm.Do(ctx, fn); err != nil {
		return uuid.UUID{}, err
	}

	return sessionID, nil
}

func (s *Service) GoOffline(ctx context.Context, driverID uuid.UUID) (models.SessionSummary, error) {
	var summary models.SessionSummary

	fn := func(ctx context.Context) error {
		ctx = wrap.WithAction(ctx, "go_offline_driver")

		// Check if driver exists in DB
		exist, err := s.repos.driver.IsDriverExist(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to check driver existence: %w", err))
		}
		if !exist {
			return wrap.Error(ctx, types.ErrUserNotFound)
		}

		// Change driver status to OFFLINE
		oldstatus, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverOffline)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to change driver status: %w", err))
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
			return wrap.Error(ctx, fmt.Errorf("failed to get session summary: %w", err))
		}

		// Refresh driver total ride summary
		if err := s.repos.driver.UpdateStats(ctx, driverID, summary.RidesCompleted, summary.Earnings); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to update driver stats: %w", err))
		}

		return nil
	}

	// Execute logic within transaction
	if err := s.trm.Do(ctx, fn); err != nil {
		return models.SessionSummary{}, err
	}

	return summary, nil
}

func (s *Service) StartRide(ctx context.Context, startTime time.Time, driverID, rideID uuid.UUID, location models.Location) error {
	fn := func(ctx context.Context) error {
		ctx = wrap.WithAction(ctx, "start_ride")

		// Get driver data
		driver, err := s.repos.driver.Get(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to get driver data: %w", err))
		}

		// Ensure driver is EN_ROUTE to pickup location
		if driver.Status != types.StatusDriverEnRoute {
			return types.ErrDriverMustBeEnRoute
		}

		// Get driver last coordinates
		lastcord, err := s.repos.coordinate.GetDriverLastCoordinate(ctx, driverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to get driver last coordinate: %w", err))
		}
		if lastcord.Latitude == 0 && lastcord.Longitude == 0 {
			return types.ErrDriverLocationNotFound
		}

		// Get ride data
		ride, err := s.repos.ride.Get(ctx, rideID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to get ride data: %w", err))
		}

		// Validate ride status and driver assignment
		if ride.Status != types.StatusArrived {
			return types.ErrRideNotArrived
		}

		// Ensure the ride is assigned to the driver starting it
		if ride.DriverID != nil && *ride.DriverID != driverID {
			return types.ErrRideDriverMismatch
		}

		// Calculate estimated arrival time to pickup location
		estimatedArrival := s.calculate.EstimatedArrival(location.Latitude, location.Longitude, lastcord.Latitude, lastcord.Longitude, driver.Vehicle.Type)

		// Change driver status in database
		if _, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverBusy); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to change driver status: %w", err))
		}

		// Get address by geocoding
		address, err := s.addressGetter.GetAddress(ctx, location.Longitude, location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.repos.coordinate.Create(ctx, driverID, types.Driver, address, location.Latitude, location.Longitude); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to insert new coordinate data: %w", err))
		}

		// Update ride status to IN_PROGRESS
		// and create a ride event
		if err := s.repos.ride.StartRide(
			ctx, rideID, driverID, startTime,
			models.RideEvent{
				OldStatus:        types.StatusArrived,
				NewStatus:        types.StatusInProgress,
				DriverID:         driverID,
				Location:         models.Location{Latitude: location.Latitude, Longitude: location.Longitude},
				EstimatedArrival: estimatedArrival,
			}); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to start ride: %w", err))
		}

		// Extract correlation_id from context for event tracing
		logCtx, _ := ctx.Value(wrap.LogCtxKey).(wrap.LogCtx)

		// Publish ride status
		if err := retry(5, 2*time.Second, func() error {
			return s.publisher.PublishRideStatus(
				ctx,
				models.RideStatusUpdateMessage{
					RideID:        rideID,
					Status:        types.StatusInProgress,
					Timestamp:     startTime,
					CorrelationID: logCtx.RequestID,
					// FinalFare will be set when the ride is completed
					FinalFare: 0,
				})
		}); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to publish ride status: %w", err))
		}

		// Publish driver status
		if err := retry(5, 2*time.Second, func() error {
			return s.publisher.PublishDriverStatus(
				ctx,
				models.DriverStatusUpdateMessage{
					DriverID:  driverID,
					Status:    types.StatusDriverBusy,
					Timestamp: startTime,
					RideID:    rideID,
				})
		}); err != nil {
			s.l.Warn(ctx, "failed to publish driver status", "error", err.Error())
		}

		return nil
	}

	if err := s.trm.Do(ctx, fn); err != nil {
		return err
	}

	return nil
}

func retry(n int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < n; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return err
}

type CompleteRideData struct {
	CompleteTime      time.Time
	DriverID          uuid.UUID
	ActualDurationMin int
	ActualDistanceKm  float64
	Location          models.Location
}

func (s *Service) CompleteRide(ctx context.Context, rideID uuid.UUID, data CompleteRideData) (earnings float64, err error) {
	fn := func(ctx context.Context) error {
		ctx = wrap.WithAction(ctx, "complete_ride")

		// Get Ride data
		ride, err := s.repos.ride.Get(ctx, rideID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to get ride data: %w", err))
		}

		// Ride status must be IN_PROGRESS
		if ride.Status != types.StatusInProgress {
			return types.ErrRideNotInProgress
		}

		// Get Driver data
		driver, err := s.repos.driver.Get(ctx, data.DriverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to get driver data: %w", err))
		}

		// Driver status must be BUSY
		if driver.Status != types.StatusDriverBusy {
			return types.ErrDriverMustBeBusy
		}

		// Get driver last coordinates
		lastcord, err := s.repos.coordinate.GetDriverLastCoordinate(ctx, data.DriverID)
		if err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to get driver last coordinate: %w", err))
		}

		// Get address by geocoding
		address, err := s.addressGetter.GetAddress(ctx, data.Location.Longitude, data.Location.Latitude)
		if err != nil {
			s.l.Warn(ctx, "Failed to get address", "error", err.Error())
		}

		// Save driver’s coordinates in the DB
		if _, err := s.repos.coordinate.Create(ctx, data.DriverID, types.Driver, address, data.Location.Latitude, data.Location.Longitude); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to insert new coordinate data: %w", err))
		}

		// Calculate fare
		distance := s.calculate.Distance(models.Location{
			Latitude:  data.Location.Latitude,
			Longitude: data.Location.Longitude,
		},
			models.Location{
				Latitude:  lastcord.Latitude,
				Longitude: lastcord.Longitude,
			},
		)
		durationMin := s.calculate.Duration(distance)
		earnings = s.calculate.Fare(ride.RideType, distance, durationMin)

		// Update ride status to COMPLETED
		if err := s.repos.ride.CompleteRide(ctx, rideID, data.DriverID, data.CompleteTime,
			models.RideEvent{
				OldStatus: types.StatusInProgress,
				NewStatus: types.StatusCompleted,
				DriverID:  data.DriverID,
				Location:  data.Location,
			}); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to complete ride:%w", err))
		}

		// Change driver status to AVAILABLE
		if _, err := s.repos.driver.ChangeStatus(ctx, data.DriverID, types.StatusDriverAvailable); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to change driver status: %w", err))
		}

		// Update driver stats: total rides, earnings
		if err := s.repos.driver.UpdateStats(ctx, data.DriverID, 1, earnings); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to update driver stats: %w", err))
		}

		// Publish ride status update
		if err := retry(5, 2*time.Second, func() error {
			return s.publisher.PublishRideStatus(
				ctx,
				models.RideStatusUpdateMessage{
					RideID:        rideID,
					Status:        types.StatusCompleted,
					Timestamp:     data.CompleteTime,
					CorrelationID: "",
					FinalFare:     earnings,
				})
		}); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to publish ride status: %w", err))
		}

		// Publish driver status update
		if err := retry(5, 2*time.Second, func() error {
			return s.publisher.PublishDriverStatus(
				ctx,
				models.DriverStatusUpdateMessage{
					DriverID:  data.DriverID,
					Status:    types.StatusDriverAvailable,
					Timestamp: data.CompleteTime,
					RideID:    rideID,
				})
		}); err != nil {
			s.l.Warn(ctx, "failed to publish driver status", "error", err.Error())
		}

		return nil
	}

	if err := s.trm.Do(ctx, fn); err != nil {
		return 0, err
	}

	return earnings, nil
}
