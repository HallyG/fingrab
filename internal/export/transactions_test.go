package export_test

import (
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/stretchr/testify/require"
)

func TestTransactions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		opts                    export.TransactionOptions
		expectedErrMsg          string
		expectedTransactionsLen int
	}{
		"success": {
			opts: export.TransactionOptions{
				EndDate:   time.Now(),
				StartDate: time.Now(),
				Options: export.Options{
					AuthToken: "token",
				},
			},
			expectedTransactionsLen: 1,
		},
		"returns error when invalid end date": {
			opts: export.TransactionOptions{
				StartDate: time.Now(),
				Options: export.Options{
					AuthToken: "token",
				},
			},
			expectedErrMsg: "invalid options: EndDate: is required.",
		},
		"returns error when invalid start date": {
			opts: export.TransactionOptions{
				EndDate: time.Now(),
				Options: export.Options{
					AuthToken: "token",
				},
			},
			expectedErrMsg: "invalid options: StartDate: is required.",
		},
		"returns error when invalid token": {
			opts: export.TransactionOptions{
				EndDate:   time.Now(),
				StartDate: time.Now(),
			},
			expectedErrMsg: "invalid options: AuthToken: is required.",
		},
		"returns error when invalid exporter": {
			opts: export.TransactionOptions{
				EndDate:   time.Now(),
				StartDate: time.Now(),
				Options: export.Options{
					AuthToken: "12345",
				},
			},
			expectedErrMsg: "exporter: constructor: invalid auth token",
		},
		"returns error when date range too long": {
			opts: export.TransactionOptions{
				StartDate: time.Now().Add(-48 * time.Hour),
				EndDate:   time.Now(),
				Options: export.Options{
					AuthToken: "token",
				},
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
