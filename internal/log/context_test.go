package log_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/HallyG/fingrab/internal/log"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupCtx    func(t *testing.T) (context.Context, *slog.Logger)
		expectEqual bool
	}{
		"logger in context": {
			setupCtx: func(t *testing.T) (context.Context, *slog.Logger) {
				t.Helper()

				logger := slog.New(slog.DiscardHandler)
				ctx := log.WithContext(t.Context(), logger)
				return ctx, logger
			},
		},
		"no logger in context": {
			setupCtx: func(t *testing.T) (context.Context, *slog.Logger) {
				t.Helper()

				logger := log.New(log.WithWriter(nil))
				return t.Context(), logger
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, originalLogger := test.setupCtx(t)
			logger := log.FromContext(ctx)

			require.NotNil(t, originalLogger)
			require.NotNil(t, logger)
			require.Equal(t, originalLogger, logger)
		})
	}
}
