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
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

// HandleDriverResponse processes driver match responses.
func (s *RideService) HandleDriverResponse(ctx context.Context, msg models.DriverMatchResponse) error {
	ctx = wrap.WithAction(wrap.WithRequestID(wrap.WithRideID(ctx, msg.RideID.String()), msg.CorrelationID), "handle_driver_response")

	// if not accepted
	if !msg.Accepted {
		s.logger.Info(ctx, "driver did not accepted the ride", "driver_id", msg.DriverID)
		return s.handleNotAccepted(ctx, msg)
	}

	var passengerID uuid.UUID
	// if accepted
	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		ride, err := s.repo.Get(ctx, msg.RideID)
		if err != nil {
			return err
		}

		if ride == nil {
			return types.ErrRideNotFound
		}

		if ride.Status != types.StatusRequested.String() {
			s.logger.Warn(ctx, "status already changed, skipping", "current_status", ride.Status)
			return types.ErrInvalidRideStatus
		}

		// if err := s.repo.Update(ctx, ride); err != nil {
		// 	return err
		// }w

		if err := s.repo.UpdateStatus(ctx, msg.RideID, types.StatusMatched); err != nil {
			return err
		}

		if err := s.repo.UpdateMatchedAt(ctx, msg.RideID); err != nil {
			return err
		}

		ride.Status = types.StatusMatched.String()
		ride.DriverID = &msg.DriverID

		message := models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        ride.Status,
			Timestamp:     ride.CreatedAt,
			DriverID:      ride.DriverID,
			CorrelationID: ride.RideNumber,
		}

		if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
			return err
		}

		passengerID = ride.PassengerID

		if err := s.passengerSender.SendToPassenger(ctx, passengerID, msg); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
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
	ctx = wrap.WithAction(wrap.WithRideID(ctx, msg.RideID.String()), "handle_driver_location_update")

	if msg.RideID == nil {
		s.logger.Warn(ctx, "ride_id not provided")
		return nil
	}

	var (
		PassengerID uuid.UUID
		wsMessage   models.PassengerLocationUpdateDTO
	)

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		ride, err := s.repo.Get(ctx, *msg.RideID)
		if err != nil {
			return err
		}
		if ride == nil {
			return types.ErrNotFound
		}

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
		wsMessage = models.PassengerLocationUpdateDTO{
			Type:   types.EventLocationUpdated.String(),
			RideID: ride.ID,
			DriverLocation: models.Location{
				Latitude:  msg.Location.Latitude,
				Longitude: msg.Location.Longitude,
			},
			DistanceToPickupKm: distanceKm,
			EstimatedArrival:   time.Now().Add(time.Duration(durationMin) * time.Minute),
		}
		PassengerID = ride.PassengerID
		return nil
	}); err != nil {
		return err
	}

	if err := s.passengerSender.SendToPassenger(ctx, PassengerID, wsMessage); err != nil {
		// Это не фатальная ошибка, мы не должны NACK'ать сообщение в RabbitMQ.
		// Пассажир просто пропустит одно обновление координат.
		s.logger.Warn(ctx, "failed to send websocket location update to passenger", "error", err, "passenger_id", PassengerID)
	}

	return nil
}

func (s *RideService) HandleDriverStatusUpdate(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_driver_status_update")

	if msg.RideID == nil {
		return wrap.Error(ctx, errors.New("ride_id not provided"))
	}

	ride, err := s.repo.Get(ctx, *msg.RideID)
	if err != nil {
		return wrap.Error(ctx, err)
	}

	if ride.DriverID == nil || *ride.DriverID != msg.DriverID {
		s.logger.Warn(ctx, "driver_id does not match the ride's driver", "ride_id", ride.ID, "ride_driver_id", ride.DriverID)
		return wrap.Error(ctx, errors.New("driver_id does not match the ride's driver"))
	}

	ctx = wrap.WithRideID(ctx, ride.ID.String())

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
		return wrap.Error(ctx, errors.New("invalid status"))
	}
}

func (s *RideService) handleDriverEnRoute(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_driver_en_route")

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		if ride.Status != types.StatusMatched.String() {
			s.logger.Warn(ctx, "status already changed, skipping", "current_status", ride.Status)
			return nil
		}

		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusEnRoute); err != nil {
			return fmt.Errorf("failed to update status to EN_ROUTE: %w", err)
		}

		// По идее не нужно отправлять драйверу сообщение о том что поездка EnRoute,
		// по сути не нужно даже ничего отправлять
		message := models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusEnRoute.String(),
			Timestamp:     time.Now(),
			DriverID:      ride.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		}

		if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
			return err
		}

		// отправляем пассажиру сообщение по вебсокету
		wsMessage := models.StatusUpdateWebSocketMessage{
			EventType: types.EventStatusChanged,
			Data:      message,
		}
		if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
			return fmt.Errorf("failed to notify passnager: %w", err)
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	eventData := struct {
		Data models.DriverStatusUpdateMessage `json:"driver_status_update_message"`
	}{
		Data: msg,
	}

	bytes, err := json.Marshal(eventData)
	if err != nil {
		s.logger.Warn(ctx, "failed to marshal event data", "error", err.Error())
		// не фатальная ошибка, продолжаем
	}

	// CreateEvent записывает событие, связанное с поездкой в таблицу ride_events
	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventStatusChanged, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "error", err.Error())
		// не фатальная ошибка, продолжаем
	}

	return nil
}
func (s *RideService) handleDriverArrived(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_driver_arrived")

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		if ride.Status != types.StatusEnRoute.String() {
			s.logger.Warn(ctx, "status already changed, skipping", "current_status", ride.Status)
			return nil
		}

		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusArrived); err != nil {
			return err
		}

		if err := s.repo.UpdateArrivedAt(ctx, ride.ID); err != nil {
			return err
		}

		message := models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusArrived.String(),
			Timestamp:     time.Now(),
			DriverID:      ride.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		}
		if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
			return err
		}

		// отправляем пассажиру сообщение по вебсокету
		wsMessage := models.StatusUpdateWebSocketMessage{
			EventType: types.EventDriverArrived,
			Data:      message,
		}
		if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
			return fmt.Errorf("failed to notify passnager: %w", err)
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	eventData := struct {
		Data models.DriverStatusUpdateMessage `json:"driver_status_update_message"`
	}{
		Data: msg,
	}

	bytes, err := json.Marshal(eventData)
	if err != nil {
		s.logger.Warn(ctx, "failed to marshal event data", "error", err.Error())
		// non-fatal
	}

	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventDriverArrived, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "error", err.Error())
		// non-fatal
	}

	return nil
}

func (s *RideService) handleRideInProgress(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_ride_in_progress")

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		if ride.Status != types.StatusArrived.String() {
			s.logger.Warn(ctx, "status already changed, skipping", "current_status", ride.Status)
			return nil
		}

		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusInProgress); err != nil {
			return err
		}

		if err := s.repo.UpdateStartedAt(ctx, ride.ID); err != nil {
			return err
		}

		message := models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusInProgress.String(),
			Timestamp:     time.Now(),
			DriverID:      ride.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		}

		if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
			return err
		}

		// отправляем пассажиру сообщение по вебсокету
		wsMessage := models.StatusUpdateWebSocketMessage{
			EventType: types.EventRideStarted,
			Data:      message,
		}
		if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
			return fmt.Errorf("failed to notify passnager: %w", err)
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	eventData := struct {
		Data models.DriverStatusUpdateMessage `json:"driver_status_update_message"`
	}{
		Data: msg,
	}

	bytes, err := json.Marshal(eventData)
	if err != nil {
		s.logger.Warn(ctx, "failed to marshal event data", "error", err.Error())
	}

	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventRideStarted, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "error", err.Error())
	}

	return nil
}

func (s *RideService) handleRideCompleted(ctx context.Context, ride *models.Ride, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "handle_ride_completed")

	if err := s.trm.Do(ctx, func(ctx context.Context) error {
		if ride.Status != types.StatusInProgress.String() {
			s.logger.Warn(ctx, "status already changed, skipping", "current_status", ride.Status)
			return nil
		}

		if err := s.repo.UpdateStatus(ctx, ride.ID, types.StatusCompleted); err != nil {
			return err
		}

		if err := s.repo.UpdateCompletedAt(ctx, ride.ID); err != nil {
			return err
		}

		message := models.RideStatusUpdateMessage{
			RideID:        ride.ID,
			Status:        types.StatusCompleted.String(),
			Timestamp:     time.Now(),
			DriverID:      ride.DriverID,
			CorrelationID: wrap.GetRequestID(ctx),
		}

		if err := s.publisher.PublishRideStatus(ctx, message); err != nil {
			return err
		}

		// отправляем пассажиру сообщение по вебсокету
		wsMessage := models.StatusUpdateWebSocketMessage{
			EventType: types.EventRideCompleted,
			Data:      message,
		}
		if err := s.passengerSender.SendToPassenger(ctx, ride.PassengerID, wsMessage); err != nil {
			return fmt.Errorf("failed to notify passnager: %w", err)
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	eventData := struct {
		Data models.DriverStatusUpdateMessage `json:"driver_status_update_message"`
	}{
		Data: msg,
	}

	bytes, err := json.Marshal(eventData)
	if err != nil {
		s.logger.Warn(ctx, "failed to marshal event data", "error", err.Error())
	}

	if err := s.eventRepo.CreateEvent(ctx, ride.ID, types.EventRideCompleted, bytes); err != nil {
		s.logger.Warn(ctx, "failed to create ride event", "error", err.Error())
	}

	return nil
}
