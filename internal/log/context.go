package log

import (
	"context"
	"log/slog"
)

type (
	logCtxKey struct{}
)

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	ctx = context.WithValue(ctx, logCtxKey{}, logger)
	return context.WithValue(ctx, logCtxKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(logCtxKey{}).(*slog.Logger)
	if ok && logger != nil {
		return logger
	}

	return New(WithWriter(nil))
}
