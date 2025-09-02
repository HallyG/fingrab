package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	resty "resty.dev/v3"
)

const (
	defaultTimeout          = 1 * time.Minute
	defaultRetryCount       = 3
	defaultRetryWaitTime    = 2 * time.Second
	defaultMaxRetryWaitTime = 10 * time.Second
)

type Client interface {
	ExecuteRequest(ctx context.Context, method, url string, values url.Values, result any) (*resty.Response, error)
}

type BaseClient struct {
	resty               *resty.Client
	errorUnmarshallerFn func(r *resty.Response) error
}

type Option func(*BaseClient)

func New(baseURL string, httpClient *http.Client, opts ...Option) *BaseClient {
	c := &BaseClient{}
	c.resty = resty.NewWithClient(httpClient).
		SetBaseURL(baseURL).
		SetTimeout(defaultTimeout).
		SetRetryCount(defaultRetryCount).
		SetRetryWaitTime(defaultRetryWaitTime).
		SetRetryMaxWaitTime(defaultMaxRetryWaitTime).
		AddResponseMiddleware(func(c *resty.Client, r *resty.Response) error {
			startTime := r.Request.Time
			endTime := r.ReceivedAt()

			req := r.Request
			zerolog.Ctx(req.Context()).Debug().
				Str("http.url", req.URL).
				Str("http.method", req.Method).
				Err(r.Err).
				Dur("http.duration_ms", endTime.Sub(startTime)).
				Int("http.status_code", r.StatusCode()).
				Msg("performed HTTP request")
			return nil
		}).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			return err != nil || r.StatusCode() >= 500
		})

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(c)
	}

	return c
}

func WithAuthToken(authToken string) Option {
	return func(c *BaseClient) {
		c.resty.SetHeader("Authorization", authToken)
	}
}

func WithBaseURL(url string) Option {
	return func(c *BaseClient) {
		c.resty.SetBaseURL(url)
	}
}

func WithErrorUnmarshaller(unmarshallerFn func(r *resty.Response) error) Option {
	return func(c *BaseClient) {
		c.errorUnmarshallerFn = unmarshallerFn
	}
}

func (c *BaseClient) ExecuteRequest(ctx context.Context, method, url string, values url.Values, result any) (*resty.Response, error) {
	req := c.resty.R().
		SetContext(ctx).
		SetResult(result).
		SetUnescapeQueryParams(false).
		SetQueryParamsFromValues(values)

	resp, err := req.Execute(method, url)
	if err != nil {
		return resp, fmt.Errorf("%s %s failed: %w", method, resp.Request.URL, err)
	}

	if resp.IsError() {
		if c.errorUnmarshallerFn != nil {
			return nil, c.errorUnmarshallerFn(resp)
		}

		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return resp, nil
}
