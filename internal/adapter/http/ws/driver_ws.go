package wshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/ws/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
	ws "github.com/Temutjin2k/ride-hail-system/pkg/wsHub"
)

type DriverHub struct {
	connections *ws.ConnectionHub
}

func NewDriverHub(connHub *ws.ConnectionHub) *DriverHub {
	return &DriverHub{
		connections: connHub,
	}
}

func (h *DriverHub) GetRideOffer(ctx context.Context, driverID uuid.UUID, offer models.RideOffer) (bool, error) {
	const op = "DriverHub.SendRideOffer"
	offer.MsgType = "ride_offer"

	conn, err := h.connections.GetConn(driverID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	ch := make(chan map[string]any, 1)
	conn.Subscribe(offer.ID.String(), ch)
	defer conn.Unsubscribe(offer.ID.String())

	if err := conn.Send(offer); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	// Timeout: 30 seconds for driver responses
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	var resp dto.OfferResp
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("%s: %s", op, "ctx (Done)")
	case <-timer.C:
		return false, fmt.Errorf("%s: %w", op, types.ErrListenTimeout)
	case data := <-ch:
		b, err := json.Marshal(data)
		if err != nil {
			errorResponse(conn, err.Error())
			return false, fmt.Errorf("%s: marshal response: %w", op, err)
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			errorResponse(conn, err.Error())
			return false, fmt.Errorf("%s: unmarshal response: %w", op, err)
		}

		v := validator.New()
		resp.Validate(v)
		if !v.Valid() {
			if err := failedValidationResponse(conn, v.Errors); err != nil {
				return false, fmt.Errorf("failed send validation response: %w", err)
			}
		}
	}

	return resp.Accepted, nil
}

func (h *DriverHub) SendRideDetails(ctx context.Context, details models.RideDetails) error {
	const op = "DriverHub.SendRideDetails"
	details.MsgType = "ride_details"

	conn, err := h.connections.GetConn(*details.DriverID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := conn.Send(details); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (h *DriverHub) ListenLocationUpdates(ctx context.Context, driverID, rideID uuid.UUID, handler func(ctx context.Context, location models.RideLocationUpdate) error) error {
	const op = "DriverHub.ListenLocationUpdates"

	conn, err := h.connections.GetConn(driverID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	randID := uuid.New().String()

	// Open message receiver
	ch := make(chan map[string]any, 1)
	conn.Subscribe(randID, ch)
	defer conn.Unsubscribe(randID)

	// Timeout: 3 days
	timer := time.NewTimer(72 * time.Hour)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %s", op, "ctx (Done)")
		case <-timer.C:
			return fmt.Errorf("%s: %w", op, types.ErrListenTimeout)
		case data := <-ch:
			now := time.Now()
			var req dto.DriverLocationUpdate
			b, err := json.Marshal(data)
			if err != nil {
				errorResponse(conn, err.Error())
				continue
			}
			if err := json.Unmarshal(b, &req); err != nil {
				errorResponse(conn, err.Error())
				continue
			}

			v := validator.New()
			req.Validate(v)
			if !v.Valid() {
				failedValidationResponse(conn, v.Errors)
				continue
			}

			if err := handler(ctx, models.RideLocationUpdate{
				DriverID:  driverID,
				RideID:    &rideID,
				TimeStamp: now,
				Coordinates: models.Coordinates{
					AccuracyMeters: req.AccuracyMeters,
					SpeedKmh:       req.SpeedKmH,
					HeadingDegrees: req.HeadingDegrees,
					Location: models.Location{
						Latitude:  *req.Latitude,
						Longitude: *req.Longitude,
					},
				},
			}); err != nil {
				if err == types.ErrRideCancelled {
					return nil
				}
				errorResponse(conn, err.Error())
				continue
			}
		}
	}
}
