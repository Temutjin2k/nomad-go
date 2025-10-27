package types

type RideEvent string

func (s RideEvent) String() string {
	return string(s)
}

const (
	EventRideRequested   RideEvent = "RIDE_REQUESTED"
	EventDriverMatched   RideEvent = "DRIVER_MATCHED"
	EventDriverArrived   RideEvent = "DRIVER_ARRIVED"
	EventRideStarted     RideEvent = "RIDE_STARTED"
	EventRideCompleted   RideEvent = "RIDE_COMPLETED"
	EventRideCancelled   RideEvent = "RIDE_CANCELLED"
	EventStatusChanged   RideEvent = "STATUS_CHANGED"
	EventLocationUpdated RideEvent = "LOCATION_UPDATED"
	EventFareAdjusted    RideEvent = "FARE_ADJUSTED"
)
