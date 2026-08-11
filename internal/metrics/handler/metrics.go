package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/tashirka1/k2/internal/core/session"
	core_view "github.com/tashirka1/k2/internal/core/view"
	"github.com/tashirka1/k2/internal/metrics/model"
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

func (h *Metrics) System(c echo.Context) error {
	return core_view.RenderTemplate(c, view.System())
}

func parseSort(c echo.Context) model.Sort {
	return model.Sort{Field: c.QueryParam("sort"), Desc: c.QueryParam("dir") == "desc"}
}

func (h *Metrics) Processes(c echo.Context) error {
	q := c.QueryParam("q")
	s := parseSort(c)
	points, err := h.s.SearchProcess(c.Request().Context(), q, s)
	if err != nil {
		slog.Error("processes query failed", "error", err)
		return c.String(http.StatusInternalServerError, "query failed")
	}
	return core_view.RenderTemplate(c, view.ProcessesPage(points, q, s))
}

func (h *Metrics) Containers(c echo.Context) error {
	q := c.QueryParam("q")
	s := parseSort(c)
	points, err := h.s.SearchContainer(c.Request().Context(), q, s)
	if err != nil {
		slog.Error("containers query failed", "error", err)
		return c.String(http.StatusInternalServerError, "query failed")
	}
	return core_view.RenderTemplate(c, view.ContainersPage(points, q, s))
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

	data, err := h.s.QueryChartData(c.Request().Context(), metricType, from, to)
	if err != nil {
		slog.Error("chart data query failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query failed"})
	}

	return c.JSON(http.StatusOK, data)
}

func (h *Metrics) Search(c echo.Context) error {
	q := c.QueryParam("q")
	category := c.Param("category")
	s := parseSort(c)

	ctx := c.Request().Context()
	isHX := c.Request().Header.Get("HX-Request") != ""

	switch category {
	case "process":
		results, err := h.s.SearchProcess(ctx, q, s)
		if err != nil {
			slog.Error("search failed", "error", err)
			return c.String(http.StatusInternalServerError, "search failed")
		}
		if isHX {
			return core_view.RenderTemplate(c, view.ProcessResults(results, q, s))
		}
		return core_view.RenderTemplate(c, view.ProcessesPage(results, q, s))
	case "container":
		results, err := h.s.SearchContainer(ctx, q, s)
		if err != nil {
			slog.Error("search failed", "error", err)
			return c.String(http.StatusInternalServerError, "search failed")
		}
		if isHX {
			return core_view.RenderTemplate(c, view.ContainerResults(results, q, s))
		}
		return core_view.RenderTemplate(c, view.ContainersPage(results, q, s))
	case "resource":
		results, err := h.s.SearchResource(ctx, q)
		if err != nil {
			slog.Error("search failed", "error", err)
			return c.String(http.StatusInternalServerError, "search failed")
		}
		return core_view.RenderTemplate(c, view.ResourceResults(results))
	default:
		return c.String(http.StatusBadRequest, "unknown category")
	}
}

func SetupHandlers(e *echo.Echo, s service.MetricsService) {
	h := NewMetrics(s)

	group := e.Group("/metrics", session.RequireAuth())
	group.GET("/system", h.System)
	group.GET("/processes", h.Processes)
	group.GET("/containers", h.Containers)
	group.GET("/chart/:type", h.ChartData)
	group.GET("/search/:category", h.Search)
}
