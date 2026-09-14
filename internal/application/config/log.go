package config

import (
	"io"
	"os"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

const defaultBodyDumpLogRotationCount uint = 30

func NewBodyDumpLog(rotationCount uint) error {
	var tag string = "internal.application.config.log.NewBodyDumpLog."

	if rotationCount == 0 {
		logrus.WithFields(logrus.Fields{
			"tag": tag + "01",
		}).Warn("body dump log rotation count is zero, falling back to the default rotation count")

		rotationCount = defaultBodyDumpLogRotationCount
	}

	dir, err := os.Getwd()

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "02",
			"error": err.Error(),
		}).Error("failed to get root path")

		return err
	}

	logfile, err := rotatelogs.New(
		dir+"/storage/log/%Y%m%d.log",
		rotatelogs.WithLinkName(dir+"/storage/log/master.log"),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(-1),
		rotatelogs.WithRotationCount(rotationCount),
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "03",
			"error": err.Error(),
		}).Error("failed to create a new rotate log")

		return err
	}

	logrus.SetFormatter(&logrus.JSONFormatter{DisableHTMLEscape: true})
	logrus.SetOutput(io.MultiWriter(os.Stdout, logfile))
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetReportCaller(true)

	return nil
}
