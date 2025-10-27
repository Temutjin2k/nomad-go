package dto

type LocationUpdate struct {
	Type           string  `json:"type"`            // тип сообщения, например "location_update"
	Latitude       float64 `json:"latitude"`        // широта
	Longitude      float64 `json:"longitude"`       // долгота
	AccuracyMeters float64 `json:"accuracy_meters"` // точность GPS
	SpeedKmh       float64 `json:"speed_kmh"`       // скорость, км/ч
	HeadingDegrees float64 `json:"heading_degrees"` // направление движения
}
