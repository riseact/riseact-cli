package logger

import (
	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func InitLogging(level logrus.Level) {
	Logger = logrus.New()
	Logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		PadLevelText:     true,
		QuoteEmptyFields: true,
	})
	SetLevel(level)
}

func SetLevel(level logrus.Level) {
	Logger.SetLevel(level)
}

func Info(msg string) {
	Logger.Info(msg)
}

func Debug(msg string) {
	Logger.Debug(msg)
}

func Warn(msg string) {
	Logger.Warn(msg)
}

func Error(msg string) {
	Logger.Error(msg)
}

func Infof(format string, args ...interface{}) {
	Logger.Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	Logger.Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Logger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Logger.Errorf(format, args...)
}
