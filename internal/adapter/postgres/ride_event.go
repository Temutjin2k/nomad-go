package postgres

import (
	"context"
	"encoding/json"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RideEvent struct {
	db *pgxpool.Pool
}

func NewRideEvent(db *pgxpool.Pool) *RideEvent {
	return &RideEvent{db: db}
}

// CreateEvent inserts a new ride event into the database.
func (r *RideEvent) CreateEvent(ctx context.Context, rideID uuid.UUID, eventType types.RideEvent, eventData json.RawMessage) error {
	q := TxorDB(ctx, r.db)

	query := `INSERT INTO ride_events (ride_id, event_type, event_data)
			  VALUES ($1, $2, $3);`

	_, err := q.Exec(ctx, query, rideID, eventType.String(), eventData)
	if err != nil {
		return err
	}
	return nil
}
