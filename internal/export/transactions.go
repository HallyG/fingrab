package export

import (
	"context"
	"fmt"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type TransactionOptions struct {
	AccountID string
	EndDate   time.Time
	StartDate time.Time
	Options
}

func (o TransactionOptions) Validate(ctx context.Context) error {
	if err := o.Options.Validate(ctx); err != nil {
		return err
	}

	return validation.ValidateStructWithContext(ctx, &o,
		validation.Field(&o.StartDate, validation.Required.Error("is required")),
		validation.Field(&o.EndDate, validation.Required.Error("is required")),
	)
}

// Transactions fetches transactions for the specified export type and options.
// It validates the options, checks the specificed date range against the exporter's maximum, and retrieves the transactions.
//
// Example:
//
//	ctx := context.Background()
//	opts := Options{AccountID: "123", StartDate: time.Now().AddDate(0, 0, -7), EndDate: time.Now(), AuthToken: "token"}
//	transactions, err := Transactions(ctx, "csv", opts)
//	if err != nil {
//	    // Handle error
//	}
func Transactions(ctx context.Context, exportType ExportType, opts TransactionOptions) ([]*domain.Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	exporter, err := NewExporter(exportType, opts.Options)
	if err != nil {
		return nil, fmt.Errorf("exporter: %w", err)
	}

	maxDateRange := exporter.MaxDateRange()
	days := (opts.EndDate.Sub(opts.StartDate).Hours()) / 24
	if maxDateRange > 0 && opts.EndDate.Sub(opts.StartDate) > maxDateRange {
		hours := maxDateRange.Hours()
		maxDays := hours / 24
		return nil, fmt.Errorf("date range %d days is too long, max is %d days", int(days), int(maxDays))
	}

	transactions, err := exporter.ExportTransactions(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("transctions: %w", err)
	}

	return transactions, nil
}
