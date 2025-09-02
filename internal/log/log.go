package log

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
)

type params struct {
	verbose    bool
	jsonFormat bool
	attrs      []slog.Attr
	writer     io.Writer
}

type Option func(params *params)

// WithVerbose enables verbose logging (sets log level to Debug).
func WithVerbose(verbose bool) Option {
	return func(params *params) {
		params.verbose = verbose
	}
}

// WithJSONFormat enables JSON formatting for logs
func WithJSONFormat(json bool) Option {
	return func(params *params) {
		params.jsonFormat = json
	}
}

// WithAttrs adds custom attributes to all log entries.
// Attributes are key-value pairs that provide additional context.
func WithAttrs(attrs ...slog.Attr) Option {
	return func(params *params) {
		params.attrs = append(params.attrs, attrs...)
	}
}

// WithWriter sets the output writer for logs.
// If w is nil, io.Discard is used
func WithWriter(w io.Writer) Option {
	return func(params *params) {
		params.writer = w
	}
}

// New creates a new slog.Logger.
// By default, logs are formatted as text, written to io.Discard, and use Info level.
//
// Example:
//
//	logger, err := log.New(
//	    log.WithWriter(os.Stdout),
//	    log.WithJSONFormat(true),
//	    log.WithVerbose(true),
//	    log.WithAttrs(slog.String("service", "api")),
//	)
func New(opts ...Option) *slog.Logger {
	var params params
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(&params)
	}

	if params.writer == nil {
		return slog.New(slog.DiscardHandler)
	}

	level := slog.LevelInfo
	if params.verbose {
		level = slog.LevelDebug
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       level,
		AddSource:   true,
		ReplaceAttr: replaceSourceAttr,
	}

	var handler slog.Handler
	if params.jsonFormat {
		handler = slog.NewJSONHandler(params.writer, handlerOpts)
	} else {
		handler = slog.NewTextHandler(params.writer, handlerOpts)
	}

	attrs := []slog.Attr{}
	attrs = append(attrs, params.attrs...)
	return slog.New(handler.WithAttrs(attrs))
}

func replaceSourceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key != slog.SourceKey {
		return a
	}

	source, ok := a.Value.Any().(*slog.Source)
	if !ok {
		return a
	}

	fileAndLine := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
	return slog.Attr{
		Key:   slog.SourceKey,
		Value: slog.StringValue(fileAndLine),
	}
}
