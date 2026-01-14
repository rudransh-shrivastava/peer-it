package tracker

import "log/slog"

type Config struct {
	Addr     string
	Listener Listener
	Logger   *slog.Logger
}
