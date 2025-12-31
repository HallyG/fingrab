package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/format"
	"github.com/HallyG/fingrab/internal/log"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

const (
	timeFormat = "2006-01-02"
	timeout    = 5 * time.Second
)

type exportTransactionOptions struct {
	StartDate string
	EndDate   string
	AuthToken string
	Timeout   time.Duration
	AccountID string
	Format    string
}

func newTransactionsCommand(exporterType export.ExportType) *cobra.Command {
	opts := &exportTransactionOptions{}
	name := string(exporterType)
	lowerName := strings.ToLower(name)
	upperName := strings.ToUpper(name)

	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "Export transactions from " + name,
		Long:  fmt.Sprintf("Export banking transactions from %s for the specified date range.", name),
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := runExportTransactions(cmd.Context(), cmd.OutOrStdout(), opts, exporterType)
			if err != nil {
				return fmt.Errorf("%s: %w", lowerName, err)
			}

			return nil
		},
		Example: fmt.Sprintf(cmdExample,
			fmt.Sprintf("fingrab %s transactions --token <api-token> --start 2025-03-01 --end 2025-03-31", lowerName),
			upperName,
			fmt.Sprintf("fingrab %s transactions --start 2025-03-01 --end 2025-03-31", lowerName),
			upperName, upperName,
			fmt.Sprintf("fingrab %s transactions --start 2025-03-01 --end 2025-03-31", lowerName),
		),
	}

	allFormats := strings.Join(lo.Map(format.All(), func(item format.FormatType, index int) string {
		return fmt.Sprintf("%v", item)
	}), ", ")

	cmd.Flags().StringVar(&opts.StartDate, "start", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.EndDate, "end", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.AuthToken, "token", "", "API auth token")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", timeout, "API request timeout")
	cmd.Flags().StringVar(&opts.AccountID, "account", "", "Account ID")
	cmd.Flags().StringVar(&opts.Format, "format", string(format.FormatTypeMoneyDance), fmt.Sprintf("Output format (options: %s,)", allFormats))

	_ = cmd.MarkFlagRequired("start")

	return cmd
}

func parseDate(str string) (time.Time, error) {
	return time.Parse(timeFormat, str)
}

func runExportTransactions(ctx context.Context, output io.Writer, opts *exportTransactionOptions, exportType export.ExportType) error {
	logger := log.FromContext(ctx).With(
		slog.String("bank", string(exportType)),
	)
	ctx = log.WithContext(ctx, logger)

	startDate, err := parseDate(opts.StartDate)
	if err != nil {
		return fmt.Errorf("start date: %w", err)
	}

	now := time.Now().Truncate(24 * time.Hour)
	endDate := now.Add(24 * time.Hour)

	if opts.EndDate != "" {
		endDate, err = parseDate(opts.EndDate)
		if err != nil {
			return fmt.Errorf("end date: %w", err)
		}
	}

	// TODO: handle the case where we generate the start date at mightnight, but now is less than that
	if endDate.Before(startDate) {
		return errors.New("end date must be after start date")
	}

	if startDate.After(now) {
		return fmt.Errorf("start date %q cannot be in the future", startDate.Format(timeFormat))
	}
	if endDate.Before(startDate) {
		return fmt.Errorf("end date %q must be after start date %q", endDate.Format(timeFormat), startDate.Format(timeFormat))
	}

	if endDate.After(now.Add(24 * time.Hour)) {
		return errors.New("end date cannot be more than 1 day in the future")
	}

	authToken, err := getAuthToken(ctx, opts, exportType)
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
		return fmt.Errorf("formatter: %w", err)
	}

	transactions, err := export.Transactions(ctx, exportType, exportOpts)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if err := format.WriteCollection(formatter, transactions); err != nil {
		return err
	}

	return nil
}
