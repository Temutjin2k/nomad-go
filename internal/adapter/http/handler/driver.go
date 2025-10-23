package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	drivergo "github.com/Temutjin2k/ride-hail-system/internal/service/driver"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type Driver struct {
	service DriverService
	l       logger.Logger
}

type DriverService interface {
	Register(ctx context.Context, newDriver *models.Driver) error
	GoOnline(ctx context.Context, driverID uuid.UUID, location models.Location) (sessionID uuid.UUID, err error)
	GoOffline(ctx context.Context, driverID uuid.UUID) (models.SessionSummary, error)
	StartRide(ctx context.Context, startTime time.Time, driverID, rideID uuid.UUID, location models.Location) error
	CompleteRide(ctx context.Context, rideID uuid.UUID, data drivergo.CompleteRideData) (earnings float64, err error)
}

func NewDriver(service DriverService, l logger.Logger) *Driver {
	return &Driver{
		service: service,
		l:       l,
	}
}
func (h *Driver) Register(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "register_driver")

	var RegisterReq dto.RegisterDriverRequest
	if err := readJSON(w, r, &RegisterReq); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	RegisterReq.Validate(v)
	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	driver := RegisterReq.ToModel()
	if err := h.service.Register(ctx, driver); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to register a new driver", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"is_verified": driver.IsVerified,
		"class":       driver.Vehicle.Type,
	}

	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Info(ctx, "driver registered successfully", "driver_id", driver.ID)
}

func (h *Driver) GoOnline(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "set_driver_online")

	driverID, err := uuid.Parse(r.PathValue("driver_id"))
	if err != nil {
		h.l.Warn(ctx, "invalid driver uuid format")
		errorResponse(w, http.StatusBadRequest, "invalid driver uuid format")
		return
	}

	var goOnlineReq dto.CoordinateUpdateReq
	if err := readJSON(w, r, &goOnlineReq); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	goOnlineReq.Validate(v)
	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	sessionID, err := h.service.GoOnline(ctx, driverID, models.Location{Latitude: *goOnlineReq.Latitude, Longitude: *goOnlineReq.Longitude})
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to set driver status to online", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"status":     "AVAILABLE",
		"message":    "You are now online and ready to accept rides",
		"session_id": sessionID,
	}

	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Info(ctx, "driver set to online successfully", "driver_id", driverID)
}

func (h *Driver) GoOffline(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "set_driver_offline")

	driverID, err := uuid.Parse(r.PathValue("driver_id"))
	if err != nil {
		h.l.Warn(ctx, "invalid driver uuid format")
		errorResponse(w, http.StatusBadRequest, "invalid driver uuid format")
		return
	}

	summary, err := h.service.GoOffline(ctx, driverID)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to set driver status to offline", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"status":     "OFFLINE",
		"message":    "You are now offline",
		"session_id": summary.SessionID,
		"session_summary": envelope{
			"duration_hours":  summary.DurationHours,
			"rides_completed": summary.RidesCompleted,
			"earnings":        summary.Earnings,
		},
	}

	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Info(ctx, "driver set to offline successfully", "driver_id", driverID)
}

func (h *Driver) StartRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "start_ride")

	driverID, err := uuid.Parse(r.PathValue("driver_id"))
	if err != nil {
		h.l.Warn(ctx, "invalid driver uuid format")
		errorResponse(w, http.StatusBadRequest, "invalid driver uuid format")
		return
	}

	var req dto.StartRideReq
	if err := readJSON(w, r, &req); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	req.Validate(v)
	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	now := time.Now()
	if err := h.service.StartRide(
		ctx, now, driverID, req.RideID,
		models.Location{
			Latitude:  *req.DriverLocation.Latitude,
			Longitude: *req.DriverLocation.Longitude,
		},
	); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to start ride", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"ride_id":    req.RideID,
		"status":     types.StatusDriverBusy,
		"started_at": now,
		"message":    "Ride started successfully",
	}

	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Info(ctx, "ride started successfully", "driver_id", driverID, "ride_id", req.RideID)
}

func (h *Driver) CompleteRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "complete_ride")

	driverID, err := uuid.Parse(r.PathValue("driver_id"))
	if err != nil {
		h.l.Warn(ctx, "invalid driver uuid format")
		errorResponse(w, http.StatusBadRequest, "invalid driver uuid format")
		return
	}

	var req dto.CompleteRideReq
	if err := readJSON(w, r, &req); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	req.Validate(v)
	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	now := time.Now()
	earnings, err := h.service.CompleteRide(
		ctx, req.RideID,
		drivergo.CompleteRideData{
			CompleteTime:      now,
			DriverID:          driverID,
			ActualDurationMin: req.ActualDurationMin,
			ActualDistanceKm:  req.ActualDistanceKm,
			Location: models.Location{
				Latitude:  *req.FinalLocation.Latitude,
				Longitude: *req.FinalLocation.Longitude,
			},
		})
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to complete ride", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"ride_id":         req.RideID,
		"status":          types.StatusDriverAvailable,
		"completed_at":    now,
		"driver_earnings": earnings,
		"message":         "Ride completed successfully",
	}

	if err := writeJSON(w, http.StatusOK, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Info(ctx, "ride finished successfully", "driver_id", driverID, "ride_id", req.RideID)
}
