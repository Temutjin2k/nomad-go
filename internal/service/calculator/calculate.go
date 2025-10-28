package ridecalc

import (
	"math"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
)

const (
	averageSpeedKmh = 50   // средняя скорость
	earthRadiusKm   = 6371 // радиус Земли в км
	earthRadiusM    = 6371000.0
	arrivalRadius   = 25.0 // радиус прибытия в метрах
)

type Calculator interface {
	Distance(p1, p2 models.Location) float64
	Duration(distanceKm float64) int
	Fare(rideType string, distanceKm float64, durationMin int) float64
	Priority(ride *models.Ride) int
	EstimatedArrival(startLat, startLon, destLat, destLon float64, vehicleClass types.VehicleClass) time.Time
	IsDriverArrived(driverLat, driverLng, targetLat, targetLng float64) bool
}

type CalculatorImpl struct{}

func New() *CalculatorImpl {
	return &CalculatorImpl{}
}

// Проверяет, находится ли водитель в радиусе arrivalRadius от цели
func (c *CalculatorImpl) IsDriverArrived(driverLat, driverLng, targetLat, targetLng float64) bool {
	dist := c.distanceMeters(driverLat, driverLng, targetLat, targetLng)
	return dist <= arrivalRadius
}

func (c *CalculatorImpl) distanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	calc := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * calc
}

// вычисление расстояние между двумя координатами, используя формулу гаверсинусов в километрах
func (c *CalculatorImpl) Distance(p1, p2 models.Location) float64 {
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
	angle := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusKm * angle
	return distance
}

// примерное время в минутах (целое число).
func (c *CalculatorImpl) Duration(distanceKm float64) int {
	if distanceKm <= 0 {
		return 0
	}
	// Время (в часах) = Расстояние / Скорость
	// Время (в минутах) = (Расстояние / Скорость) * 60
	durationMinutes := (distanceKm / averageSpeedKmh) * 60
	return int(math.Ceil(durationMinutes))
}

// рассчет предварительную стоимость поездки на основе тарифов
func (c *CalculatorImpl) Fare(rideType string, distanceKm float64, durationMin int) float64 {
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

func (c *CalculatorImpl) Priority(ride *models.Ride) int {
	priority := 1

	// Правило №1: Час пик
	// Увеличиваем приоритет утром (7-10) и вечером (17-20).
	currentHour := time.Now().Hour()
	if (currentHour >= 7 && currentHour < 10) || (currentHour >= 17 && currentHour < 20) {
		priority += 3
	}

	// Правило №2: Тип поездки
	// Более дорогие поездки получают небольшой бонус.
	if ride.RideType == "PREMIUM" || ride.RideType == "XL" {
		priority += 2
	}

	if minutes := c.Duration(c.Distance(ride.Pickup, ride.Destination)); minutes < 15 {
		priority += 3
	}
	// Правило №3 (на будущее): Статус пассажира
	// Если бы у нас была система ролей с подписками по типу VIP, VIP++, SSS ранг, то давали бы доп приоритет

	if priority > 10 {
		priority = 10
	}

	return priority
}

// getEstimatedArrival calculates the estimated arrival time based on distance and average speed.
func (c *CalculatorImpl) EstimatedArrival(startLat, startLon, destLat, destLon float64, vehicleClass types.VehicleClass) time.Time {
	distanceKm := c.Distance(
		models.Location{
			Latitude:  startLat,
			Longitude: startLon,
		},
		models.Location{
			Latitude:  destLat,
			Longitude: destLon,
		},
	)

	// Time in minutes
	timeMin := c.Duration(distanceKm)

	// Convert minutes to duration
	timeDuration := time.Duration(timeMin) * time.Minute

	return time.Now().Add(timeDuration)
}
