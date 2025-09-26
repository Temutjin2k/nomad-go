package handler

import "github.com/Temutjin2k/ride-hail-system/pkg/logger"

type Driver struct {
	l logger.Logger
}

func NewDriver(l logger.Logger) *Driver {
	return &Driver{
		l: l,
	}
}
