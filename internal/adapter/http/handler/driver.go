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

type Driver struct {
	service DriverService
	l       logger.Logger
}

type DriverService interface {
	Register(ctx context.Context, newDriver *models.Driver) error
	GoOnline(ctx context.Context, driverID uuid.UUID, latitude, longitude float64) (sessionID uuid.UUID, err error)
}

func NewDriver(service DriverService, l logger.Logger) *Driver {
	return &Driver{
		service: service,
		l:       l,
	}
}
func (h *Driver) Register(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "register_driver")

	var RegisterReq dto.RegisterReq
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
}

func (h *Driver) GoOnline(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "set_driver_online")

	driverID, err := uuid.Parse(r.PathValue("driver_id"))
	if err != nil {
		h.l.Warn(ctx, "invalid driver uuid format")
		errorResponse(w, http.StatusBadRequest, "invalid driver uuid format")
		return
	}

	var goOnlineReq dto.LocationUpdateReq
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

	sessionID, err := h.service.GoOnline(ctx, driverID, *goOnlineReq.Latitude, *goOnlineReq.Longitude)
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
}
