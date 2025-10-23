package models

import "time"

type ActiveRidesResponse struct {
	Rides    []RideInfo `json:"rides"`
	Metadata Metadata   `json:"metadata"`
}

type OverviewResponse struct {
	Timestamp          time.Time      `json:"timestamp"`
	Metrics            Metrics        `json:"metrics"`
	DriverDistribution map[string]int `json:"driver_distribution"`
	Hotspots           []Hotspot      `json:"hotspots"`
}

type Metrics struct {
	ActiveRides                int     `json:"active_rides"`
	AvailableDrivers           int     `json:"available_drivers"`
	BusyDrivers                int     `json:"busy_drivers"`
	TotalRidesToday            int     `json:"total_rides_today"`
	TotalRevenueToday          float64 `json:"total_revenue_today"`
	AverageWaitTimeMinutes     float64 `json:"average_wait_time_minutes"`
	AverageRideDurationMinutes float64 `json:"average_ride_duration_minutes"`
	CancellationRate           float64 `json:"cancellation_rate"`
}

type Hotspot struct {
	Location       string `json:"location"`
	ActiveRides    int    `json:"active_rides"`
	WaitingDrivers int    `json:"waiting_drivers"`
}
