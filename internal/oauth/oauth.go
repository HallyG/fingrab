package oauth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/HallyG/fingrab/internal/log"
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
	WaitForApprovalInApp bool // waits for user input before returning token, useful in the scenario where we need to approve access in the app
	Scopes               []string
}

func Exchange(ctx context.Context, cfg *Config) (string, error) {
	ready := make(chan string, 1)
	defer close(ready)

	token, err := ExchangeToken(ctx, ready, &oauth2cli.Config{
		OAuth2Config: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  cfg.AuthURL,
				TokenURL: cfg.TokenURL,
			},
			Scopes: cfg.Scopes,
		},
		LocalServerReadyChan:   ready,
		LocalServerBindAddress: []string{"localhost:64131"},
		Logf: func(format string, args ...interface{}) {
			log.FromContext(ctx).DebugContext(ctx, fmt.Sprintf(format, args...))
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to exchange oauth token: %w", err)
	}

	log.FromContext(ctx).InfoContext(ctx, "exchanged oauth token")
	if cfg.WaitForApprovalInApp {
		if err := WaitForApprovalInApp(ctx); err != nil {
			return "", fmt.Errorf("failed waiting for app approval: %w", err)
		}
	}

	return token.AccessToken, nil
}

func ExchangeToken(ctx context.Context, ready chan string, cfg *oauth2cli.Config) (*oauth2.Token, error) {
	var token *oauth2.Token

	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return LoginWithBrowser(ctx, ready)
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

func LoginWithBrowser(ctx context.Context, ready <-chan string) error {
	select {
	case loginURL := <-ready:
		_, _ = fmt.Println("You will be redirected to your web browser to complete the login process")
		_, _ = fmt.Println("If the page did not open automatically, open this URL manually:", loginURL)
		if err := browser.OpenURL(loginURL); err != nil {
			log.FromContext(ctx).WarnContext(ctx, "could not open browser", slog.Any("err", err))
		}

		return nil
	case <-ctx.Done():
		return fmt.Errorf("context done while waiting for auth: %w", ctx.Err())
	}
}

func WaitForApprovalInApp(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.InfoContext(ctx, "please open your app and approve this application to access your account")
	logger.InfoContext(ctx, "press Enter once you have approved the app in your app...")

	done := make(chan struct{})
	go func() {
		defer close(done)
		var input string
		fmt.Scanln(&input)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for user approval: %w", ctx.Err())
	}
}
