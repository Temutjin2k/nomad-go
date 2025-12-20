package handler

import (
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

type Health struct {
	serviceName string
	log         logger.Logger
}

func NewHealth(serviceName string, log logger.Logger) *Health {
	return &Health{
		serviceName: serviceName,
		log:         log,
	}
}

// HealthCheck godoc
// @Summary      Health Check
// @Description  Returns the health status of the service
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
// HealthCheck - returns system information.
func (a *Health) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "health_check")

	response := map[string]any{
		"status": "available",
		"system_info": map[string]string{
			"service-name": a.serviceName,
		},
	}

	if err := writeJSON(w, http.StatusOK, response, nil); err != nil {
		a.log.Error(ctx, "healthcheck", err)
		return
	}
}
