package drivergo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

func (s *Service) SearchDriver(ctx context.Context, req models.RideRequestedMessage) error {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:     "search_driver",
		RideID:     req.RideID.String(),
		RideNumber: req.RideNumber,
		RequestID:  wrap.GetRequestID(ctx),
	})

	offer := s.prepareRideOffer(req)

	return s.waitForDriverAcceptance(ctx, req, offer)
}

// Формируем оффер один раз
func (s *Service) prepareRideOffer(req models.RideRequestedMessage) models.RideOffer {
	distance := s.logic.calculate.Distance(models.Location{
		Latitude:  req.PickupLocation.Latitude,
		Longitude: req.PickupLocation.Longitude,
	}, models.Location{
		Latitude:  req.DestinationLocation.Latitude,
		Longitude: req.DestinationLocation.Longitude,
	})
	durationMin := s.logic.calculate.Duration(distance)

	return models.RideOffer{
		ID:                          uuid.New(),
		MsgType:                     "ride_offer",
		RideID:                      req.RideID,
		RideNumber:                  req.RideNumber,
		PickupLocation:              req.PickupLocation,
		DestinationLocation:         req.DestinationLocation,
		EstimatedFare:               req.EstimatedFare,
		EstimatedRideDurationMinute: durationMin,
		DriverEarnings:              s.logic.calculate.Fare(req.RideType, distance, durationMin),
		ExpiresAt:                   time.Now().Add(30 * time.Second),
		DistanceToPickupKm:          0,
	}
}

// Поиск доступных водителей
func (s *Service) searchAvailableDrivers(ctx context.Context, rideType string, loc models.Location) ([]models.DriverWithDistance, error) {
	drivers, err := s.repos.driver.SearchDrivers(ctx, rideType, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to find available drivers: %w", err)
	}
	if len(drivers) == 0 {
		return nil, types.ErrDriversNotFound
	}
	return drivers, nil
}

// Отправка оффера водителю и обработка принятия
func (s *Service) offerRideToDriver(ctx context.Context, correlationID string, driver models.DriverWithDistance, offer models.RideOffer) (bool, error) {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		DriverID: driver.ID.String(),
		OfferID:  offer.ID.String(),
	})

	s.l.Info(ctx, "sending offer to driver")
	offer.DistanceToPickupKm = driver.DistanceKm

	accepted, err := s.infra.communicator.SendRideOffer(ctx, driver.ID, offer)
	if err != nil {
		s.l.Debug(ctx, "failed to send ride offer", "error", err)
		return false, nil // игнорируем ошибки отправки для поиска других водителей
	}

	if !accepted {
		s.l.Info(ctx, "driver declined or timeout")
		return false, nil
	}

	// Пытаемся заблокировать водителя
	if err := s.infra.trm.Do(ctx, func(ctx context.Context) error {
		old, err := s.repos.driver.ChangeStatus(ctx, driver.ID, types.StatusDriverBusy)
		if err != nil {
			s.l.Error(ctx, "failed to change driver status", err)
			return err
		}
		if old == types.StatusDriverBusy {
			s.l.Error(ctx, "driver is already busy", types.ErrDriverAlreadyBusy)
			return types.ErrDriverAlreadyBusy
		}

		// Publish driver response
		if err := s.infra.publisher.PublishDriverResponse(ctx, models.DriverMatchResponse{
			RideID:                  offer.RideID,
			DriverID:                driver.ID,
			Accepted:                true,
			EstimatedArrivalMinutes: s.logic.calculate.Duration(driver.DistanceKm),
			DriverLocation:          driver.Location,
			CorrelationID:           correlationID,
			DriverInfo: models.DriverInfo{
				Name:    driver.Name,
				Rating:  driver.Rating,
				Vehicle: driver.Vehicle,
			},
		}); err != nil {
			s.l.Error(ctx, "failed to publish driver response", err)
			return err
		}

		// Publish driver status update
		if err := s.infra.publisher.PublishDriverStatus(ctx, models.DriverStatusUpdateMessage{
			DriverID:  driver.ID,
			Status:    types.StatusDriverBusy.String(),
			Timestamp: time.Now(),
			RideID:    &offer.RideID,
		}); err != nil {
			s.l.Error(ctx, "failed to publish driver status", err)
			return err
		}
		return nil
	}); err != nil {
		return false, err
	}

	s.l.Info(ctx, "driver accepted the ride offer")
	return true, nil
}

// Основной цикл поиска водителя с тикером и таймером
func (s *Service) waitForDriverAcceptance(ctx context.Context, req models.RideRequestedMessage, offer models.RideOffer) error {
	timer := time.NewTimer(time.Hour * 24)
	tick := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	defer tick.Stop()

	trySearch := func() (bool, error) {
		loc := models.Location{
			Latitude:  req.PickupLocation.Latitude,
			Longitude: req.PickupLocation.Longitude,
			Address:   req.PickupLocation.Address,
		}

		drivers, err := s.searchAvailableDrivers(ctx, req.RideType, loc)
		if err != nil {
			return false, err
		}

		for _, driver := range drivers {
			accepted, _ := s.offerRideToDriver(ctx, req.CorrelationID, driver, offer)
			if accepted {
				return true, nil
			}
		}
		return false, nil
	}

	// Первая попытка сразу
	accepted, err := trySearch()
	if err != nil {
		return wrap.Error(ctx, err)
	}
	if accepted {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("driver search stop: (ctx Done)")
		case <-timer.C:
			return types.ErrDriverSearchTimeout
		case <-tick.C:
			accepted, err := trySearch()
			if err != nil {
				if errors.Is(err, types.ErrDriversNotFound) {
					s.l.Warn(ctx, "Drivers are not found")
					continue
				}
				return wrap.Error(ctx, err)
			}
			if accepted {
				return nil
			}
		}
	}
}

func (s *Service) HandleRideStatus(ctx context.Context, req models.RideStatusUpdateMessage) error {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action: "match_driver",
		RideID: req.RideID.String(),
	})

	if req.DriverID == nil {
		return wrap.Error(ctx, types.ErrDriverIDNotExist)
	}
	ctx = wrap.WithDriverID(ctx, req.DriverID.String())

	switch req.Status {
	case types.StatusCancelled.String():
		if err := s.cancelRide(ctx, *req.DriverID, req.RideID); err != nil {
			return wrap.Error(ctx, err)
		}

	case types.StatusMatched.String():
		if err := s.processMatchedRide(ctx, *req.DriverID, req.RideID); err != nil {
			return wrap.Error(ctx, err)
		}

		finalDest, err := s.repos.ride.GetPickupCoordinate(ctx, req.RideID)
		if err != nil {
			return wrap.Error(ctx, err)
		}

		// Track him in a real time
		ctx, cancel := context.WithCancel(ctx)
		if err := s.infra.communicator.ListenLocationUpdates(ctx, *req.DriverID, req.RideID,
			func(ctx context.Context, current models.RideLocationUpdate) error {
				return s.processDriverLocation(ctx, cancel, current, *finalDest)
			}); err != nil {
			return wrap.Error(ctx, err)
		}

	default:
		s.l.Warn(ctx, "unsupported ride status update", "status", req.Status)
	}
	return nil
}

func (s *Service) cancelRide(ctx context.Context, driverID, rideID uuid.UUID) error {
	return s.infra.trm.Do(ctx, func(ctx context.Context) error {
		if _, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverAvailable); err != nil {
			return fmt.Errorf("failed to change driver status to available after ride cancellation: %w", err)
		}

		if err := s.infra.publisher.PublishDriverStatus(ctx, models.DriverStatusUpdateMessage{
			DriverID:  driverID,
			Status:    types.StatusDriverAvailable.String(),
			Timestamp: time.Now(),
			RideID:    &rideID,
		}); err != nil {
			return fmt.Errorf("failed to publish driver status after ride cancellation: %w", err)
		}
		return nil
	})
}

func (s *Service) processMatchedRide(ctx context.Context, driverID, rideID uuid.UUID) error {
	return s.infra.trm.Do(ctx, func(ctx context.Context) error {
		// Get Ride ID
		details, err := s.repos.ride.GetDetails(ctx, rideID)
		if err != nil {
			return fmt.Errorf("failed to get ride: %w", err)
		}

		if _, err := s.repos.driver.ChangeStatus(ctx, details.DriverID, types.StatusDriverEnRoute); err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}

		// Send ride details(pickup location, navigation)
		if err := s.infra.communicator.SendRideDetails(ctx, *details); err != nil {
			return fmt.Errorf("failed to send ride details: %w", err)
		}

		if err := s.infra.publisher.PublishDriverStatus(ctx, models.DriverStatusUpdateMessage{
			DriverID:  driverID,
			Status:    types.StatusDriverEnRoute.String(),
			Timestamp: time.Now(),
			RideID:    &details.RideID,
		}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}

		return nil
	})
}

func (s *Service) processDriverLocation(ctx context.Context, cancel context.CancelFunc, current models.RideLocationUpdate, destination models.Location) error {
	if _, err := s.UpdateLocation(ctx, current); err != nil {
		return err
	}

	if !s.logic.calculate.IsDriverArrived(current.Location.Latitude, current.Location.Longitude, destination.Latitude, destination.Longitude) {
		return nil
	}

	return s.infra.trm.Do(ctx, func(ctx context.Context) error {
		if _, err := s.repos.driver.ChangeStatus(ctx, current.DriverID, types.DriverStatus(types.StatusArrived)); err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}

		if err := s.infra.publisher.PublishDriverStatus(
			ctx,
			models.DriverStatusUpdateMessage{
				DriverID:  current.DriverID,
				Status:    types.StatusArrived.String(),
				Timestamp: time.Now(),
				RideID:    current.RideID,
			}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}
		cancel()
		return nil
	})
}
