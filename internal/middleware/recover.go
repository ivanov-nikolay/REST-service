package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func RecoverWithLogger(log *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					log.WithFields(logrus.Fields{
						"panic":      fmt.Sprintf("%v", r),
						"stack":      string(stack),
						"uri":        c.Request().RequestURI,
						"method":     c.Request().Method,
						"ip":         c.RealIP(),
						"user_agent": c.Request().UserAgent(),
					}).Error("panic recovered")
					c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				}
			}()
			return next(c)
		}
	}
}
