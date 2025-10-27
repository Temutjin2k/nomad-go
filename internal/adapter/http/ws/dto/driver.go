package dto

import (
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type OfferResp struct {
	MsgType         string                  `json:"type"` // By default must be: "ride_response"
	ID              uuid.UUID               `json:"offer_id"`
	RideID          uuid.UUID               `json:"ride_id"`
	Accepted        bool                    `json:"accepted"`
	CurrentLocation dto.CoordinateUpdateReq `json:"current_location"`
}

func (r *OfferResp) Validate(v *validator.Validator) {
	v.Check(r.ID != uuid.NilUUID, "offer_id", "must be provided")
	v.Check(r.RideID != uuid.NilUUID, "ride_id", "must be provided")
	v.Check(r.MsgType != "ride_response", "type", "must be: ride_response type")
	r.CurrentLocation.Validate(v)
}

// Websocket message: From Driver ← Location Update:
type DriverLocationUpdate struct {
	MsgType string `json:"type"` // by default must be: `location_update`
	dto.UpdateLocationReq
}

func (r *DriverLocationUpdate) Validate(v *validator.Validator) {
	v.Check(r.MsgType != "", "type", "must be: location_update type")
	r.UpdateLocationReq.Validate(v)
}
