package handler

import "github.com/Temutjin2k/ride-hail-system/pkg/logger"

type Ride struct {
	l logger.Logger
}

func NewRide(l logger.Logger) *Ride {
	return &Ride{
		l: l,
	}
}
