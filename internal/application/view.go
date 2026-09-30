package application

import (
	"github.com/MrAndreID/goweb/internal/application/view"
	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

func RegisterViews(e *echo.Echo) error {
	renderer, err := view.NewRenderer(user.Views)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   "internal.application.view.RegisterViews.01",
			"error": err.Error(),
		}).Error("failed to build html renderer")

		return err
	}

	e.Renderer = renderer

	e.HTTPErrorHandler = renderer.HTTPErrorHandler

	e.StaticFS("/static", view.StaticFS())

	return nil
}
