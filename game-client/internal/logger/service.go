package logger

import (
	"github.com/db-mipt/game-client/internal/config"
	"github.com/sirupsen/logrus"
)

func NewLogger(cfg *config.Config) *logrus.Entry {
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	logrus.SetLevel(level)

	return logrus.WithFields(logrus.Fields{
		"app_name": cfg.AppName,
		"version":  cfg.AppVersion,
	})
}
