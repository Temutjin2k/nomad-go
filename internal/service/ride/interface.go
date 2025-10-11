package ride

import "context"

type RideRepo interface{
	CreateRide(ctx context.Context)
	CancelRide(ctx context.Context)
	GetRide(ctx context.Context)
}