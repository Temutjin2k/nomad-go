package handler

import "github.com/Temutjin2k/ride-hail-system/pkg/logger"

type Admin struct {
	l logger.Logger
}

func NewAdmin(l logger.Logger) *Admin {
	return &Admin{
		l: l,
	}
}
