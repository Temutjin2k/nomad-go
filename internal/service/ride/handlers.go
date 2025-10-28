package ride

import (
	"context"
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

		if ride.Status != types.StatusRequested {
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

		ride.Status = types.StatusMatched
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
		return types.ErrRideNotFound
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
		case types.StatusMatched, types.StatusEnRoute, types.StatusArrived, types.StatusInProgress:
			// Поездка активна, продолжаем и отправляем обновление
		default:
			// Поездка завершена, отменена или еще не началась. Игнорируем.
			s.logger.Info(ctx, "skipping location update for inactive ride", "status", ride.Status)
			return nil
		}

		//  Определяем, куда едет водитель (к пассажиру или к месту назначения)
		var targetLocation models.Location
		if ride.Status == types.StatusInProgress {
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
