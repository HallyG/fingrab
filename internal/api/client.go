package api

import (
	"context"
	"fmt"
	"net/http"
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
	ExecuteRequest(ctx context.Context, method, url string, params map[string]string, result any) (*resty.Response, error)
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

func (c *BaseClient) ExecuteRequest(ctx context.Context, method, url string, params map[string]string, result any) (*resty.Response, error) {
	req := c.resty.R().
		SetContext(ctx).
		SetResult(result).
		SetUnescapeQueryParams(false)

	for key, value := range params {
		req.SetQueryParam(key, value)
	}

	startTime := time.Now()
	resp, err := req.Execute(method, url)
	endTime := time.Since(startTime)

	event := zerolog.Ctx(ctx).Debug().
		Str("http.url", req.URL).
		Str("http.method", req.Method).
		Err(err).
		Dur("http.duration_ms", time.Duration(endTime.Milliseconds()))

	if resp != nil {
		event.Int("http.status_code", resp.StatusCode())
	}

	event.Msg("performed HTTP request")

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
