package format_test

import (
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/format"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	t.Run("returns expected formats", func(t *testing.T) {
		t.Parallel()

		formats := format.All()

		require.Len(t, formats, 2)
		require.Equal(t, []format.FormatType{format.FormatTypeMoneyDance, format.FormatTypeYNAB}, formats)
	})
}

func TestNewFormatter(t *testing.T) {
	t.Run("returns error for unknown format", func(t *testing.T) {
		t.Parallel()

		_, err := format.NewFormatter("unknown", nil)
		require.Error(t, err)
	})
}

func writeTestTransactions(t *testing.T, now time.Time, formatter format.Formatter) {
	t.Helper()

	err := formatter.WriteHeader()
	require.NoError(t, err)

	err = formatter.WriteTransaction(&domain.Transaction{
		CreatedAt: now,
		Reference: "Test Transaction",
		Category:  "Test Category",
		Amount:    domain.Money{MinorUnit: 12345, Currency: "GBP"},
		Notes:     "Test Notes",
		IsDeposit: true,
	})
	require.NoError(t, err)

	err = formatter.WriteTransaction(&domain.Transaction{
		CreatedAt: now,
		Reference: "Another Test Transaction",
		Category:  "Another Test Category",
		Amount:    domain.Money{MinorUnit: -12345, Currency: "GBP"},
		Notes:     "More notes",
	})
	require.NoError(t, err)

	err = formatter.Flush()
	require.NoError(t, err)
}
