package logger

import "github.com/sirupsen/logrus"

type CustomFormatter struct {
	logrus.TextFormatter
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("15:04:05") // Hour:Minute:Second format

	level := entry.Level.String()
	message := entry.Message

	log := timestamp + " " + f.colorizeLevel(level) + " " + message + "\n"

	return []byte(log), nil
}

func (f *CustomFormatter) colorizeLevel(level string) string {
	var color string

	switch level {
	case "info":
		color = "\033[32m" // Green
	case "warning":
		color = "\033[33m" // Yellow
	case "error":
		color = "\033[31m" // Red
	case "debug":
		color = "\033[34m" // Blue
	default:
		color = "\033[37m" // Default color
	}

	return color + level + "\033[0m"
}

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&CustomFormatter{})
	log.SetLevel(logrus.DebugLevel)
	return log
}
