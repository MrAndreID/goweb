package config

import (
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AppName              string  `env:"APP_NAME" envDefault:"GoWeb"`
	AppPort              string  `env:"APP_PORT,notEmpty"`
	AppDebug             bool    `env:"APP_DEBUG" envDefault:"false"`
	AppVersion           string  `env:"APP_VERSION" envDefault:"v1.0.0"`
	AppTimeout           int     `env:"APP_TIMEOUT" envDefault:"5"`
	AppShutdownTimeout   int     `env:"APP_SHUTDOWN_TIMEOUT" envDefault:"10"`
	AppRateLimit         float64 `env:"APP_RATE_LIMIT" envDefault:"10.0"`
	AppRateLimitDeadline int     `env:"APP_RATE_LIMIT_DEADLINE" envDefault:"60"`

	UseBodyDumpLog           bool `env:"USE_BODY_DUMP_LOG" envDefault:"false"`
	BodyDumpLogRotationCount uint `env:"BODY_DUMP_LOG_ROTATION_COUNT" envDefault:"30"`

	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`

	BackendBaseURL          string `env:"BACKEND_BASE_URL" envDefault:"http://127.0.0.1:10001"`
	BackendAppKey           string `env:"BACKEND_APP_KEY"`
	BackendTimeout          int    `env:"BACKEND_TIMEOUT" envDefault:"10"`
	BackendMaxResponseBytes int    `env:"BACKEND_MAX_RESPONSE_BYTES" envDefault:"2097152"`
	BackendMaxIdleConns     int    `env:"BACKEND_MAX_IDLE_CONNS" envDefault:"100"`
	BackendMaxConnsPerHost  int    `env:"BACKEND_MAX_CONNS_PER_HOST" envDefault:"100"`
	BackendRetryCount       int    `env:"BACKEND_RETRY_COUNT" envDefault:"2"`
}

func New() (*Config, error) {
	var (
		tag string = "internal.application.config.main.New."
		cfg Config
	)

	logrus.SetFormatter(&logrus.JSONFormatter{})

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			logrus.WithFields(logrus.Fields{
				"tag":   tag + "01",
				"error": err.Error(),
			}).Error("failed to load environment file")

			return &cfg, err
		}
	} else if !os.IsNotExist(err) {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "02",
			"error": err.Error(),
		}).Error("failed to check environment file")

		return &cfg, err
	}

	if err := env.Parse(&cfg); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "03",
			"error": err.Error(),
		}).Error("failed to parse environment")

		return &cfg, err
	}

	if cfg.UseBodyDumpLog {
		if err := NewBodyDumpLog(cfg.BodyDumpLogRotationCount); err != nil {
			logrus.WithFields(logrus.Fields{
				"tag":   tag + "04",
				"error": err.Error(),
			}).Error("failed to initiate a body dump for log")

			return &cfg, err
		}
	}

	LoadVersion(&cfg)

	return &cfg, nil
}
