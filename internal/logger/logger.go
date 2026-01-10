package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[37m"
)

type PrettyHandler struct {
	mu  sync.Mutex
	out io.Writer
}

func NewPrettyHandler(out io.Writer) *PrettyHandler {
	return &PrettyHandler{
		mu:  sync.Mutex{},
		out: out,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	timestamp := r.Time.Format(time.TimeOnly)
	level := h.colorizeLevel(r.Level)
	msg := r.Message

	line := fmt.Sprintf("%s %s %s", timestamp, level, msg)

	r.Attrs(func(a slog.Attr) bool {
		line += fmt.Sprintf(" %s%s%s=%v", colorGray, a.Key, colorReset, a.Value.Any())
		return true
	})

	_, err := fmt.Fprintln(h.out, line)
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *PrettyHandler) colorizeLevel(level slog.Level) string {
	var color string
	var name string

	switch level {
	case slog.LevelDebug:
		color = colorBlue
		name = "DEBUG"
	case slog.LevelInfo:
		color = colorGreen
		name = "INFO"
	case slog.LevelWarn:
		color = colorYellow
		name = "WARN"
	case slog.LevelError:
		color = colorRed
		name = "ERROR"
	default:
		color = colorGray
		name = level.String()
	}

	return fmt.Sprintf("%s%-5s%s", color, name, colorReset)
}

func NewLogger() *slog.Logger {
	handler := NewPrettyHandler(os.Stdout)
	return slog.New(handler)
}
