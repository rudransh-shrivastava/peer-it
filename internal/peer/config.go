package peer

import "github.com/sirupsen/logrus"

type Config struct {
	Addr        string
	Logger      *logrus.Logger
	TrackerAddr string
}
