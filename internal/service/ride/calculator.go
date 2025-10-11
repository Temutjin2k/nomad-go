package ride

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

const (
	averageSpeedKmh = 50 // средняя скорость в пути
	earthRadiusKm = 6371	
)

// вычисление расстояние между двумя координатами, используя формулу гаверсинусов
func calculateDistance(p1, p2 models.Location) float64 {
	// градусы в радианы
	lat1Rad := p1.Latitude * math.Pi / 180
	lon1Rad := p1.Longitude * math.Pi / 180
	lat2Rad := p2.Latitude * math.Pi / 180
	lon2Rad := p2.Longitude * math.Pi / 180

	// разница долгот и широт
	diffLat := lat2Rad - lat1Rad
	diffLon := lon2Rad - lon1Rad

	// формула гаверсинусов
	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(diffLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusKm * c
	return distance
}

// примерное время в минутах (целое число).
func calculateDuration(distanceKm float64) int {
	if distanceKm <= 0 {
		return 0
	}
	// Время (в часах) = Расстояние / Скорость
	// Время (в минутах) = (Расстояние / Скорость) * 60
	durationMinutes := (distanceKm / averageSpeedKmh) * 60
	return int(math.Ceil(durationMinutes))
}

// рассчет предварительную стоимость поездки на основе тарифов
func calculateFare(rideType string, distanceKm float64, durationMin int) float64 {
	var baseFare, ratePerKm, ratePerMin float64

	switch rideType {
	case "PREMIUM":
		baseFare = 800
		ratePerKm = 120
		ratePerMin = 60
	case "XL":
		baseFare = 1000
		ratePerKm = 150
		ratePerMin = 75
	case "ECONOMY":
		fallthrough // Если тип не указан или ECONOMY, используем его по умолчанию
	default:
		baseFare = 500
		ratePerKm = 100
		ratePerMin = 50
	}

	// Формула расчета: Базовая ставка + (стоимость за км) + (стоимость за минуты)
	fare := baseFare + (distanceKm * ratePerKm) + (float64(durationMin) * ratePerMin)
	return fare
}

// создать уникальный номер поездки
func (s *RideService) generateRideNumber(ctx context.Context) (string, error) {
	datePart := time.Now().Format("20060102")
	
	count, err := s.repo.CountByDate(ctx, time.Now())
	if err != nil {
		return "", wrap.Error(ctx, err)
	}
	
	nextSequence := count + 1
	return fmt.Sprintf("RIDE_%s_%03d", datePart, nextSequence), nil
}