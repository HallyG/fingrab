package oauth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/HallyG/fingrab/internal/log"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/int128/oauth2cli"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	ClientID             string
	ClientSecret         string
	AuthURL              string
	TokenURL             string
	WaitForApprovalInApp bool // Wait for user input after token exchange (useful in the scenarios where additional approval is needed in a mobile app)
	Scopes               []string
}

func (c *Config) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.ClientID, validation.Required.Error("client id is required")),
		validation.Field(&c.ClientSecret, validation.Required.Error("client secret is required")),
		validation.Field(&c.AuthURL, validation.Required.Error("auth url is required")),
		validation.Field(&c.TokenURL, validation.Required.Error("token url is required")),
	)
}

func (c *Config) ToOAuth2Config() oauth2.Config {
	return oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.AuthURL,
			TokenURL: c.TokenURL,
		},
		Scopes: c.Scopes,
	}
}

func Exchange(ctx context.Context, cfg *Config, userInput io.Reader) (string, error) {
	if err := cfg.Validate(ctx); err != nil {
		return "", fmt.Errorf("invalid oauth2 config: %w", err)
	}

	ready := make(chan string, 1)
	defer close(ready)

	token, err := exchangeToken(ctx, ready, &oauth2cli.Config{
		OAuth2Config:           cfg.ToOAuth2Config(),
		LocalServerReadyChan:   ready,
		LocalServerBindAddress: []string{"localhost:64131"},
		Logf: func(format string, args ...any) {
			log.FromContext(ctx).DebugContext(ctx, fmt.Sprintf(format, args...))
		},
	})
	if err != nil {
		return "", err
	}

	log.FromContext(ctx).DebugContext(ctx, "exchanged oauth token")
	if cfg.WaitForApprovalInApp {
		if err := WaitForApprovalInApp(ctx, userInput); err != nil {
			return "", fmt.Errorf("failed waiting for app approval: %w", err)
		}
	}

	return token.AccessToken, nil
}

func exchangeToken(ctx context.Context, ready chan string, cfg *oauth2cli.Config) (*oauth2.Token, error) {
	var token *oauth2.Token

	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return LoginWithBrowser(ctx, ready, browser.OpenURL)
	})

	errg.Go(func() error {
		oauthToken, err := oauth2cli.GetToken(ctx, *cfg)
		if err != nil {
			return fmt.Errorf("could not get oauth token: %w", err)
		}
		token = oauthToken
		return nil
	})

	if err := errg.Wait(); err != nil {
		return nil, fmt.Errorf("authorization error: %s", err)
	}

	return token, nil
}

func LoginWithBrowser(ctx context.Context, ready <-chan string, browserOpenUrlFn func(string) error) error {
	select {
	case loginURL := <-ready:
		log.FromContext(ctx).InfoContext(ctx, "you will be redirected to your web browser to complete the login process")
		log.FromContext(ctx).InfoContext(ctx, fmt.Sprintf("if the page did not open automatically, open this URL manually: %s", loginURL))

		if err := browserOpenUrlFn(loginURL); err != nil {
			log.FromContext(ctx).WarnContext(ctx, "could not open browser", slog.Any("err", err))
		}

		return nil
	case <-ctx.Done():
		return fmt.Errorf("context done while waiting for auth: %w", ctx.Err())
	}
}

func WaitForApprovalInApp(ctx context.Context, input io.Reader) error {
	logger := log.FromContext(ctx)
	logger.InfoContext(ctx, "please open your app and approve this application to access your account")
	logger.InfoContext(ctx, "press Enter once you have approved the application...")

	done := make(chan error, 1)
	go func() {
		defer close(done)
		reader := bufio.NewReader(input)
		_, err := reader.ReadString('\n')
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for user approval: %w", ctx.Err())
	}
}
