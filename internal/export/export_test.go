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

func TestNewExporter(t *testing.T) {
	t.Parallel()

	t.Run("returns error for unknown type", func(t *testing.T) {
		t.Parallel()

		exporter, err := export.NewExporter(export.ExportType("wow"), export.Options{})

		require.Nil(t, exporter)
		require.ErrorContains(t, err, "unsupported export type")
	})
}

func TestTransactions(t *testing.T) {
	t.Parallel()

	export.Register(ExportTypeStub, func(opts export.Options) (export.Exporter, error) {
		if opts.AuthToken == "12345" {
			return nil, errors.New("some error")
		}

		return &StubExporter{
			transactions: []*domain.Transaction{
				{},
			},
		}, nil
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		transactions, err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			EndDate:   time.Now(),
			StartDate: time.Now(),
			AuthToken: "token",
		})

		require.Len(t, transactions, 1)
		require.NoError(t, err)
	})

	t.Run("invalid opts", func(t *testing.T) {
		t.Parallel()

		transcations, err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			StartDate: time.Now(),
			AuthToken: "token",
		})

		require.Nil(t, transcations)
		require.ErrorContains(t, err, "end time is required")
	})

	t.Run("invalid exporter", func(t *testing.T) {
		t.Parallel()

		transcations, err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			EndDate:   time.Now(),
			StartDate: time.Now(),
			AuthToken: "12345",
		})

		require.Nil(t, transcations)
		require.ErrorContains(t, err, "failed to create stubtype exporter")
	})

	t.Run("date range too long", func(t *testing.T) {
		t.Parallel()

		transactions, err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			StartDate: time.Now().Add(-48 * time.Hour),
			EndDate:   time.Now(),
			AuthToken: "token",
		})

		require.ErrorContains(t, err, "date range is too long, max is 1 days")
		require.Nil(t, transactions)
	})
}

const ExportTypeStub export.ExportType = "stubtype"

var _ export.Exporter = (*StubExporter)(nil)

type StubExporter struct {
	transactions []*domain.Transaction
	err          error
}

func (s *StubExporter) Type() export.ExportType {
	return ExportTypeStub
}

func (s *StubExporter) MaxDateRange() time.Duration {
	return 24 * time.Hour
}

func (s *StubExporter) ExportTransactions(ctx context.Context, opts export.Options) ([]*domain.Transaction, error) {
	return s.transactions, s.err
}
