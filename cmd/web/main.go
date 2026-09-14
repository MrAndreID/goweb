package main

import (
	"os"

	"github.com/MrAndreID/goweb/internal/application"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := application.Start(); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   "cmd.web.main.main.01",
			"error": err.Error(),
		}).Error("failed to start application")

		os.Exit(1)
	}
}
