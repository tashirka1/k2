package session

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

const (
	UserSessionsKey string = "user"
	UserIdKey       string = "userId"
)

func GetUserId(c echo.Context) int {
	sess, err := session.Get(UserSessionsKey, c)
	if err != nil || sess == nil {
		slog.Warn("session get failed", "error", err)
		return 0
	}
	userId, ok := sess.Values[UserIdKey].(int)
	if !ok {
		return 0
	}
	return userId
}

func ClearSession(c echo.Context) {
	sess, err := session.Get(UserSessionsKey, c)
	if err != nil || sess == nil {
		return
	}
	sess.Options.MaxAge = -1
	_ = sess.Save(c.Request(), c.Response())
}

func SetUserId(c echo.Context, userId int) {
	sess, err := session.Get(UserSessionsKey, c)
	if err != nil || sess == nil {
		slog.Error("session get failed", "error", err)
		return
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	sess.Values[UserIdKey] = userId
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		slog.Error("session save failed", "error", err)
	}
}
