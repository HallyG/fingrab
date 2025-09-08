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

		format, err := format.NewFormatter("unknown", nil)

		require.Nil(t, format)
		require.ErrorContains(t, err, "unsupported type: unknown")
	})
}

func TestWriteAll(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		headerErr        error
		transactionErr   error
		flushErr         error
		expectedErrorMsg string
	}{
		"returns error when writing header fails": {
			expectedErrorMsg: "write header: io error",
			headerErr:        errors.New("io error"),
		},
		"returns error when writing transaction fails": {
			expectedErrorMsg: "write transaction: io error",
			transactionErr:   errors.New("io error"),
		},
		"returns error when flush fails": {
			expectedErrorMsg: "flush: io error",
			flushErr:         errors.New("io error"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			buffer := bytes.NewBuffer(nil)
			formatter := &StubFormatter{w: buffer}
			formatter.headerErr = test.headerErr
			formatter.transactionErr = test.transactionErr
			formatter.flushErr = test.flushErr

			err := format.WriteCollection(formatter, testTransactions(t, time.Now()))
			require.ErrorContains(t, err, test.expectedErrorMsg)
		})
	}
}

func testTransactions(t *testing.T, now time.Time) []*domain.Transaction {
	t.Helper()

	// We chose this time because it's close to midnight boundary, allowing us to test timezone changes
	time, err := time.Parse(time.RFC3339, "2025-05-04T23:16:52.392Z")
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
