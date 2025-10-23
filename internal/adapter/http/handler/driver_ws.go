package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
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

func (h *DriverHub) SendRideOffer(ctx context.Context, driverID uuid.UUID, offer models.RideOffer) (bool, error) {
	const op = "DriverHub.SendRideOffer"

	// Преобразуем структуру в map
	var msg map[string]any
	data, err := json.Marshal(offer)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	// Отправляем через WebSocket
	if err := h.connections.SendTo(driverID, msg); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	conn, err := h.connections.GetConn(driverID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	ch := make(chan map[string]any, 1)
	conn.Subscribe(offer.ID.String(), ch)
	defer conn.Unsubscribe(offer.ID.String())

	// Timeout: 30 seconds for driver responses
	timer := time.NewTicker(time.Second * 30)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false, fmt.Errorf("%s: %s", op, "ctx (Done)")
	case <-timer.C:
		return false, fmt.Errorf("%s: %w", op, ws.ErrListenTimeout)
	case <-ch:

	}

	return true, nil
}
