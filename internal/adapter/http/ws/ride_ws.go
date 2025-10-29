package wshandler

import (
	"context"

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

// SendToPassenger
func (h *RideWsHandler) SendToPassenger(ctx context.Context, passengerID uuid.UUID, data any) error {
	conn, err := h.connections.GetConn(passengerID)
	if err != nil {
		return err
	}

	if err := conn.Send(data); err != nil {
		return err
	}

	return nil
}

// // TODO: imporove написал на скорую руку
// // SendDriverLocation
// func (h *RideWsHandler) SendDriverLocation(ctx context.Context, passengerID uuid.UUID, location *models.DriverLocationUpdate) error {
// 	ctx = wrap.WithAction(ctx, "ws_send_driver_location")
// 	conn, err := h.connections.GetConn(passengerID)
// 	if err != nil {
// 		return wrap.Error(ctx, err)
// 	}

// 	if err := conn.Send(location); err != nil {
// 		return wrap.Error(ctx, fmt.Errorf("failed to send driver location to passenger:%w", err))
// 	}

// 	return nil
// }

// func (h *RideWsHandler) SendRideAcceptedMessage(ctx context.Context, passengerID uuid.UUID, data models.DriverMatchResponse) error {
// 	ctx = wrap.WithAction(ctx, "ws_send_ride_accepted_message")
// 	conn, err := h.connections.GetConn(passengerID)
// 	if err != nil {
// 		return wrap.Error(ctx, err)
// 	}

// 	if err := conn.Send(data); err != nil {
// 		return wrap.Error(ctx, fmt.Errorf("failed to send ride accepted message to passenger:%w", err))
// 	}

// 	return nil
// }

// func (h *RideWsHandler) SendRideStatusUpdate(ctx context.Context, passengerID uuid.UUID, data any) error {
// 	ctx = wrap.WithAction(ctx, "ws_send_ride_status_update")
// 	conn, err := h.connections.GetConn(passengerID)
// 	if err != nil {
// 		return wrap.Error(ctx, err)
// 	}

// 	// TODO: define data type

// 	if err := conn.Send(msg); err != nil {
// 		return wrap.Error(ctx, fmt.Errorf("failed to send ride status update to passenger:%w", err))
// 	}
// }
