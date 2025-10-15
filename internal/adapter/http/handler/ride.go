package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type RideService interface {
    Create(ctx context.Context, ride *models.Ride) (*models.Ride, error) 
    Cancel(ctx context.Context, rideID uuid.UUID, reason string) (*models.Ride, error)
    Get(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
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

		domainModel, err := request.ToModel()
    if err != nil {
        errorResponse(w, http.StatusBadRequest, "invalid passenger_id format")
        return
    }

    createdRide, err := h.ride.Create(ctx, domainModel)
    if err != nil {
        h.l.Error(wrap.ErrorCtx(ctx, err), "failed to create ride", err)
        errorResponse(w, GetCode(err), err.Error())
        return
    }

		response := envelope{
			"ride_id": createdRide.ID,
			"ride_number": createdRide.RideNumber,
			"status": createdRide.Status,
			"estimated_fare": createdRide.EstimatedFare,
			"estimated_duration_minutes": createdRide.EstimatedDurationMin,
			"estimated_distance_km": createdRide.EstimatedDistanceKm,
		}

    if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
        h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
        internalErrorResponse(w, err.Error())
    }
}

func (h *Ride) CancelRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "cancel_ride")

	rideIDstr := r.PathValue("ride_id") 
	rideID, err := uuid.Parse(rideIDstr)
	if err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid ride ID format")
			return
	}

	var request dto.CancelRideRequest

	v := validator.New()
	request.Validate(v)

	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	cancelledRide, err := h.ride.Cancel(ctx, rideID, request.Reason)
	if err != nil {
			h.l.Error(wrap.ErrorCtx(ctx, err), "failed to cancel ride", err)
			errorResponse(w, GetCode(err), err.Error())
			return
	}

	response := envelope{
		"ride_id": cancelledRide.ID,
		"status": cancelledRide.Status,
		"cancelled_at": cancelledRide.CancelledAt,
		"message": cancelledRide.CancellationReason,
	}

	if err := writeJSON(w, http.StatusAccepted, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}
}
