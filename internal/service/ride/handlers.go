package ride

import (
	"context"

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

		if err := s.repo.UpdateStatus(ctx, msg.RideID, types.StatusMatched); err != nil {
			return wrap.Error(ctx, err)
		}
		passengerID = ride.PassengerID
		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	s.passengerSender.SendRideAcceptedMessage(ctx, passengerID, "")

	return nil
}

// handleNotAccepted processes the scenario when a driver does not accept the ride.
func (s *RideService) handleNotAccepted(ctx context.Context, msg models.DriverMatchResponse) error {
	// TODO maybe send to passanger
	return nil
}

func (s *RideService) HandleDriverLocationUpdate(ctx context.Context, msg models.DriverLocationUpdate) error {
	return nil
}
