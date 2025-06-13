package logging

import (
	"context"
	"net/http"
	"os"

	"log/slog"
)

type Logger struct {
	*slog.Logger
}

func New() *Logger {
	base := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)
	return &Logger{Logger: base}
}

func (l *Logger) LogRequest(ctx context.Context, r *http.Request, status int, user string, err error) {
	level := slog.LevelInfo
	if status > 299 {
		level = slog.LevelError
	}
	attrs := []any{
		"user", user,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"status", status,
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	l.Log(ctx, level, "request", attrs...)
}
