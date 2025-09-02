package log

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/lmittmann/tint"
)

type params struct {
	verbose    bool
	jsonFormat bool
	colour     bool
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

// WithJSONHandler enables JSON formatting for logs
func WithJSONHandler() Option {
	return func(params *params) {
		params.jsonFormat = true
	}
}

// WithTextHandler enables JSON formatting for logs
func WithTextHandler(colourOutput bool) Option {
	return func(params *params) {
		params.jsonFormat = false
		params.colour = colourOutput
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

	var handler slog.Handler = slog.NewTextHandler(params.writer, &slog.HandlerOptions{
		Level:       level,
		AddSource:   true,
		ReplaceAttr: replaceSourceAttr,
	})
	if params.jsonFormat {
		handler = slog.NewJSONHandler(params.writer, &slog.HandlerOptions{
			Level:       level,
			AddSource:   true,
			ReplaceAttr: replaceSourceAttr,
		})
	} else if params.colour {
		handler = tint.NewHandler(params.writer, &tint.Options{
			Level:       level,
			AddSource:   true,
			ReplaceAttr: replaceSourceAttrWithColour,
		})
	}

	attrs := []slog.Attr{}
	attrs = append(attrs, params.attrs...)
	return slog.New(handler.WithAttrs(attrs))
}

func replaceSourceAttrWithColour(groups []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindAny {
		if _, ok := a.Value.Any().(error); ok {
			return tint.Attr(9, a)
		}
	}

	if a.Key == "bank" && a.Value.Kind() == slog.KindString {
		return tint.Attr(11, a)
	}

	return replaceSourceAttr(groups, a)
}

func replaceSourceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindAny {
		if _, ok := a.Value.Any().(error); ok {
			return tint.Attr(9, a)
		}
	}

	if a.Key == "bank" && a.Value.Kind() == slog.KindString {
		return tint.Attr(11, a)
	}

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
