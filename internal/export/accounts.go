package export

import (
	"context"
	"fmt"

	"github.com/HallyG/fingrab/internal/domain"
)

type AccountOptions struct {
	Options
}

func (o AccountOptions) Validate(ctx context.Context) error {
	if err := o.Options.Validate(ctx); err != nil {
		return err
	}

	return nil
}

func Accounts(ctx context.Context, exportType ExportType, opts AccountOptions) ([]*domain.Account, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	exporter, err := NewExporter(exportType, opts.Options)
	if err != nil {
		return nil, fmt.Errorf("exporter: %w", err)
	}

	accounts, err := exporter.ExportAccounts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("transctions: %w", err)
	}

	return accounts, nil
}
