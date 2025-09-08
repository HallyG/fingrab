package log

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/lmittmann/tint"
)

type params struct {
	verbose   bool
	attrs     []slog.Attr
	writer    io.Writer
	handlerFn func(w io.Writer, opts *slog.HandlerOptions) slog.Handler
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
		params.handlerFn = func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewJSONHandler(params.writer, &slog.HandlerOptions{
				Level:       opts.Level,
				AddSource:   opts.AddSource,
				ReplaceAttr: opts.ReplaceAttr,
			})
		}
	}
}

// WithTextHandler enables text formatting for logs
func WithTextHandler() Option {
	return func(params *params) {
		params.handlerFn = func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewTextHandler(params.writer, &slog.HandlerOptions{
				Level:       opts.Level,
				AddSource:   opts.AddSource,
				ReplaceAttr: opts.ReplaceAttr,
			})
		}
	}
}

func WithColourTextHandler() Option {
	return func(params *params) {
		params.handlerFn = func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return tint.NewHandler(w, &tint.Options{
				Level:     opts.Level,
				AddSource: opts.AddSource,
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Value.Kind() == slog.KindAny {
						if _, ok := a.Value.Any().(error); ok {
							return tint.Attr(9, a)
						}
					}

					return opts.ReplaceAttr(groups, a)
				},
			})
		}
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

func WithHandler(fn func(w io.Writer, opts *slog.HandlerOptions) slog.Handler) Option {
	return func(params *params) {
		params.handlerFn = fn
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
		ReplaceAttr: ReplaceSourceAttr,
	}
	var handler slog.Handler = slog.NewTextHandler(params.writer, handlerOpts)
	if params.handlerFn != nil {
		handler = params.handlerFn(params.writer, handlerOpts)
	}

	attrs := []slog.Attr{}
	attrs = append(attrs, params.attrs...)
	return slog.New(handler.WithAttrs(attrs))
}

func ReplaceSourceAttr(groups []string, a slog.Attr) slog.Attr {
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
