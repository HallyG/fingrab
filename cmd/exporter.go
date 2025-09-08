package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/format"
	"github.com/HallyG/fingrab/internal/log"
	"github.com/HallyG/fingrab/internal/util/sliceutil"
	"github.com/spf13/cobra"
)

const (
	timeFormat = "2006-01-02"
	timeout    = 5 * time.Second
	day        = 24 * time.Hour
)

var (
	exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export transactions between two dates",
		Long:  "Export banking transactions for the specified date range from supported providers",
	}
)

type exportOptions struct {
	StartDateStr string
	EndDateStr   string
	AuthToken    string
	Timeout      time.Duration
	AccountID    string
	Format       string
}

func newExportCommand(exporterType export.ExportType) *cobra.Command {
	opts := &exportOptions{}
	name := string(exporterType)

	cmd := &cobra.Command{
		Use:   strings.ToLower(name),
		Short: "Export transactions from " + name,
		Long:  `Export banking transactions for the specified date range from supported providers.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := runExport(cmd.Context(), cmd.OutOrStdout(), opts, exporterType)
			if err != nil {
				return fmt.Errorf("export transactions: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.StartDateStr, "start", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.EndDateStr, "end", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.AuthToken, "token", "", fmt.Sprintf("API authentication token (can also be set via %s_TOKEN environment variable)", strings.ToUpper(name)))
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", timeout, "API request timeout")
	cmd.Flags().StringVar(&opts.AccountID, "account", "", "Account ID")
	cmd.Flags().StringVar(&opts.Format, "format", string(format.FormatTypeMoneyDance), fmt.Sprintf("Output format (options: %s,)", sliceutil.ToDelimitedString(format.All())))

	_ = cmd.MarkFlagRequired("start")

	return cmd
}

func parseDate(str string) (time.Time, error) {
	return time.Parse(timeFormat, str)
}

func getAuthToken(opts *exportOptions, exportType export.ExportType) (string, error) {
	if opts.AuthToken != "" {
		return opts.AuthToken, nil
	}
	// Get token from environment variable if not provided via flag
	envVar := strings.ToUpper(string(exportType)) + "_TOKEN"
	authToken := os.Getenv(envVar)
	if authToken == "" {
		return "", fmt.Errorf("authentication token is required. Please provide it via --token flag or %s environment variable", envVar)
	}

	return opts.AuthToken, nil
}

func runExport(ctx context.Context, output io.Writer, opts *exportOptions, exportType export.ExportType) error {
	logger := log.FromContext(ctx).With(
		slog.String("bank", string(exportType)),
	)
	ctx = log.WithContext(ctx, logger)

	startDate, err := parseDate(opts.StartDateStr)
	if err != nil {
		return fmt.Errorf("start date: %w", err)
	}

	now := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	endDate := now.Add(24 * time.Hour)

	if opts.EndDateStr != "" {
		endDate, err = parseDate(opts.EndDateStr)
		if err != nil {
			return fmt.Errorf("end date: %w", err)
		}
	}

	// TODO: handle the case where we generate the start date at mightnight, but now is less than that
	if startDate.After(now) {
		return fmt.Errorf("start date %q cannot be in the future", startDate.Format(timeFormat))
	}
	if endDate.Before(startDate) {
		return fmt.Errorf("end date %q must be after start date %q", endDate.Format(timeFormat), startDate.Format(timeFormat))
	}

	authToken, err := getAuthToken(opts, exportType)
	if err != nil {
		return err
	}

	exportOpts := export.Options{
		StartDate: startDate,
		EndDate:   endDate,
		AccountID: opts.AccountID,
		AuthToken: authToken,
		Timeout:   opts.Timeout,
	}

	formatter, err := format.NewFormatter(format.FormatType(opts.Format), output)
	if err != nil {
		return fmt.Errorf("create formatter: %w", err)
	}

	transactions, err := export.Transactions(ctx, export.ExportType("hello world"), exportOpts)
	if err != nil {
		return fmt.Errorf("%v transactions: %w", strings.ToLower(string(exportType)), err)
	}

	if err := format.WriteCollection(formatter, transactions); err != nil {
		return err
	}

	return nil
}
