package drivergo

import (
	"math"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
)

const (
	EarthRadiusKm       = 6371.0
	DefaultSpeedEconomy = 30.0
	DefaultSpeedXL      = 35.0
	DefaultSpeedPremium = 40.0
)

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// HaversineDistance calculates the Haversine distance between two geographic points.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := degreesToRadians(lat1)
	lon1Rad := degreesToRadians(lon1)
	lat2Rad := degreesToRadians(lat2)
	lon2Rad := degreesToRadians(lon2)

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Pow(math.Sin(deltaLat/2), 2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Pow(math.Sin(deltaLon/2), 2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

// getEstimatedArrival calculates the estimated arrival time based on distance and average speed.
func (s *Service) getEstimatedArrival(startLat, startLon, destLat, destLon float64, vehicleClass types.VehicleClass) time.Time {
	var avgSpeedKmh float64

	switch vehicleClass {
	case types.ClassXL:
		avgSpeedKmh = DefaultSpeedXL
	case types.ClassPremium:
		avgSpeedKmh = DefaultSpeedPremium
	default:
		avgSpeedKmh = DefaultSpeedEconomy
	}

	// Distance in kilometers
	distanceKm := HaversineDistance(startLat, startLon, destLat, destLon)

	// Time in hours
	timeHours := distanceKm / avgSpeedKmh

	// Convert hours to duration
	timeDuration := time.Duration(timeHours * float64(time.Hour))

	return time.Now().Add(timeDuration)
}
