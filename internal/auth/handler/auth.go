package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/tashirka1/k2/internal/auth/model"
	"github.com/tashirka1/k2/internal/auth/service"
	"github.com/tashirka1/k2/internal/auth/view"
	"github.com/tashirka1/k2/internal/core/session"
	core_view "github.com/tashirka1/k2/internal/core/view"

	"github.com/labstack/echo/v4"
)

type Auth struct {
	s service.AuthService
}

func NewAuth(s service.AuthService) *Auth {
	return &Auth{s: s}
}

func (h *Auth) Root(c echo.Context) error {
	userId := session.GetUserId(c)
	if userId != 0 {
		return c.Redirect(http.StatusSeeOther, "/dashboard")
	}
	return core_view.RenderTemplate(c, view.Root())
}

func (h *Auth) GetLogin(c echo.Context) error {
	userId := session.GetUserId(c)
	if userId != 0 {
		return c.Redirect(http.StatusSeeOther, "/dashboard")
	}
	return core_view.RenderTemplate(c, view.Login(0))
}

func (h *Auth) PostLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" {
		c.Response().Header().Set("HX-Retarget", "#errors")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return core_view.RenderTemplate(c, view.LoginError("username is required"))
	}
	if password == "" {
		c.Response().Header().Set("HX-Retarget", "#errors")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return core_view.RenderTemplate(c, view.LoginError("password is required"))
	}

	ok, err := h.s.CheckLogin(c.Request().Context(), username, password, time.Now())
	if err != nil {
		c.Response().Header().Set("HX-Retarget", "#errors")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		if errors.Is(err, model.ErrInvalidLogin) {
			return core_view.RenderTemplate(c, view.LoginError("invalid username or password"))
		}
		if errors.Is(err, model.ErrLockedOut) {
			return core_view.RenderTemplate(c, view.LoginError("account locked. try again in 1 minute"))
		}
		slog.Error("login error", "error", err)
		return core_view.RenderTemplate(c, view.LoginError("internal error"))
	}
	if !ok {
		c.Response().Header().Set("HX-Retarget", "#errors")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return core_view.RenderTemplate(c, view.LoginError("invalid username or password"))
	}

	session.SetUserId(c, 1)
	c.Response().Header().Set("HX-Redirect", "/dashboard")
	return nil
}

func (h *Auth) Logout(c echo.Context) error {
	session.ClearSession(c)
	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *Auth) Dashboard(c echo.Context) error {
	return core_view.RenderTemplate(c, view.Dashboard(1))
}

func SetupHandlers(e *echo.Echo, s service.AuthService) {
	h := NewAuth(s)

	e.GET("/", h.Root)
	e.GET("/login", h.GetLogin)
	e.POST("/login", h.PostLogin)
	e.GET("/logout", h.Logout)
	e.GET("/dashboard", session.RequireAuth()(h.Dashboard))
}
