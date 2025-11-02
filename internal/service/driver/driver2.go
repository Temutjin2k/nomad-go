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
	offer := s.prepareRideOffer(req)

	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:     "search_driver",
		RideID:     req.RideID.String(),
		RideNumber: req.RideNumber,
		RequestID:  wrap.GetRequestID(ctx),
		OfferID:    offer.ID.String(),
	})

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

	accepted, err := s.infra.communicator.GetRideOffer(ctx, driver.ID, offer)
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
		return nil
	}); err != nil {
		return false, err
	}

	s.l.Info(ctx, "driver accepted the ride offer")
	return true, nil
}

// Основной цикл поиска водителя с тикером и таймером
func (s *Service) waitForDriverAcceptance(ctx context.Context, req models.RideRequestedMessage, offer models.RideOffer) error {
	// общий таймаут поиска
	searchTimeout := 2 * time.Minute
	// интервал между попытками (отсчитывается после каждой попытки)
	interval := 5 * time.Second

	timeout := time.NewTimer(searchTimeout)
	defer timeout.Stop()

	// timer для интервальных попыток, стартуем, но будем сбрасывать после первой итерации
	tick := time.NewTimer(interval)
	// если хотим, чтобы первая попытка была немедленной — можно остановить tick сейчас,
	// а потом Reset после первой попытки. Но здесь мы просто сбросим его после первой попытки.
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
		s.l.Warn(ctx, "driver first search attempt failed", "error", err)
	}
	if accepted {
		return nil
	}

	// Так как мы уже сделали первую попытку, нужно сбросить tick (отсчитать интервал заново).
	// Если timer ещё не истёк — безопасно остановим и сбросим.
	if !tick.Stop() {
		// Если канал уже сработал и значение ещё не прочитано — прочитаем его, чтобы не заблокировать Reset.
		select {
		case <-tick.C:
		default:
		}
	}
	tick.Reset(interval)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("driver search stop: (ctx Done)")
		case <-timeout.C:
			return types.ErrDriverSearchTimeout
		case <-tick.C:
			accepted, err := trySearch()
			if err != nil {
				s.l.Warn(ctx, fmt.Sprintf("driver search attempt failed: %v", err))

				// Просто продолжаем искать — сбрасываем интервал и идём дальше
				if !tick.Stop() {
					select {
					case <-tick.C:
					default:
					}
				}
				tick.Reset(interval)
				continue
			}

			if accepted {
				return nil
			}

			// Не найдено — сбрасываем интервал и продолжаем
			if !tick.Stop() {
				select {
				case <-tick.C:
				default:
				}
			}
			tick.Reset(interval)
		}
	}
}

// HandleRideStatus обрабатывает статусы поездки
func (s *Service) HandleRideStatus(ctx context.Context, req models.RideStatusUpdateMessage) error {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action: "match_driver",
		RideID: req.RideID.String(),
	})

	// if driverID not provided search from database
	if req.DriverID == nil {
		ride, err := s.repos.ride.Get(ctx, req.RideID)
		if err != nil {
			return wrap.Error(ctx, err)
		}

		if ride.DriverID == nil {
			return errors.New("driver id not found")
		}
		req.DriverID = ride.DriverID
	}

	switch req.Status {
	case types.StatusCancelled.String():
		if err := s.cancelRide(ctx, *req.DriverID); err != nil {
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
		if err := s.infra.communicator.ListenLocationUpdates(ctx, *req.DriverID, req.RideID,
			func(ctx context.Context, current models.RideLocationUpdate) error {
				return s.processDriverLocation(ctx, current, *finalDest)
			}); err != nil {
			return wrap.Error(ctx, err)
		}

	default:
		s.l.Warn(ctx, "unsupported ride status update", "status", req.Status)
	}
	return nil
}

func (s *Service) cancelRide(ctx context.Context, driverID uuid.UUID) error {
	if _, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverAvailable); err != nil {
		return fmt.Errorf("failed to change driver status to available after ride cancellation: %w", err)
	}
	return nil
}

func (s *Service) processMatchedRide(ctx context.Context, driverID, rideID uuid.UUID) error {
	return s.infra.trm.Do(ctx, func(ctx context.Context) error {
		// Get Ride ID
		details, err := s.repos.ride.GetDetails(ctx, rideID)
		if err != nil {
			return fmt.Errorf("failed to get ride: %w", err)
		}

		if details.DriverID == nil {
			return errors.New("driver id not found")
		}
		details.DriverID = &driverID

		if _, err := s.repos.driver.ChangeStatus(ctx, driverID, types.StatusDriverEnRoute); err != nil {
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

func (s *Service) processDriverLocation(ctx context.Context, current models.RideLocationUpdate, destination models.Location) error {
	if current.RideID == nil {
		c, ok := ctx.Value(wrap.LogCtxKey).(wrap.LogCtx)

		// Логируем, что RideID отсутствует
		s.l.Warn(ctx, "ride_id is nil in current, trying to extract from context")

		if !ok {
			return errors.New("failed to extract LogCtx from context")
		}

		if c.RideID == uuid.NilUUID.String() {
			return errors.New("ride_id is not set at message")
		}

		convID, _ := uuid.Parse(c.RideID)
		current.RideID = &convID
	}

	if _, err := s.UpdateLocation(ctx, current); err != nil {
		return err
	}

	status, err := s.repos.ride.Status(ctx, *current.RideID)
	if err != nil {
		return fmt.Errorf("failed to get ride status: %w", err)
	}
	if *status == types.StatusCancelled {
		s.l.Error(ctx, "ride has been cancelled", types.ErrRideCancelled)
		return nil
	}

	if !s.logic.calculate.IsDriverArrived(current.Location.Latitude, current.Location.Longitude, destination.Latitude, destination.Longitude) {
		return nil
	}

	if err := s.infra.trm.Do(ctx, func(ctx context.Context) error {
		if _, err := s.repos.driver.ChangeStatus(ctx, current.DriverID, types.StatusDriverArrived); err != nil {
			return fmt.Errorf("failed to change driver status: %w", err)
		}

		if err := s.infra.publisher.PublishDriverStatus(
			ctx,
			models.DriverStatusUpdateMessage{
				DriverID:  current.DriverID,
				Status:    types.StatusDriverArrived.String(),
				Timestamp: time.Now(),
				RideID:    current.RideID,
			}); err != nil {
			return fmt.Errorf("failed to publish driver status: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
