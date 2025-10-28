package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
}

type Coordinates struct {
	Location       Location `json:"location"`
	AccuracyMeters float64  `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64  `json:"speed_kmh,omitempty"`
	HeadingDegrees float64  `json:"heading_degrees,omitempty"`
}

// RabbitMQ message: For Location Update → <location_fanout> exchange
type RideLocationUpdate struct {
	DriverID  uuid.UUID  `json:"driver_id"`
	RideID    *uuid.UUID `json:"ride_id,omitempty"`
	TimeStamp time.Time  `json:"timestamp"`

	Coordinates
}
