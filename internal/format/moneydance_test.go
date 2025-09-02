package format_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/format"
	"github.com/stretchr/testify/require"
)

func TestMoneyDanceFormatter(t *testing.T) {
	t.Parallel()

	t.Run("writes CSV data", func(t *testing.T) {
		t.Parallel()

		now, err := time.Parse("2006-01-02", "2025-04-16")
		require.NoError(t, err)

		buffer := bytes.NewBuffer(nil)
		formatter, err := format.NewFormatter(format.FormatTypeMoneyDance, buffer)
		require.NoError(t, err)

		writeTestTransactions(t, now, formatter)

		expected := `check number,date,description,category,amount,memo
Dep,2025-04-16,Test Transaction,Test Category,123.45,Test Notes
Trn,2025-04-16,Another Test Transaction,Another Test Category,-123.45,More notes
Trn,2025-05-04,Transaction With Date Affected By Timezone,Test Category,-1.00,Test Notes
`
		require.Equal(t, expected, buffer.String())
	})
}
