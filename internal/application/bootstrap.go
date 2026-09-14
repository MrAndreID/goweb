package application

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MrAndreID/goweb/internal/application/config"
	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

var (
	UserService *user.Service
)

func initService(app *Application) {
	UserService = user.NewService(
		user.NewRepository(user.RepositoryConfig{
			BaseURL:          app.Config.BackendBaseURL,
			AppKey:           app.Config.BackendAppKey,
			Timeout:          app.Config.BackendTimeout,
			MaxResponseBytes: app.Config.BackendMaxResponseBytes,
			MaxIdleConns:     app.Config.BackendMaxIdleConns,
			MaxConnsPerHost:  app.Config.BackendMaxConnsPerHost,
			RetryCount:       app.Config.BackendRetryCount,
		}),
	)
}

func Start() error {
	var tag string = "internal.application.bootstrap.Start."

	cfg, err := config.New()

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to initiate configuration")

		return err
	}

	app := &Application{
		Config: cfg,
	}

	initService(app)

	e, err := newServer(cfg)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "02",
			"error": err.Error(),
		}).Error("failed to initiate server")

		return err
	}

	RegisterRoutes(e)

	return run(e, cfg)
}

func run(e *echo.Echo, cfg *config.Config) error {
	var tag string = "internal.application.bootstrap.run."

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	startConfig := echo.StartConfig{
		Address:         ":" + cfg.AppPort,
		GracefulTimeout: time.Duration(cfg.AppShutdownTimeout) * time.Second,
		OnShutdownError: func(err error) {
			logrus.WithFields(logrus.Fields{
				"tag":   tag + "01",
				"error": err.Error(),
			}).Error("failed to drain in flight requests within the shutdown timeout")
		},
	}

	if err := startConfig.Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "02",
			"error": err.Error(),
		}).Error("failed to serve http")

		return err
	}

	logrus.WithFields(logrus.Fields{
		"tag": tag + "03",
	}).Info("server stopped, releasing connections")

	return nil
}
