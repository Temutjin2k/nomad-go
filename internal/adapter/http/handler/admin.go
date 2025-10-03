package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type AdminService interface {
	GetOverview(ctx context.Context) (any, error)
	GetActiveRides(ctx context.Context) (*models.ActiveRidesResponse, error)
}

type Admin struct {
	s AdminService
	l logger.Logger
}

func NewAdmin(s AdminService, l logger.Logger) *Admin {
	return &Admin{
		s: s,
		l: l,
	}
}

func (h *Admin) GetOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	res, err := h.s.GetOverview(ctx)
	if err != nil {
		h.l.Error(logger.ErrorCtx(ctx, err), "GetOverview: failed to get overview", err)
		internalErrorResponse(w, err.Error())
		return
	}

	if err := writeJSON(w, http.StatusOK, envelope{"data": res}, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Admin) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	res, err := h.s.GetActiveRides(ctx)
	if err != nil {
		h.l.Error(logger.ErrorCtx(ctx, err), "GetActiveRides: failed to get active rides", err)
		internalErrorResponse(w, err.Error())
		return
	}

	if err := writeJSON(w, http.StatusOK, envelope{"data": res}, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
