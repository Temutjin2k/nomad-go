package models

type SessionSummary struct {
	SessionID      string
	DurationHours  float64
	RidesCompleted int
	Earnings       float64
}
