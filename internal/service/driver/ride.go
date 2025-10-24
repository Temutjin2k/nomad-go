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
	ctx = wrap.WithAction(ctx, "search_driver")

	offer := s.prepareRideOffer(req)

	return s.waitForDriverAcceptance(ctx, req, offer)
}

// Формируем оффер один раз
func (s *Service) prepareRideOffer(req models.RideRequestedMessage) models.RideOffer {
	distance := s.logic.calculate.Distance(models.Location{
		Latitude:  req.PickupLocation.Lat,
		Longitude: req.PickupLocation.Lng,
	}, models.Location{
		Latitude:  req.DestinationLocation.Lat,
		Longitude: req.DestinationLocation.Lng,
	})
	durationMin := s.logic.calculate.Duration(distance)

	return models.RideOffer{
		ID:                          uuid.New(),
		MsgType:                     "ride_offer",
		RideID:                      uuid.New(),
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
		return nil, wrap.Error(ctx, fmt.Errorf("failed to find available drivers: %w", err))
	}
	if len(drivers) == 0 {
		return nil, types.ErrDriversNotFound
	}
	return drivers, nil
}

// Отправка оффера водителю и обработка принятия
func (s *Service) offerRideToDriver(ctx context.Context, correlationID string, driver models.DriverWithDistance, offer models.RideOffer) (bool, error) {
	offer.DistanceToPickupKm = driver.DistanceKm
	s.l.Info(ctx, "sending offer to driver", "driver_id", driver.ID, "offer_id", offer.ID)

	accepted, err := s.infra.sender.SendRideOffer(ctx, driver.ID, offer)
	if err != nil {
		s.l.Debug(ctx, "failed to send ride offer", err, "driver_id", driver.ID)
		return false, nil // игнорируем ошибки отправки для поиска других водителей
	}

	if !accepted {
		s.l.Info(ctx, "driver declined or timeout", "driver_id", driver.ID)
		return false, nil
	}

	// Пытаемся заблокировать водителя
	if err := s.infra.trm.Do(ctx, func(ctx context.Context) error {
		old, err := s.repos.driver.ChangeStatus(ctx, driver.ID, types.StatusDriverBusy)
		if err != nil {
			s.l.Error(ctx, "failed to change driver status", err, "driver_id", driver.ID)
			return err
		}
		if old == types.StatusDriverBusy {
			s.l.Error(ctx, "driver is already busy", types.ErrDriverAlreadyBusy)
			return types.ErrDriverAlreadyBusy
		}

		if err = s.infra.publisher.PublishDriverResponse(ctx, models.DriverMatchResponse{
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

		if err := s.infra.publisher.PublishDriverStatus(ctx, models.DriverStatusUpdateMessage{
			DriverID:  driver.ID,
			Status:    types.StatusDriverBusy,
			RideID:    offer.RideID,
			Timestamp: time.Now(),
		}); err != nil {
			s.l.Error(ctx, "failed to publish driver status", err)
		}
		return nil
	}); err != nil {
		return false, err
	}

	s.l.Info(ctx, "driver accepted the ride offer", "driver_id", driver.ID)
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
			Latitude:  req.PickupLocation.Lat,
			Longitude: req.PickupLocation.Lng,
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
		return err
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
					break
				}
				return err
			}
			if accepted {
				return nil
			}
		}
	}
}

func (s *Service) MatchDriver(ctx context.Context) {
	// Consume confirmation data from queue

	// Check if driver is not busy

	// Send ride details(pickup location, navigation)

	// Change status to en_route

	// Track him in a real time
	// ListenUpdates(ctx, driverID)
}

func (s *Service) ListenUpdates(ctx context.Context, driverID uuid.UUID) {
	// Open message receiver
	// Dont forget about graceful shutdown
	// Also listen driver status, if he is available, return function

	// Handle messages by type:
	/*
		// Simple example:
			switch msg{
				case UpdateMsg:
					// Driver must send coordinates every 3-5 seconds(problem at client side)

					// Receive location data
					// Update coordinates in database

					// Publish to location_fanout
				case ArriveMsg:
					// Receive arrive msg
					// Publish status change
					return
			}
	*/

}
