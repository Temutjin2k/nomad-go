package models

type Passenger struct {
	Name           string   `json:"passenger_name"`
	Phone          string   `json:"passenger_phone"`
	PickupLocation Location `json:"pickup_location"`
}
