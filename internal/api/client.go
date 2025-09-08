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
	authorizationHeader     = "Authorization"
)

type Option func(*resty.Client)

// WithAuthToken configures the client to include an Authorization header with the specified token.
func WithAuthToken(authToken string) Option {
	return func(c *resty.Client) {
		c.SetHeader(authorizationHeader, authToken)
	}
}

// WithAuthToken configures the client to use the specified URL as the base URL for all requests.
func WithBaseURL(url string) Option {
	return func(c *resty.Client) {
		c.SetBaseURL(url)
	}
}

// WithError configures the client to unmarshal error responses into a specific error type.
// Example:
//
//	type APIError struct { Message string }
//	client := New("https://api.example.com", &http.Client{}, WithError[APIError]())
func WithError[E error]() Option {
	return func(c *resty.Client) {
		var err E
		c.SetError(&err)
	}
}

// New creates a new resty.Client with a pre-configured base URL, timeout logic, retry logic, and a logging middleware.
// Additional options can be provided to further customize the client.
//
// Example:
//
//	client := New("https://api.example.com", &http.Client{}, WithAuthToken("Bearer token"))
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

// ExecuteRequest performs an HTTP request with the specified method, URL, and query parameters, and unmarshals the response into the provided type T.
// Returns an error if the request fails or the if the response indicates an error (4xx or 5xx status code).
// Example:
//
//	type User struct { ID string; Name string }
//	user, err := ExecuteRequest[User](ctx, client, "GET", "/users/123", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("User: %+v\n", user)
func ExecuteRequest[T any](ctx context.Context, client *resty.Client, method, url string, values url.Values) (*T, error) {
	var result T

	resp, err := client.R().
		SetContext(ctx).
		SetResult(&result).
		SetUnescapeQueryParams(false).
		SetQueryParamsFromValues(values).
		Execute(method, url)
	if err != nil {
		return nil, fmt.Errorf("execute %s %s: %w", method, resp.Request.URL, err)
	}

	if resp.IsError() {
		if err, ok := resp.Error().(error); ok {
			return nil, err
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}
