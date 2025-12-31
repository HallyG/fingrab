package export_test

import (
	"context"
	"errors"
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
		input  export.Options
		output string
	}{
		"whitespace removed and bearer is added": {
			input: export.Options{
				AuthToken: " eyJ ",
			},
			output: "Bearer eyJ",
		},
		"whitespace removed and bearer is not added": {
			input: export.Options{
				AuthToken: " Bearer eyJ ",
			},
			output: "Bearer eyJ",
		},
		"bearer is added": {
			input: export.Options{
				AuthToken: "eyJ",
			},
			output: "Bearer eyJ",
		},
		"bearer is not added": {
			input: export.Options{
				AuthToken: "Bearer eyJ",
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

func TestTransactions(t *testing.T) {
	t.Parallel()

	export.Register(ExportTypeStub, func(opts export.Options) (export.Exporter, error) {
		if opts.AuthToken == "12345" {
			return nil, errors.New("invalid auth token")
		}

		return &StubExporter{
			transactions: []*domain.Transaction{
				{},
			},
		}, nil
	})

	tests := map[string]struct {
		opts                    export.Options
		expectedErrMsg          string
		expectedTransactionsLen int
	}{
		"success": {
			opts: export.Options{
				EndDate:   time.Now(),
				StartDate: time.Now(),
				AuthToken: "token",
			},
			expectedTransactionsLen: 1,
		},
		"returns error when invalid end date": {
			opts: export.Options{
				StartDate: time.Now(),
				AuthToken: "token",
			},
			expectedErrMsg: "invalid options: EndDate: is required.",
		},
		"returns error when invalid start date": {
			opts: export.Options{
				EndDate:   time.Now(),
				AuthToken: "token",
			},
			expectedErrMsg: "invalid options: StartDate: is required.",
		},
		"returns error when invalid token": {
			opts: export.Options{
				EndDate:   time.Now(),
				StartDate: time.Now(),
			},
			expectedErrMsg: "invalid options: AuthToken: is required.",
		},
		"returns error when invalid exporter": {
			opts: export.Options{
				EndDate:   time.Now(),
				StartDate: time.Now(),
				AuthToken: "12345",
			},
			expectedErrMsg: "exporter: constructor: invalid auth token",
		},
		"returns error when date range too long": {
			opts: export.Options{
				StartDate: time.Now().Add(-48 * time.Hour),
				EndDate:   time.Now(),
				AuthToken: "token",
			},
			expectedErrMsg: "date range 2 days is too long, max is 1 days",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			transactions, err := export.Transactions(t.Context(), ExportTypeStub, test.opts)

			if test.expectedErrMsg != "" {
				require.Nil(t, transactions)
				require.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Len(t, transactions, test.expectedTransactionsLen)
			}
		})
	}
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

func (s *StubExporter) ExportAccounts(ctx context.Context) ([]*domain.Account, error) {
	return s.accounts, s.err
}

func (s *StubExporter) ExportTransactions(ctx context.Context, opts export.Options) ([]*domain.Transaction, error) {
	return s.transactions, s.err
}
