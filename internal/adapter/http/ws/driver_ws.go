package wshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/ws/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
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

	conn, err := h.connections.GetConn(driverID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	ch := make(chan map[string]any, 1)
	conn.Subscribe(offer.ID.String(), ch)
	defer conn.Unsubscribe(offer.ID.String())

	if err := conn.Send(msg); err != nil {
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
		return false, fmt.Errorf("%s: %w", op, ws.ErrListenTimeout)
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
