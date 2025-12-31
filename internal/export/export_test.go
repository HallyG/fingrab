package export_test

import (
	"context"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	t.Parallel()

	t.Run("returns expected exporters", func(t *testing.T) {
		t.Parallel()

		exporters := export.All()
		require.NotNil(t, exporters)
	})
}

func TestBearerToken(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input  export.TransactionOptions
		output string
	}{
		"whitespace removed and bearer is added": {
			input: export.TransactionOptions{
				Options: export.Options{
					AuthToken: " eyJ ",
				},
			},
			output: "Bearer eyJ",
		},
		"whitespace removed and bearer is not added": {
			input: export.TransactionOptions{
				Options: export.Options{
					AuthToken: " Bearer eyJ ",
				},
			},
			output: "Bearer eyJ",
		},
		"bearer is added": {
			input: export.TransactionOptions{
				Options: export.Options{
					AuthToken: "eyJ",
				},
			},
			output: "Bearer eyJ",
		},
		"bearer is not added": {
			input: export.TransactionOptions{
				Options: export.Options{
					AuthToken: "Bearer eyJ",
				},
			},
			output: "Bearer eyJ",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := test.input.BearerAuthToken()

			require.Equal(t, test.output, output)
		})
	}
}

func TestNewExporter(t *testing.T) {
	t.Parallel()

	t.Run("returns error for unknown type", func(t *testing.T) {
		t.Parallel()

		exporter, err := export.NewExporter(export.ExportType("wow"), export.Options{})

		require.Nil(t, exporter)
		require.ErrorContains(t, err, "unsupported type: wow")
	})
}

const ExportTypeStub export.ExportType = "stubtype"

var _ export.Exporter = (*StubExporter)(nil)

type StubExporter struct {
	transactions []*domain.Transaction
	accounts     []*domain.Account
	err          error
}

func (s *StubExporter) Type() export.ExportType {
	return ExportTypeStub
}

func (s *StubExporter) MaxDateRange() time.Duration {
	return 24 * time.Hour
}

func (s *StubExporter) ExportAccounts(ctx context.Context, opts export.AccountOptions) ([]*domain.Account, error) {
	return s.accounts, s.err
}

func (s *StubExporter) ExportTransactions(ctx context.Context, opts export.TransactionOptions) ([]*domain.Transaction, error) {
	return s.transactions, s.err
}
