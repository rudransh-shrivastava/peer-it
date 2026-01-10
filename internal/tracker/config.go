package tracker

import "log/slog"

type Config struct {
	Addr   string
	Logger *slog.Logger
}
