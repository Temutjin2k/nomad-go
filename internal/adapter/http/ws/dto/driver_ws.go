package dto

import (
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type OfferResp struct {
	ID              uuid.UUID               `json:"offer_id"`
	MsgType         string                  `json:"type"` // By default must be: "ride_offer"
	RideID          uuid.UUID               `json:"ride_id"`
	Accepted        bool                    `json:"accepted"`
	CurrentLocation dto.CoordinateUpdateReq `json:"current_location"`
}

func (r *OfferResp) Validate(v *validator.Validator) {
	v.Check(r.ID != uuid.NilUUID, "offer_id", "must be provided")
	v.Check(r.RideID != uuid.NilUUID, "ride_id", "must be provided")
	v.Check(r.MsgType != "ride_offer", "type", "must be: ride_offer type")
	r.CurrentLocation.Validate(v)
}
