package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/HallyG/fingrab/internal/log"
	resty "resty.dev/v3"
)

const (
	defaultTimeout          = 1 * time.Minute
	defaultRetryCount       = 3
	defaultRetryWaitTime    = 2 * time.Second
	defaultMaxRetryWaitTime = 10 * time.Second
)

type Option func(*resty.Client)

func New(baseURL string, httpClient *http.Client, opts ...Option) *resty.Client {
	client := resty.NewWithClient(httpClient).
		SetBaseURL(baseURL).
		SetTimeout(defaultTimeout).
		SetRetryCount(defaultRetryCount).
		SetRetryWaitTime(defaultRetryWaitTime).
		SetRetryMaxWaitTime(defaultMaxRetryWaitTime).
		AddResponseMiddleware(func(client *resty.Client, r *resty.Response) error {
			startTime := r.Request.Time
			endTime := r.ReceivedAt()

			req := r.Request
			ctx := r.Request.Context()

			log.FromContext(ctx).DebugContext(ctx, "performed HTTP request", slog.Group("http",
				slog.Any("err", r.Err),
				slog.Duration("duration_ms", endTime.Sub(startTime)),
				slog.Int("status_code", r.StatusCode()),
				slog.String("url", req.URL),
				slog.String("method", req.Method),
			))
			return nil
		}).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			retry := err != nil || r.StatusCode() >= 500
			return retry
		})

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(client)
	}

	return client
}

func WithAuthToken(authToken string) Option {
	return func(c *resty.Client) {
		c.SetHeader("Authorization", authToken)
	}
}

func WithBaseURL(url string) Option {
	return func(c *resty.Client) {
		c.SetBaseURL(url)
	}
}

func WithError[E error]() Option {
	return func(c *resty.Client) {
		var err E
		c.SetError(&err)
	}
}

func ExecuteRequest[T any](ctx context.Context, client *resty.Client, method, url string, values url.Values) (*T, error) {
	var result T

	resp, err := client.R().
		SetContext(ctx).
		SetResult(&result).
		SetUnescapeQueryParams(false).
		SetQueryParamsFromValues(values).
		Execute(method, url)
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w", method, resp.Request.URL, err)
	}

	if resp.IsError() {
		if err, ok := resp.Error().(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}
