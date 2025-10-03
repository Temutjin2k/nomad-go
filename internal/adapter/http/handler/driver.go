package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type Driver struct {
	service DriverService
	l       logger.Logger
}

type DriverService interface {
	Register(ctx context.Context, newDriver *models.Driver) error
}

func NewDriver(service DriverService, l logger.Logger) *Driver {
	return &Driver{
		service: service,
		l:       l,
	}
}
func (h *Driver) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = wrap.WithAction(ctx, "register_driver")

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
