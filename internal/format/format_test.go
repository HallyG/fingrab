package format_test

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/format"
	"github.com/stretchr/testify/require"
)

var _ format.Formatter = (*StubFormatter)(nil)

type StubFormatter struct {
	w              io.Writer
	headerErr      error
	transactionErr error
	flushErr       error
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

func TestAll(t *testing.T) {
	t.Parallel()

	t.Run("returns expected formats", func(t *testing.T) {
		t.Parallel()

		formats := format.All()

		require.Len(t, formats, 2)
		require.Equal(t, []format.FormatType{format.FormatTypeMoneyDance, format.FormatTypeYNAB}, formats)
	})
}

func TestNewFormatter(t *testing.T) {
	t.Parallel()

	t.Run("returns error for unknown format", func(t *testing.T) {
		t.Parallel()

		_, err := format.NewFormatter("unknown", nil)
		require.Error(t, err)
	})
}

func TestWriteAll(t *testing.T) {
	t.Parallel()

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

			err := format.WriteCollection(formatter, testTransactions(t, time.Now()))
			require.ErrorContains(t, err, test.expectedMsg)
		})
	}
}

func testTransactions(t *testing.T, now time.Time) []*domain.Transaction {
	t.Helper()

	time, err := time.Parse(time.RFC3339, "2025-05-04T23:16:52.392Z") // close to midnight boundary so we can test timezone changes
	require.NoError(t, err)

	return []*domain.Transaction{
		{
			CreatedAt: now,
			Reference: "Test Transaction",
			Category:  "Test Category",
			Amount:    domain.Money{MinorUnit: 12345, Currency: "GBP"},
			Notes:     "Test Notes",
			IsDeposit: true,
		},
		{
			CreatedAt: now,
			Reference: "Another Test Transaction",
			Category:  "Another Test Category",
			Amount:    domain.Money{MinorUnit: -12345, Currency: "GBP"},
			Notes:     "More notes",
		},
		{
			CreatedAt: time,
			Reference: "Transaction With Date Affected By Timezone",
			Category:  "Test Category",
			Amount:    domain.Money{MinorUnit: -100, Currency: "GBP"},
			Notes:     "Test Notes",
		},
	}
}
