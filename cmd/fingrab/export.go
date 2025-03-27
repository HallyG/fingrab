package fingrab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/format"
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
			return runExport(cmd.Context(), opts, exporterType)
		},
	}

	cmd.Flags().StringVar(&opts.StartDateStr, "start", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.EndDateStr, "end", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.AuthToken, "token", "", fmt.Sprintf("API authentication token (can also be set via %s_TOKEN environment variable)", strings.ToUpper(name)))
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", timeout, "API request timeout")
	cmd.Flags().StringVar(&opts.AccountID, "account", "", "Account ID")
	cmd.Flags().StringVar(&opts.Format, "format", string(format.FormatTypeMoneyDance), fmt.Sprintf("Output format (options: %s,)", formatOptions()))

	_ = cmd.MarkFlagRequired("start")

	return cmd
}

func runExport(ctx context.Context, opts *exportOptions, exporterType export.ExportType) error {
	startDate, err := parseDate(opts.StartDateStr)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	endDate := time.Now()
	if opts.EndDateStr != "" {
		endDate, err = parseDate(opts.EndDateStr)
		if err != nil {
			return fmt.Errorf("invalid end date: %w", err)
		}
	}

	if endDate.Before(startDate) {
		return errors.New("end date must be after start date")
	}

	if startDate.After(time.Now()) {
		return errors.New("start date cannot be in the future")
	}

	// Get token from environment variable if not provided via flag
	if opts.AuthToken == "" {
		envVar := strings.ToUpper(string(exporterType)) + "_TOKEN"
		opts.AuthToken = os.Getenv(envVar)

		if opts.AuthToken == "" {
			return fmt.Errorf("authentication token is required. Please provide it via --token flag or %s environment variable", envVar)
		}
	}

	exportOpts := export.Options{
		StartDate: startDate,
		EndDate:   endDate,
		AccountID: opts.AccountID,
		AuthToken: opts.AuthToken,
		Timeout:   opts.Timeout,
		Format:    format.FormatType(opts.Format),
	}

	formatter, err := format.NewFormatter(exportOpts.Format, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	return export.Transactions(ctx, exporterType, exportOpts, formatter)
}

func parseDate(str string) (time.Time, error) {
	return time.Parse(timeFormat, str)
}

func formatOptions() string {
	types := format.All()
	strTypes := make([]string, len(types))

	for i, t := range types {
		strTypes[i] = string(t)
	}

	return strings.Join(strTypes, ", ")
}
