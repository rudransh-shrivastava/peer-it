package tracker

import "github.com/sirupsen/logrus"

type Config struct {
	Addr   string
	Logger *logrus.Logger
}
