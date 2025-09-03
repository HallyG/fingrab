package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/log"
	monzoexporter "github.com/HallyG/fingrab/internal/monzo/exporter"
	"github.com/HallyG/fingrab/internal/oauth"
	starlingexporter "github.com/HallyG/fingrab/internal/starling/exporter"
)

var bankConfigs = map[export.ExportType]oauth.Config{
	monzoexporter.ExportTypeMonzo: {
		ClientID:             "dummy_monzo_client_id",
		ClientSecret:         "",
		AuthURL:              "https://auth.monzo.com",
		TokenURL:             "https://api.monzo.com/oauth2/token",
		WaitForApprovalInApp: true,
	},
	starlingexporter.ExportTypeStarling: {
		ClientID:     "dummy_starling_client_id",
		ClientSecret: "",
		AuthURL:      "https://oauth.starlingbank.com/oauth/authorize",
		TokenURL:     "https://api.starlingbank.com/oauth2/token",
	},
}

func getAuthToken(ctx context.Context, opts *exportOptions, exportType export.ExportType) (string, error) {
	// Get token from cli flag
	if opts.AuthToken != "" {
		return opts.AuthToken, nil
	}

	envVar := strings.ToUpper(string(exportType)) + "_TOKEN"
	authToken := os.Getenv(envVar)
	if authToken != "" {
		return authToken, nil
	}

	// clientID
	// clientSecret

	log.FromContext(ctx).WarnContext(ctx, "no valid authentication token found, starting OAuth flow")
	return startOAuth(ctx, exportType)
}

func startOAuth(ctx context.Context, exportType export.ExportType) (string, error) {
	config, exists := bankConfigs[exportType]
	if !exists {
		return "", fmt.Errorf("unsupported bank type: %s", exportType)
	}

	return oauth.Exchange(ctx, &config, os.Stdin)
}
