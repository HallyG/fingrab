package format_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/format"
	"github.com/stretchr/testify/require"
)

func TestYNABFormatter(t *testing.T) {
	t.Run("writes CSV data", func(t *testing.T) {
		t.Parallel()

		now, err := time.Parse("2006-01-02", "2025-04-16")
		require.NoError(t, err)

		buffer := bytes.NewBuffer(nil)
		formatter, err := format.NewFormatter(format.FormatTypeYNAB, buffer)
		require.NoError(t, err)

		writeTestTransactions(t, now, formatter)

		expected := `Date,Payee,Memo,Amount
04/16/2025,Test Transaction,Test Notes,123.45
04/16/2025,Another Test Transaction,More notes,-123.45
05/04/2025,Transaction With Date Affected By Timezone,Test Notes,-1.00
`
		require.Equal(t, expected, buffer.String())
	})
}
