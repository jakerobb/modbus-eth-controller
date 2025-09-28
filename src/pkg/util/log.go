package util

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

func GetLogger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func LogDebug(ctx context.Context, msg string, args ...interface{}) {
	if ctx.Value("debug") == true {
		GetLogger(ctx).Debug(msg, args...)
	}
}
