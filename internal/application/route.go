package application

import (
	"net/http"

	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/labstack/echo/v5"
)

func RegisterRoutes(e *echo.Echo) {
	e.GET("/", func(c *echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/users")
	})

	user.NewHandler(e.Group(""), UserService)
}
