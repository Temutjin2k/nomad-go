package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type RideService interface {
 Create(ctx context.Context, request *dto.CreateRideRequest) error
 Cancel(ctx context.Context, request *dto.CancelRideRequest) error
}

type Ride struct {
	l logger.Logger
	ride RideService
}

func NewRide(l logger.Logger, ride RideService) *Ride {
	return &Ride{
		l: l,
		ride: ride,
	}
}

func (h *Ride) CreateRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "create_ride")

	var request dto.CreateRideRequest
	if err := readJSON(w, r, &request); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	request.Validate(v)

	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	if err := h.ride.Create(ctx, &request); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to create ride", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{}


	// TODO: добавить причину
	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}
}

func (h *Ride) CancelRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "cancel_ride")
	var request dto.CancelRideRequest

	v := validator.New()
	request.Validate(v)

	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	if err := h.ride.Cancel(ctx, &request); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to cancel ride", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{}

	if err := writeJSON(w, http.StatusAccepted, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}
}
