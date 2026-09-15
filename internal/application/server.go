package application

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/MrAndreID/goweb/internal/application/config"
	"github.com/MrAndreID/goweb/internal/application/view"
	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/MrAndreID/gomiddleware/v2"
	"github.com/MrAndreID/gopackage/v2"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/sirupsen/logrus"
)

func newServer(cfg *config.Config) (*echo.Echo, error) {
	var tag string = "internal.application.server.newServer."

	e := echo.New()

	e.Validator = gopackage.CustomValidator()
	e.JSONSerializer = gopackage.CustomJSONSerializer()

	renderer, err := view.NewRenderer(user.Views)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to build html renderer")

		return nil, err
	}

	e.Renderer = renderer

	e.HTTPErrorHandler = renderer.HTTPErrorHandler

	e.StaticFS("/static", view.StaticFS())

	e.Pre(middleware.RemoveTrailingSlash())
	e.Pre(gomiddleware.EchoSetRequestID)

	e.Use(middleware.Recover())

	if cfg.UseBodyDumpLog {
		e.Use(middleware.BodyDump(func(c *echo.Context, requestBody, responseBody []byte, err error) {
			request := struct {
				Header any    `json:"header"`
				Body   string `json:"body"`
			}{
				Header: c.Request().Header,
				Body:   string(requestBody),
			}

			response := struct {
				Header any    `json:"header"`
				Body   string `json:"body"`
			}{
				Header: c.Response().Header(),
				Body:   string(responseBody),
			}

			logrus.WithFields(logrus.Fields{
				"request":   request,
				"requestId": c.Get("RequestID"),
				"response":  response,
				"url":       c.Request().Host + c.Request().URL.String(),
			}).Info("body dump")
		}))
	}

	secureMiddleware := middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            63072000,
		HSTSPreloadEnabled:    true,
		ContentSecurityPolicy: "default-src 'self'; style-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}

	if cfg.AppDebug {
		secureMiddleware.HSTSMaxAge = 0
		secureMiddleware.HSTSPreloadEnabled = false
	}

	e.Use(middleware.SecureWithConfig(secureMiddleware))
	e.Use(middleware.RequestLogger())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.AllowedOrigins,
		AllowHeaders: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
	}))

	e.Use(gomiddleware.EchoSetNoCache)
	e.Use(gomiddleware.EchoSetMaintenanceMode("storage/maintenance.flag"))

	e.Use(middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Timeout: time.Duration(cfg.AppTimeout) * time.Second,
		ErrorHandler: func(c *echo.Context, err error) error {
			if errors.Is(err, context.DeadlineExceeded) {
				return echo.NewHTTPError(http.StatusRequestTimeout, "REQUEST_TIMEOUT")
			}

			return err
		},
	}))

	rateLimiterConfig := middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: cfg.AppRateLimit, Burst: int(cfg.AppRateLimit), ExpiresIn: time.Duration(cfg.AppRateLimitDeadline) * time.Second},
		),
		ErrorHandler: func(c *echo.Context, err error) error {
			return echo.NewHTTPError(http.StatusForbidden, "FORBIDDEN")
		},
		DenyHandler: func(c *echo.Context, identifier string, err error) error {
			return echo.NewHTTPError(http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
		},
	}

	e.Use(middleware.RateLimiterWithConfig(rateLimiterConfig))

	return e, nil
}
