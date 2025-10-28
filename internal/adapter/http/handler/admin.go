package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type AdminService interface {
	Overview(ctx context.Context) (*models.OverviewResponse, error)
	ActiveRides(ctx context.Context, filters models.Filters) (*models.ActiveRidesResponse, error)
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

	overview, err := h.s.Overview(ctx)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to get overview", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	h.l.Debug(ctx, "fetched overview", "metrics", overview.Metrics)

	if err := writeJSON(w, http.StatusOK, overview, nil); err != nil {
		h.l.Error(ctx, "failed to write response", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

var activeRidesSortSafeList = []string{"ride_number", "started_at", "estimated_completion", "created_at", "-ride_number", "-started_at", "-estimated_completion", "-created_at"}

func (h *Admin) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = wrap.WithAction(ctx, "admin_get_active_rides")

	v := validator.New()
	qs := r.URL.Query()

	// filter options
	page := readInt(qs, "page", 1, v)
	pageSize := readInt(qs, "page_size", 20, v)
	sort := readString(qs, "sort", "created_at")

	filters, err := models.NewFilters(page, pageSize, sort, activeRidesSortSafeList)
	if err != nil {
		internalErrorResponse(w, "intenal error")
		return
	}

	filters.Validate(v)

	if !v.Valid() {
		failedValidationResponse(w, v.Errors)
		return
	}

	rides, err := h.s.ActiveRides(ctx, filters)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to get active rides", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	h.l.Debug(ctx, "fetched active rides", "total", rides.Metadata.TotalRecords)

	if err := writeJSON(w, http.StatusOK, rides, nil); err != nil {
		h.l.Error(ctx, "failed to write response", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
