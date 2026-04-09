package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/log"
	"github.com/spf13/cobra"
)

type exportAccountsOptions struct {
	AuthToken string
	Timeout   time.Duration
}

func newAccountsCommand(exporterType export.ExportType) *cobra.Command {
	opts := &exportAccountsOptions{}
	name := string(exporterType)
	upperName := strings.ToUpper(name)
	lowerName := strings.ToLower(name)

	cmd := &cobra.Command{
		Use:   "accounts",
		Short: fmt.Sprintf("List accounts from %s", name),
		Long:  fmt.Sprintf("Fetch and display all available %s account IDs for the authenticated user", name),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAccountsCommand(cmd.Context(), cmd.OutOrStdout(), opts, exporterType)
		},
		Example: fmt.Sprintf(cmdExample,
			fmt.Sprintf("fingrab %s accounts --token <api-token>", lowerName),
			upperName,
			fmt.Sprintf("fingrab %s accounts", lowerName),
			upperName, upperName,
			fmt.Sprintf("fingrab %s accounts", lowerName),
		),
	}

	cmd.Flags().StringVar(&opts.AuthToken, "token", "", "API auth token")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", timeout, "API request timeout")

	return cmd
}

func runAccountsCommand(ctx context.Context, output io.Writer, opts *exportAccountsOptions, exportType export.ExportType) error {
	logger := log.FromContext(ctx).With(
		slog.String("bank", string(exportType)),
	)
	ctx = log.WithContext(ctx, logger)

	authToken, err := getAuthToken(ctx, exportType, opts.AuthToken)
	if err != nil {
		return fmt.Errorf("%s: authentication failed: %w", strings.ToLower(string(exportType)), err)
	}

	exportOpts := export.AccountOptions{
		Options: export.Options{
			AuthToken: authToken,
			Timeout:   opts.Timeout,
		},
	}

	accounts, err := export.Accounts(ctx, exportType, exportOpts)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	for _, account := range accounts {
		_, _ = fmt.Fprintln(output, account.ID)
	}

	return nil
}
