package log_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/HallyG/fingrab/internal/log"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("zero opts", func(t *testing.T) {
		t.Parallel()

		logger := log.New()
		require.NotNil(t, logger)
	})

	t.Run("empty opts", func(t *testing.T) {
		t.Parallel()

		opts := make([]log.Option, 0)
		logger := log.New(opts...)
		require.NotNil(t, logger)

		require.Equal(t, slog.DiscardHandler, logger.Handler())
	})

	t.Run("nil opts", func(t *testing.T) {
		t.Parallel()

		logger := log.New(nil)
		require.NotNil(t, logger)

		require.Equal(t, slog.DiscardHandler, logger.Handler())
	})

	tests := map[string]struct {
		opts            []log.Option
		expectedAttrs   []string
		expectedJSON    bool
		expectedVerbose bool
	}{
		"verbose": {
			opts:            []log.Option{log.WithVerbose(true)},
			expectedVerbose: true,
		},
		"json format": {
			opts:         []log.Option{log.WithJSONHandler()},
			expectedJSON: true,
		},
		"text format": {
			opts:         []log.Option{log.WithTextHandler()},
			expectedJSON: false,
		},
		"populates source attribute": {
			expectedAttrs: []string{"source=log_test.go"},
		},
		"custom attributes": {
			opts:          []log.Option{log.WithAttrs(slog.String("very", "nice"))},
			expectedAttrs: []string{"very=nice"},
		},
		"overwritten source attribute": {
			opts:          []log.Option{log.WithAttrs(slog.String("source", "something else"))},
			expectedAttrs: []string{`source="something else"`},
		},
		"custom handler": {
			opts: []log.Option{
				log.WithHandler(func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
					return slog.NewTextHandler(w, opts).WithAttrs([]slog.Attr{slog.String("custom", "handler")})
				}),
			},
			expectedAttrs: []string{`custom=handler`},
		},
		"colour handler": {
			opts: []log.Option{
				log.WithColourTextHandler(),
			},
			expectedAttrs: []string{"\x1b[92mINF\x1b[0m", "\x1b[2;91merr=\x1b[22m\"some error\"\x1b[0m"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			test.opts = append(test.opts, log.WithWriter(&buf))
			logger := log.New(test.opts...)

			logger.Info("test message", slog.Any("err", errors.New("some error")))
			output := buf.String()
			for _, attr := range test.expectedAttrs {
				require.Contains(t, output, attr)
			}

			assertIsVerbose(t, logger, &buf, test.expectedVerbose)
			assertIsJSONHandler(t, logger, test.expectedJSON)
		})
	}
}

func assertIsVerbose(t *testing.T, logger *slog.Logger, buf *bytes.Buffer, expectedVerbose bool) {
	t.Helper()

	logger.Debug("debug message")
	output := buf.String()

	if expectedVerbose {
		require.Contains(t, output, "debug message")
	} else {
		require.NotContains(t, output, "debug message")
	}
}

func assertIsJSONHandler(t *testing.T, logger *slog.Logger, expectedJSON bool) {
	t.Helper()

	handler := logger.Handler()
	isJSONHandler := false
	switch handler.(type) {
	case *slog.JSONHandler:
		isJSONHandler = true
	default:
		isJSONHandler = false
	}

	require.Equal(t, expectedJSON, isJSONHandler)
}

func TestReplaceSourceAttr(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		attr     slog.Attr
		expected slog.Attr
	}{
		"non-source attribute": {
			attr:     slog.String("other", "value"),
			expected: slog.String("other", "value"),
		},
		"source attribute with invalid value type": {
			attr:     slog.Attr{Key: slog.SourceKey, Value: slog.StringValue("invalid")},
			expected: slog.Attr{Key: slog.SourceKey, Value: slog.StringValue("invalid")},
		},
		"valid source attribute": {
			attr: slog.Attr{
				Key: slog.SourceKey,
				Value: slog.AnyValue(&slog.Source{
					Function: "TestFunc",
					File:     "/path/to/file.go",
					Line:     42,
				}),
			},
			expected: slog.Attr{
				Key:   slog.SourceKey,
				Value: slog.StringValue("file.go:42"),
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := log.ReplaceSourceAttr(nil, test.attr)

			require.Equal(t, test.expected, result)
		})
	}
}
