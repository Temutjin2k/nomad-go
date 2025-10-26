package wshandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	ws "github.com/Temutjin2k/ride-hail-system/pkg/wsHub"
)

type RideWsHandler struct {
	connections *ws.ConnectionHub
}

func NewRideWsHandler(connections *ws.ConnectionHub) *RideWsHandler {
	return &RideWsHandler{
		connections: connections,
	}
}

// TODO: imporove написал на скорую руку
// SendDriverLocation
func (h *RideWsHandler) SendDriverLocation(ctx context.Context, passengerID uuid.UUID, location *models.DriverLocationUpdate) error {
	ctx = wrap.WithAction(ctx, "ws_send_driver_location")
	conn, err := h.connections.GetConn(passengerID)
	if err != nil {
		return wrap.Error(ctx, err)
	}

	if err := conn.Send(location); err != nil {
		return wrap.Error(ctx, fmt.Errorf("failed to send driver location to passenger:%w", err))
	}

	return nil
}

func (h *RideWsHandler) SendRideAcceptedMessage(ctx context.Context, passengerID uuid.UUID, status string) error {
	return errors.ErrUnsupported
}

func (h *RideWsHandler) SendRideStatusUpdate(ctx context.Context, passengerID uuid.UUID, status string) error {
	return errors.ErrUnsupported
}
