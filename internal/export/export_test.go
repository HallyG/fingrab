package export_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/format"
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

		buffer := bytes.NewBuffer(nil)
		formatter := &StubFormatter{w: buffer}

		err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			Format:    format.FormatTypeMoneyDance,
			EndDate:   time.Now(),
			StartDate: time.Now(),
			AuthToken: "token",
		}, formatter)

		require.NoError(t, err)
	})

	t.Run("invalid opts", func(t *testing.T) {
		t.Parallel()

		buffer := bytes.NewBuffer(nil)
		formatter := &StubFormatter{w: buffer}

		err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			Format:    format.FormatTypeMoneyDance,
			StartDate: time.Now(),
			AuthToken: "token",
		}, formatter)

		require.ErrorContains(t, err, "end time is required")
	})

	t.Run("invalid exporter", func(t *testing.T) {
		t.Parallel()

		buffer := bytes.NewBuffer(nil)
		formatter := &StubFormatter{w: buffer}

		err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			Format:    format.FormatTypeMoneyDance,
			EndDate:   time.Now(),
			StartDate: time.Now(),
			AuthToken: "12345",
		}, formatter)

		require.ErrorContains(t, err, "failed to create stubtype exporter")
	})

	t.Run("date range too long", func(t *testing.T) {
		t.Parallel()

		buffer := bytes.NewBuffer(nil)
		formatter := &StubFormatter{w: buffer}

		err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			Format:    format.FormatTypeMoneyDance,
			StartDate: time.Now().Add(-48 * time.Hour),
			EndDate:   time.Now(),
			AuthToken: "token",
		}, formatter)

		require.ErrorContains(t, err, "date range is too long, max is 1 days")
	})

	tests := []struct {
		name           string
		expectedMsg    string
		headerErr      error
		transactionErr error
		flushErr       error
	}{
		{
			name:        "formatter write header error",
			expectedMsg: "header error",
			headerErr:   errors.New("header error"),
		},
		{
			name:           "formatter write transaction error",
			expectedMsg:    "transaction error",
			transactionErr: errors.New("transaction error"),
		},
		{
			name:        "formatter flush error",
			expectedMsg: "flush error",
			flushErr:    errors.New("flush error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			buffer := bytes.NewBuffer(nil)
			formatter := &StubFormatter{w: buffer}
			formatter.headerErr = test.headerErr
			formatter.transactionErr = test.transactionErr
			formatter.flushErr = test.flushErr

			err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
				Format:    format.FormatTypeMoneyDance,
				EndDate:   time.Now(),
				StartDate: time.Now(),
				AuthToken: "token",
			}, formatter)

			require.ErrorContains(t, err, test.expectedMsg)
		})
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		buffer := bytes.NewBuffer(nil)
		formatter := &StubFormatter{w: buffer}

		err := export.Transactions(t.Context(), ExportTypeStub, export.Options{
			Format:    format.FormatTypeMoneyDance,
			EndDate:   time.Now(),
			StartDate: time.Now(),
			AuthToken: "token",
		}, formatter)

		require.NoError(t, err)
		require.Contains(t, "header content\ntransaction content\n", buffer.String())
	})
}

const ExportTypeStub export.ExportType = "stubtype"

var _ export.Exporter = (*StubExporter)(nil)
var _ format.Formatter = (*StubFormatter)(nil)

type StubExporter struct {
	transactions []*domain.Transaction
	err          error
}

type StubFormatter struct {
	w              io.Writer
	headerErr      error
	transactionErr error
	flushErr       error
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

func (s *StubFormatter) WriteHeader() error {
	if s.headerErr != nil {
		return s.headerErr
	}

	_, err := s.w.Write([]byte("header content\n"))

	return err
}

func (s *StubFormatter) WriteTransaction(transaction *domain.Transaction) error {
	if s.transactionErr != nil {
		return s.transactionErr
	}

	_, err := s.w.Write([]byte("transaction content\n"))

	return err
}

func (s *StubFormatter) Flush() error {
	return s.flushErr
}
