package ride

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

// HandleDriverResponse processes driver match responses.
func (s *RideService) HandleDriverResponse(ctx context.Context, msg models.DriverMatchResponse) error {
	ctx = wrap.WithAction(wrap.WithRequestID(wrap.WithRideID(ctx, msg.RideID.String()), msg.CorrelationID), "handle_driver_response")

	s.logger.Debug(ctx, "HandleDriverResponse")

	// if not accepted
	if !msg.Accepted {
		s.logger.Info(ctx, "driver did not accepted the ride", "driver_id", msg.DriverID)
		return s.handleNotAccepted(ctx, msg)
	}

	ride, err := s.repo.Get(ctx, msg.RideID)
	if err != nil {
		return wrap.Error(ctx, fmt.Errorf("%w: failed to get ride: %w", types.ErrDatabaseFailed, err))
	}

	if ride == nil {
		return wrap.Error(ctx, types.ErrRideNotFound)
	}

	if ride.Status != types.StatusRequested.String() {
		s.logger.Warn(ctx, "status already changed", "current_status", ride.Status, "expected_status", types.StatusRequested)
		return wrap.Error(ctx, types.ErrInvalidRideStatus)
	}

	// Изменяем статус поездки на matched, добавляем driver_id
	if err := s.repo.DriverMatchedForRide(ctx, ride.ID, msg.DriverID, ride.EstimatedFare); err != nil {
		return wrap.Error(ctx, fmt.Errorf("%w: failed to update ride status: %w", types.ErrDatabaseFailed, err))
	}

	message := models.RideStatusUpdateMessage{
		RideID:        ride.ID,
		Status:        types.StatusMatched.String(),
		Timestamp:     time.Now(),
		DriverID:      &msg.DriverID,
		CorrelationID: wrap.GetRequestID(ctx),
	}

	if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
		return wrap.Error(ctx, fmt.Errorf("%w: %w", types.ErrFailedToPublishRideStatus, err))
	}

	data := models.StatusUpdateWebSocketMessage{
		EventType: types.EventDriverMatched,
		Data:      msg,
	}

	// Уведомляем пассажира по вебсокету
	if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, data); err != nil {
		s.logger.Warn(ctx, "failed to notify passenger about driver matching", "event_type", types.EventDriverMatched, "error", err.Error())
	}

	// записываем ивент
	eventData, _ := json.Marshal(msg) // non fatal event so just ignore error
	if err := s.eventRepo.CreateEvent(ctx, msg.RideID, types.EventDriverMatched, eventData); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "event_type", types.EventDriverMatched, "error", err.Error())
	}

	return nil
}

// handleNotAccepted processes the scenario when a driver does not accept the ride.
func (s *RideService) handleNotAccepted(ctx context.Context, msg models.DriverMatchResponse) error {
	ctx = wrap.WithAction(wrap.WithRequestID(wrap.WithRideID(ctx, msg.RideID.String()), msg.CorrelationID), "handle_not_accepted")
	s.logger.Info(ctx, "handling not accepted driver response", "driver_id", msg.DriverID)
	return nil
}

func (s *RideService) HandleDriverLocationUpdate(ctx context.Context, msg models.RideLocationUpdate) error {
	ctx = wrap.WithAction(wrap.WithDriverID(ctx, msg.DriverID.String()), "handle_driver_location_update")

	s.logger.Debug(ctx, "HandleDriverLocationUpdate")

	if msg.RideID == nil {
		s.logger.Warn(ctx, "ride_id not provided")
		return nil
	}

	// Get ride
	ride, err := s.repo.Get(ctx, *msg.RideID)
	if err != nil {
		return err
	}
	if ride == nil {
		return types.ErrNotFound
	}
	ctx = wrap.WithPassengerID(ctx, ride.PassengerID.String())

	switch ride.Status {
	case types.StatusMatched.String(), types.StatusEnRoute.String(), types.StatusArrived.String(), types.StatusInProgress.String():
		// Поездка активна, продолжаем и отправляем обновление
	default:
		// Поездка завершена, отменена или еще не началась. Игнорируем.
		s.logger.Info(ctx, "skipping location update for inactive ride", "status", ride.Status)
		return nil
	}

	//  Определяем, куда едет водитель (к пассажиру или к месту назначения)
	var targetLocation models.Location
	if ride.Status == types.StatusInProgress.String() {
		targetLocation = ride.Destination
	} else {
		targetLocation = ride.Pickup
	}

	// (Опционально) Рассчитываем оставшееся расстояние и время
	driverCurrentLocation := models.Location{
		Latitude:  msg.Location.Latitude,
		Longitude: msg.Location.Longitude,
	}

	distanceKm := s.calculate.Distance(driverCurrentLocation, targetLocation)
	durationMin := s.calculate.Duration(distanceKm)

	// 5. Формируем сообщение для WebSocket
	wsMessage := models.PassengerLocationUpdateDTO{
		Type:   types.EventLocationUpdated.String(),
		RideID: ride.ID,
		DriverLocation: models.Location{
			Latitude:  msg.Location.Latitude,
			Longitude: msg.Location.Longitude,
		},
		DistanceToPickupKm: distanceKm,
		EstimatedArrival:   time.Now().Add(time.Duration(durationMin) * time.Minute),
	}

	// записываем ивент
	eventData, _ := json.Marshal(wsMessage) // non fatal event so just ignore error
	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventLocationUpdated, eventData); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "event_type", types.EventLocationUpdated, "error", err.Error())
	}

	if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
		s.logger.Warn(ctx, "failed to send a driver location update to passenger via websocket", "error", err)
	}

	return nil
}

// HandleDriverStatusUpdate обрабатывает сообщение от driver сервиса об изменений статуса водителя
func (s *RideService) HandleDriverStatusUpdate(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_driver_status_update")

	s.logger.Debug(ctx, "HandleDriverStatusUpdate")

	if msg.RideID == nil {
		return wrap.Error(ctx, errors.New("ride_id not provided"))
	}

	ride, err := s.repo.Get(ctx, *msg.RideID)
	if err != nil {
		return wrap.Error(ctx, fmt.Errorf("%w: failed to get ride: %w", types.ErrDatabaseFailed, err))
	}

	ctx = wrap.WithRideID(wrap.WithDriverID(wrap.WithPassengerID(ctx, ride.PassengerID.String()), msg.DriverID.String()), ride.ID.String())

	switch msg.Status {
	case types.StatusDriverEnRoute.String():
		return wrap.Error(ctx, s.handleDriverEnRoute(ctx, ride, msg))
	case types.StatusDriverArrived.String():
		return wrap.Error(ctx, s.handleDriverArrived(ctx, ride, msg))
	case types.StatusInProgress.String():
		return wrap.Error(ctx, s.handleRideInProgress(ctx, ride, msg))
	case types.StatusCompleted.String():
		return wrap.Error(ctx, s.handleRideCompleted(ctx, ride, msg))
	default:
		// TODO: что вернуть
		return wrap.Error(ctx, errors.New("invalid driver status"))
	}
}

func (s *RideService) handleDriverEnRoute(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_driver_en_route")

	s.logger.Debug(ctx, "handleDriverEnRoute")

	if ride.Status != types.StatusMatched.String() {
		s.logger.Warn(ctx, "invalid ride status", "current_status", ride.Status, "expectes_status", types.StatusMatched)
		return fmt.Errorf("invalid ride status expected: %s", types.StatusMatched)
	}

	if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusEnRoute); err != nil {
		return wrap.Error(ctx, fmt.Errorf("failed to update status to EN_ROUTE: %w", err))
	}

	s.logger.Info(ctx, "updated ride status to EN_ROUTE")

	// отправляем пассажиру сообщение по вебсокету
	wsMessage := models.StatusUpdateWebSocketMessage{
		EventType: types.EventStatusChanged,
		Data: models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusEnRoute.String(),
			Timestamp:     time.Now(),
			DriverID:      &msg.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		},
	}
	if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
		s.logger.Warn(ctx, "failed to notify passenger", "error", err)
	}

	bytes, _ := json.Marshal(msg) // non fatal event so just ignore error
	// CreateEvent записывает событие, связанное с поездкой в таблицу ride_events
	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventStatusChanged, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "event_type", types.EventStatusChanged, "error", err.Error())
		// не фатальная ошибка, продолжаем
	}

	return nil
}
func (s *RideService) handleDriverArrived(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_driver_arrived")

	s.logger.Debug(ctx, "handleDriverArrived")

	if ride.Status != types.StatusEnRoute.String() {
		s.logger.Warn(ctx, "invalid ride status", "current_status", ride.Status, "expectes_status", types.StatusEnRoute)
		return fmt.Errorf("invalid ride status expected: %s", types.StatusEnRoute)
	}

	if err := s.trm.Do(ctx, func(ctx context.Context) error {

		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusArrived); err != nil {
			return err
		}

		if err := s.repo.UpdateArrivedAt(ctx, ride.ID); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	s.logger.Info(ctx, "updated ride status to ARRIVED")

	// отправляем пассажиру сообщение по вебсокету
	wsMessage := models.StatusUpdateWebSocketMessage{
		EventType: types.EventDriverArrived,
		Data: models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusArrived.String(),
			Timestamp:     time.Now(),
			DriverID:      &msg.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		},
	}
	if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
		s.logger.Warn(ctx, "failed to notify passenger", "error", err)
	}

	bytes, _ := json.Marshal(msg) // non fatal event so just ignore error
	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventDriverArrived, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "event_type", types.EventDriverArrived, "error", err.Error())
		// non-fatal
	}

	return nil
}

func (s *RideService) handleRideInProgress(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_ride_in_progress")

	s.logger.Debug(ctx, "handleRideInProgress")

	if ride.Status != types.StatusArrived.String() {
		s.logger.Warn(ctx, "invalid ride status", "current_status", ride.Status, "expectes_status", types.StatusArrived)
		return fmt.Errorf("invalid ride status expected: %s", types.StatusArrived)
	}

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusInProgress); err != nil {
			return err
		}

		if err := s.repo.UpdateStartedAt(ctx, ride.ID); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	s.logger.Info(ctx, "updated ride status to IN_PROGRESS")

	// отправляем пассажиру сообщение по вебсокету
	wsMessage := models.StatusUpdateWebSocketMessage{
		EventType: types.EventRideStarted,
		Data: models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusInProgress.String(),
			Timestamp:     time.Now(),
			DriverID:      &msg.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		},
	}
	if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
		s.logger.Warn(ctx, "failed to notify passenger", "error", err)
	}

	bytes, _ := json.Marshal(msg) // non fatal event so just ignore error
	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventRideStarted, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "event_type", types.EventRideStarted, "error", err.Error())
	}

	return nil
}

func (s *RideService) handleRideCompleted(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_ride_completed")

	s.logger.Debug(ctx, "handleRideCompleted")

	if ride.Status != types.StatusInProgress.String() {
		s.logger.Warn(ctx, "invalid ride status", "current_status", ride.Status, "expectes_status", types.StatusInProgress)
		return fmt.Errorf("invalid ride status expected: %s", types.StatusInProgress)
	}

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusCompleted); err != nil {
			return err
		}

		if err := s.repo.UpdateCompletedAt(ctx, ride.ID); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	s.logger.Info(ctx, "updated ride status to COMPLETED")

	// отправляем пассажиру сообщение по вебсокету
	wsMessage := models.StatusUpdateWebSocketMessage{
		EventType: types.EventRideCompleted,
		Data: models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusCompleted.String(),
			Timestamp:     time.Now(),
			DriverID:      &msg.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		},
	}
	if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
		s.logger.Warn(ctx, "failed to notify passenger", "error", err)
	}

	bytes, _ := json.Marshal(msg) // non fatal event so just ignore error
	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventRideCompleted, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "event_type", types.EventRideCompleted, "error", err.Error())
	}

	return nil
}
