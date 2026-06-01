package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
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
	OutputDir string
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
	cmd.Flags().StringVar(&opts.AccountID, "account", "", "Account ID (omit to export all accounts)")
	cmd.Flags().StringVar(&opts.Format, "format", string(format.FormatTypeMoneyDance), fmt.Sprintf("Output format (one of: %s)", allFormats))
	cmd.Flags().StringVar(&opts.OutputDir, "output-dir", "", fmt.Sprintf("Output each account to a separate file in this directory (e.g. %s_<account-id>.csv)", lowerName))

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

	if opts.OutputDir != "" {
		if _, statErr := os.Stat(opts.OutputDir); os.IsNotExist(statErr) {
			return fmt.Errorf("output-dir does not exist: %s", opts.OutputDir)
		}
	}

	authToken, err := getAuthToken(ctx, exportType, opts.AuthToken)
	if err != nil {
		return err
	}

	baseOpts := export.Options{
		AuthToken: authToken,
		Timeout:   opts.Timeout,
	}

	accounts, err := export.Accounts(ctx, exportType, export.AccountOptions{Options: baseOpts})
	if err != nil {
		return fmt.Errorf("accounts: %w", err)
	}

	if len(accounts) == 0 {
		return errors.New("no accounts found")
	}

	accounts = lo.Filter(accounts, func(acct *domain.Account, _ int) bool {
		if opts.AccountID != "" {
			return acct.ID == opts.AccountID
		}

		return true
	})

	if opts.OutputDir != "" {
		for _, account := range accounts {
			fileName := fmt.Sprintf("%s_%s.csv", strings.ToLower(string(exportType)), account.ID)
			filePath := filepath.Join(opts.OutputDir, fileName)

			f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer f.Close()

			if err := exportToWriter(ctx, f, opts, exportType, account.ID, startDate, endDate, baseOpts); err != nil {
				return err
			}
		}

		return nil
	}

	// Combined stdout: one header, all accounts' rows, one flush.
	formatter, err := format.NewFormatter(format.FormatType(opts.Format), output)
	if err != nil {
		return fmt.Errorf("formatter: %w", err)
	}

	if err := formatter.WriteHeader(); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, account := range accounts {
		transactions, err := export.Transactions(ctx, exportType, export.TransactionOptions{
			StartDate: startDate,
			EndDate:   endDate,
			AccountID: account.ID,
			Options:   baseOpts,
		})
		if err != nil {
			return fmt.Errorf("export account %s: %w", account.ID, err)
		}

		for _, t := range transactions {
			if err := formatter.WriteTransaction(t); err != nil {
				return fmt.Errorf("write transaction: %w", err)
			}
		}
	}

	return formatter.Flush()
}

// exportToWriter fetches transactions for one account and writes them to w using format.WriteCollection.
func exportToWriter(ctx context.Context, w io.Writer, opts *exportTransactionOptions, exportType export.ExportType, accountID string, startDate, endDate time.Time, baseOpts export.Options) error {
	formatter, err := format.NewFormatter(format.FormatType(opts.Format), w)
	if err != nil {
		return fmt.Errorf("formatter: %w", err)
	}

	transactions, err := export.Transactions(ctx, exportType, export.TransactionOptions{
		StartDate: startDate,
		EndDate:   endDate,
		AccountID: accountID,
		Options:   baseOpts,
	})
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	return format.WriteCollection(formatter, transactions)
}
