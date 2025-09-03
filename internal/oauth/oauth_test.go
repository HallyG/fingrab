package oauth_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/oauth"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name           string
		config         *oauth.Config
		expectedErrMsg string
	}{
		"valid config": {
			config: &oauth.Config{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
			},
		},
		"returns error when missing client ID": {
			config: &oauth.Config{
				ClientSecret: "test-secret",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
			},
			expectedErrMsg: "client id is required",
		},
		"returns error when missing client secret": {
			config: &oauth.Config{
				ClientID: "test-client",
				AuthURL:  "https://example.com/auth",
				TokenURL: "https://example.com/token",
			},
			expectedErrMsg: "client secret is required",
		},
		"returns error when missing auth URL": {
			config: &oauth.Config{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				TokenURL:     "https://example.com/token",
			},
			expectedErrMsg: "auth url is required",
		},
		"returns error when missing token URL": {
			config: &oauth.Config{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				AuthURL:      "https://example.com/auth",
			},
			expectedErrMsg: "token url is required",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := test.config.Validate(t.Context())

			if test.expectedErrMsg != "" {
				require.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigToOAuth2Config(t *testing.T) {
	t.Parallel()

	config := &oauth.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		Scopes:       []string{"read", "write"},
	}

	oauth2Config := config.ToOAuth2Config()

	require.Equal(t, "test-client", oauth2Config.ClientID)
	require.Equal(t, "test-secret", oauth2Config.ClientSecret)
	require.Equal(t, "https://example.com/auth", oauth2Config.Endpoint.AuthURL)
	require.Equal(t, "https://example.com/token", oauth2Config.Endpoint.TokenURL)
	require.Equal(t, []string{"read", "write"}, oauth2Config.Scopes)
}

func TestExchange(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) *oauth.Config {
		t.Helper()

		return &oauth.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			AuthURL:      "https://example.com/auth",
			TokenURL:     "https://example.com/token",
		}
	}

	t.Run("returns error when cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		config := setup(t)
		result, err := oauth.Exchange(ctx, config, strings.NewReader(""))

		require.Error(t, err)
		require.Empty(t, result)
		require.Contains(t, err.Error(), "context done while waiting for auth: context canceled")
	})
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestWaitForApprovalInApp(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input       io.Reader
		expectedErr string
		ctxFn       func(t *testing.T) context.Context
	}{
		"success approval": {
			input: strings.NewReader("\n"),
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)
				return ctx
			},
		},
		"returns error when context cancelled": {
			input: strings.NewReader(""),
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			expectedErr: "context cancelled while waiting for user approval: context canceled",
		},
		"returns error when reader fails": {
			input: &errorReader{err: errors.New("reader failure")},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)
				return ctx
			},
			expectedErr: "failed to read user input: reader failure",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := test.ctxFn(t)

			err := oauth.WaitForApprovalInApp(ctx, test.input)

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type stubBrowserFn struct {
	err         error
	capturedURL string
	callCount   int
}

func (s *stubBrowserFn) openURL(url string) error {
	s.capturedURL = url
	s.callCount++

	return s.err
}

func TestLoginWithBrowser(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		readyChannelFn func(t *testing.T) <-chan string
		browserFn      func(t *testing.T) *stubBrowserFn
		ctxFn          func(t *testing.T) context.Context
		expectedErr    string
		expectedURL    string
		expectedCalls  int
	}{
		"success with valid URL": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				ready := make(chan string, 1)
				ready <- "https://example.com/auth?client_id=test"
				close(ready)

				return ready
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()

				return &stubBrowserFn{}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)

				return ctx
			},
			expectedURL:   "https://example.com/auth?client_id=test",
			expectedCalls: 1,
		},
		"success with browser open failure": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				ready := make(chan string, 1)
				ready <- "https://example.com/auth"
				close(ready)

				return ready
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()

				return &stubBrowserFn{
					err: errors.New("browser failed to open"),
				}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)

				return ctx
			},
			expectedURL:   "https://example.com/auth",
			expectedCalls: 1,
		},
		"success with empty URL": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				ready := make(chan string, 1)
				ready <- ""
				close(ready)

				return ready
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()

				return &stubBrowserFn{}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)

				return ctx
			},
			expectedCalls: 1,
		},
		"returns error when context cancelled immediately": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				return make(chan string)
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()

				return &stubBrowserFn{}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				return ctx
			},
			expectedErr: "context done while waiting for auth: context canceled",
		},
		"returns error when context times out": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				return make(chan string)
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()

				return &stubBrowserFn{}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
				t.Cleanup(cancel)

				return ctx
			},
			expectedErr: "context done while waiting for auth: context deadline exceeded",
		},
		"success when URL arrives before context cancellation": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				ready := make(chan string, 1)
				// Send URL immediately so it's available before context cancellation
				ready <- "https://example.com/auth"

				close(ready)
				return ready
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()
				return &stubBrowserFn{}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				// Cancel after a short delay to simulate race condition
				go func() {
					time.Sleep(5 * time.Millisecond)
					cancel()
				}()

				return ctx
			},
			expectedURL:   "https://example.com/auth",
			expectedCalls: 1,
		},
		"success when ready channel is closed without sending": {
			readyChannelFn: func(t *testing.T) <-chan string {
				t.Helper()

				ready := make(chan string)
				close(ready)

				return ready
			},
			browserFn: func(t *testing.T) *stubBrowserFn {
				t.Helper()

				return &stubBrowserFn{}
			},
			ctxFn: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)

				return ctx
			},
			expectedCalls: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := test.ctxFn(t)
			ready := test.readyChannelFn(t)
			browserFn := test.browserFn(t)

			err := oauth.LoginWithBrowser(ctx, ready, browserFn.openURL)

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expectedCalls, browserFn.callCount)
			if test.expectedCalls > 0 {
				require.Equal(t, test.expectedURL, browserFn.capturedURL)
			}
		})
	}
}
