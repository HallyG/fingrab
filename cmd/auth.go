package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/log"
	monzoexporter "github.com/HallyG/fingrab/internal/monzo/exporter"
	"github.com/HallyG/fingrab/internal/oauth"
	starlingexporter "github.com/HallyG/fingrab/internal/starling/exporter"
)

const (
	envTokenSuffix        = "_TOKEN"
	envClientIDSuffix     = "_CLIENT_ID"
	envClientSecretSuffix = "_CLIENT_SECRET"
)

var bankConfigs = map[export.ExportType]oauth.Config{
	monzoexporter.ExportTypeMonzo: {
		AuthURL:              "https://auth.monzo.com",
		TokenURL:             "https://api.monzo.com/oauth2/token",
		WaitForApprovalInApp: true,
	},
	starlingexporter.ExportTypeStarling: {
		AuthURL:  "https://oauth.starlingbank.com/oauth/authorize",
		TokenURL: "https://api.starlingbank.com/oauth2/token",
	},
}

func getAuthToken(ctx context.Context, opts *exportOptions, exportType export.ExportType) (string, error) {
	logger := log.FromContext(ctx)

	// Try token from CLI flag first
	if opts.AuthToken != "" {
		logger.DebugContext(ctx, "using auth token from cli flag")
		return opts.AuthToken, nil
	}

	// Try token from environment variable
	envVar := getEnvVarName(exportType, envTokenSuffix)
	authToken := os.Getenv(envVar)
	if authToken != "" {
		logger.DebugContext(ctx, "using auth token from environment variable", "env_var", envVar)
		return authToken, nil
	}

	logger.DebugContext(ctx, "no auth token found, starting OAuth flow")
	return startOAuth(ctx, exportType)
}

func startOAuth(ctx context.Context, exportType export.ExportType) (string, error) {
	logger := log.FromContext(ctx)

	config, exists := bankConfigs[exportType]
	if !exists {
		supportedTypes := export.All()
		return "", fmt.Errorf("unsupported bank type: %s (supported types: %v)", exportType, supportedTypes)
	}

	// Try oauth client credentials from environment variables
	clientIDEnvVar := getEnvVarName(exportType, envClientIDSuffix)
	clientSecretEnvVar := getEnvVarName(exportType, envClientSecretSuffix)

	if clientID := strings.TrimSpace(os.Getenv(clientIDEnvVar)); clientID != "" {
		logger.DebugContext(ctx, "using client ID from environment variable", slog.String("env.var", clientIDEnvVar))
		config.ClientID = clientID
	}

	if clientSecret := strings.TrimSpace(os.Getenv(clientSecretEnvVar)); clientSecret != "" {
		logger.DebugContext(ctx, "using client secret from environment variable", slog.String("env.var", clientSecretEnvVar))
		config.ClientSecret = clientSecret
	}

	logger.DebugContext(ctx, "starting OAuth2 flow", "bank", exportType)
	return oauth.Exchange(ctx, &config, os.Stdin)
}

func getEnvVarName(exportType export.ExportType, suffix string) string {
	return strings.ToUpper(string(exportType)) + suffix
}
