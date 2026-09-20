package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

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

	if err := validate(&cfg); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "04",
			"error": err.Error(),
		}).Error("invalid application configuration")

		return &cfg, err
	}

	if cfg.UseBodyDumpLog {
		if err := NewBodyDumpLog(cfg.BodyDumpLogRotationCount); err != nil {
			logrus.WithFields(logrus.Fields{
				"tag":   tag + "05",
				"error": err.Error(),
			}).Error("failed to initiate a body dump for log")

			return &cfg, err
		}
	}

	LoadVersion(&cfg)

	return &cfg, nil
}

func validate(cfg *Config) error {
	port, err := strconv.Atoi(strings.TrimSpace(cfg.AppPort))

	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("APP_PORT must be a valid TCP port between 1 and 65535")
	}

	positiveInts := map[string]int{
		"APP_TIMEOUT":                cfg.AppTimeout,
		"APP_SHUTDOWN_TIMEOUT":       cfg.AppShutdownTimeout,
		"APP_RATE_LIMIT_DEADLINE":    cfg.AppRateLimitDeadline,
		"BACKEND_TIMEOUT":            cfg.BackendTimeout,
		"BACKEND_MAX_RESPONSE_BYTES": cfg.BackendMaxResponseBytes,
		"BACKEND_MAX_IDLE_CONNS":     cfg.BackendMaxIdleConns,
		"BACKEND_MAX_CONNS_PER_HOST": cfg.BackendMaxConnsPerHost,
	}

	for name, value := range positiveInts {
		if value <= 0 {
			return fmt.Errorf("%s must be greater than zero", name)
		}
	}

	if cfg.AppRateLimit <= 0 {
		return fmt.Errorf("APP_RATE_LIMIT must be greater than zero")
	}

	if cfg.BackendRetryCount < 0 {
		return fmt.Errorf("BACKEND_RETRY_COUNT must not be negative")
	}

	if cfg.UseBodyDumpLog && cfg.BodyDumpLogRotationCount == 0 {
		return fmt.Errorf("BODY_DUMP_LOG_ROTATION_COUNT must be greater than zero when USE_BODY_DUMP_LOG is enabled")
	}

	backendURL, err := url.ParseRequestURI(strings.TrimSpace(cfg.BackendBaseURL))

	if err != nil || backendURL.Host == "" || (backendURL.Scheme != "http" && backendURL.Scheme != "https") {
		return fmt.Errorf("BACKEND_BASE_URL must be a valid HTTP or HTTPS URL")
	}

	for _, origin := range cfg.AllowedOrigins {
		parsedOrigin, err := url.ParseRequestURI(strings.TrimSpace(origin))

		if err != nil || parsedOrigin.Host == "" || parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" || (parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https") {
			return fmt.Errorf("ALLOWED_ORIGINS contains an invalid origin")
		}
	}

	return nil
}
