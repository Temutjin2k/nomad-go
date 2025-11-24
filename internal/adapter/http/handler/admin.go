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

// GetOverview godoc
// @Summary      Get system overview
// @Description  Get system metrics and statistics overview
// @Tags         admin
// @Produce      json
// @Success      200 {object} models.OverviewResponse "System overview data"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      403 {object} map[string]interface{} "Forbidden - Admin only"
// @Failure      500 {object} map[string]interface{} "Internal server error"
// @Security     BearerAuth
// @Router       /admin/overview [get]
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

// GetActiveRides godoc
// @Summary      Get active rides
// @Description  Get list of all currently active rides with pagination and filtering
// @Tags         admin
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Param        sort query string false "Sort field" default(created_at)
// @Success      200 {object} models.ActiveRidesResponse "List of active rides"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      403 {object} map[string]interface{} "Forbidden - Admin only"
// @Failure      422 {object} map[string]interface{} "Validation error"
// @Failure      500 {object} map[string]interface{} "Internal server error"
// @Security     BearerAuth
// @Router       /admin/rides/active [get]
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
