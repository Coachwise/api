// Package logger is the application's single, consistently-formatted logger.
// Use it instead of the stdlib log package so every subsystem (API, workers,
// events) emits uniform, level-controlled output.
package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Log is the shared logger instance.
var Log = build()

func build() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	l.SetLevel(levelFromEnv())
	return l
}

func levelFromEnv() logrus.Level {
	if lvl, err := logrus.ParseLevel(strings.ToLower(os.Getenv("LOG_LEVEL"))); err == nil {
		return lvl
	}
	return logrus.InfoLevel
}

// Fields is re-exported so callers don't need to import logrus directly.
type Fields = logrus.Fields

func WithFields(f Fields) *logrus.Entry      { return Log.WithFields(f) }
func Info(a ...interface{})                  { Log.Info(a...) }
func Warn(a ...interface{})                  { Log.Warn(a...) }
func Error(a ...interface{})                 { Log.Error(a...) }
func Infof(format string, a ...interface{})  { Log.Infof(format, a...) }
func Warnf(format string, a ...interface{})  { Log.Warnf(format, a...) }
func Errorf(format string, a ...interface{}) { Log.Errorf(format, a...) }
func Fatalf(format string, a ...interface{}) { Log.Fatalf(format, a...) }
