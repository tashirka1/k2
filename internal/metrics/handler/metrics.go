package handler

import (
	"log/slog"
	"net/http"
	"time"

	core_view "github.com/tashirka1/k2/internal/core/view"
	"github.com/tashirka1/k2/internal/metrics/service"
	"github.com/tashirka1/k2/internal/metrics/view"

	"github.com/labstack/echo/v4"
)

type Metrics struct {
	s service.MetricsService
}

func NewMetrics(s service.MetricsService) *Metrics {
	return &Metrics{s: s}
}

func (h *Metrics) Dashboard(c echo.Context) error {
	return core_view.RenderTemplate(c, view.Dashboard())
}

func (h *Metrics) Processes(c echo.Context) error {
	return core_view.RenderTemplate(c, view.ProcessesPage())
}

func (h *Metrics) Containers(c echo.Context) error {
	return core_view.RenderTemplate(c, view.ContainersPage())
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

func (h *Metrics) ChartData(c echo.Context) error {
	metricType := c.Param("type")
	period := c.QueryParam("period")
	if period == "" {
		period = "1h"
	}
	dur := parseDuration(period, time.Hour)
	from := time.Now().Add(-dur)
	to := time.Now()

	data, err := h.s.QueryResources(c.Request().Context(), metricType, from, to)
	if err != nil {
		slog.Error("chart data query failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query failed"})
	}

	return c.JSON(http.StatusOK, data)
}

func (h *Metrics) Search(c echo.Context) error {
	q := c.QueryParam("q")
	category := c.Param("category")

	results, err := h.s.SearchFTS(c.Request().Context(), category, q)
	if err != nil {
		slog.Error("search failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "search failed"})
	}

	return c.JSON(http.StatusOK, results)
}

func SetupHandlers(e *echo.Echo, s service.MetricsService) {
	h := NewMetrics(s)

	group := e.Group("/metrics")
	group.GET("/dashboard", h.Dashboard)
	group.GET("/processes", h.Processes)
	group.GET("/containers", h.Containers)
	group.GET("/chart/:type", h.ChartData)
	group.GET("/search/:category", h.Search)
}
