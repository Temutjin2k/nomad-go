package dto

import (
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type RegisterReq struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name"`
	LicenseNumber string         `json:"license_number"`
	Vehicle       models.Vehicle `json:"vehicle"`
}

func (r *RegisterReq) Validate(v *validator.Validator) {
	// ID
	v.Check(r.ID != uuid.UUID{}, "id", "must be provided")

	// Name
	v.Check(r.Name != "", "name", "must be provided")
	v.Check(len(r.Name) < 100, "name", "must be less than 100 characters")

	// License Number
	v.Check(r.LicenseNumber != "", "license_number", "must be provided")
	v.Check(len(r.LicenseNumber) < 10, "license_number", "must be less than 10 characters")

	// Vehicle.Make
	v.Check(r.Vehicle.Make != "", "vehicle.make", "must be provided")
	v.Check(len(r.Vehicle.Make) < 50, "vehicle.make", "must be less than 50 characters")

	// Vehicle.Model
	v.Check(r.Vehicle.Model != "", "vehicle.model", "must be provided")
	v.Check(len(r.Vehicle.Model) < 50, "vehicle.model", "must be less than 50 characters")

	// Vehicle.Color
	v.Check(r.Vehicle.Color != "", "vehicle.color", "must be provided")
	v.Check(len(r.Vehicle.Color) < 30, "vehicle.color", "must be less than 30 characters")

	// Vehicle.Plate
	v.Check(r.Vehicle.Plate != "", "vehicle.plate", "must be provided")
	v.Check(len(r.Vehicle.Plate) < 12, "vehicle.plate", "must be less than 12 characters")

	// Vehicle.Year
	v.Check(r.Vehicle.Year != 0, "vehicle.year", "must be provided")
	v.Check(
		r.Vehicle.Year >= 1886 && r.Vehicle.Year <= time.Now().Year(),
		"vehicle.year",
		fmt.Sprintf("must be between 1886 and %d", time.Now().Year()),
	)
}

func (r *RegisterReq) ToModel() *models.Driver {
	return &models.Driver{
		ID:            r.ID,
		Name:          r.Name,
		LicenseNumber: r.LicenseNumber,
		Vehicle:       r.Vehicle,
	}
}

type LocationUpdateReq struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func (r *LocationUpdateReq) Validate(v *validator.Validator) {
	if r.Latitude != nil && r.Longitude != nil {
		v.Check(*r.Latitude >= -90 && *r.Latitude <= 90, "latitude", "must be between -90 and 90")
		v.Check(*r.Longitude >= -180 && *r.Latitude <= 180, "longitude", "must be between -90 and 90")
	} else {
		v.Check(r.Latitude != nil, "latitude", "must be provided")
		v.Check(r.Longitude != nil, "longitude", "must be provided")
	}
}
