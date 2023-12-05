package log

import (
	"context"

	"golang.org/x/exp/slog"
)

func Error(ctx context.Context, msg string, err error, args ...any) {
	if err == nil {
		slog.ErrorContext(ctx, msg, args)
	} else {
		slog.ErrorContext(ctx, msg, "error", err, args)
	}
}

func Warn(ctx context.Context, msg string, err error, args ...any) {
	if err == nil {
		slog.WarnContext(ctx, msg, args)
	} else {
		slog.WarnContext(ctx, msg, "error", err, args)
	}
}

func Info(ctx context.Context, msg string, err error, args ...any) {
	if err == nil {
		slog.InfoContext(ctx, msg, args)
	} else {
		slog.InfoContext(ctx, msg, "error", err, args)
	}
}

func Debug(ctx context.Context, msg string, err error, args ...any) {
	if err == nil {
		slog.DebugContext(ctx, msg, args)
	} else {
		slog.DebugContext(ctx, msg, "error", err, args)
	}
}
