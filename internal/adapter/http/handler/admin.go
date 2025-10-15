package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

type AdminService interface {
	GetOverview(ctx context.Context) (*models.OverviewResponse, error)
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
	ctx = wrap.WithAction(ctx, "admin_get_overview")

	overview, err := h.s.GetOverview(ctx)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to get overview", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Debug(ctx, "fetched overview", "timestamp", overview.Timestamp)

	if err := writeJSON(w, http.StatusOK, overview, nil); err != nil {
		h.l.Error(ctx, "failed to write response", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Admin) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = wrap.WithAction(ctx, "admin_get_active_rides")

	rides, err := h.s.GetActiveRides(ctx)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to get active rides", err)
		internalErrorResponse(w, err.Error())
		return
	}

	h.l.Debug(ctx, "fetched active rides", "count", len(rides.Rides))

	if err := writeJSON(w, http.StatusOK, rides, nil); err != nil {
		h.l.Error(ctx, "failed to write response", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
