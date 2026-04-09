package export_test

import (
	"errors"
	"testing"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/stretchr/testify/require"
)

func TestAccounts(t *testing.T) {
	t.Parallel()

	export.Register(ExportTypeStub, func(opts export.Options) (export.Exporter, error) {
		if opts.AuthToken == "12345" {
			return nil, errors.New("invalid auth token")
		}

		return &StubExporter{
			transactions: []*domain.Transaction{
				{},
			},
			accounts: []*domain.Account{
				{},
			},
		}, nil
	})

	tests := map[string]struct {
		opts                export.AccountOptions
		expectedErrMsg      string
		expectedAccountsLen int
	}{
		"success": {
			opts: export.AccountOptions{
				Options: export.Options{
					AuthToken: "token",
				},
			},
			expectedAccountsLen: 1,
		},
		"returns error when invalid token": {
			opts:           export.AccountOptions{},
			expectedErrMsg: "invalid options: AuthToken: is required.",
		},
		"returns error when invalid exporter": {
			opts: export.AccountOptions{
				Options: export.Options{
					AuthToken: "12345",
				},
			},
			expectedErrMsg: "exporter: constructor: invalid auth token",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			accounts, err := export.Accounts(t.Context(), ExportTypeStub, test.opts)

			if test.expectedErrMsg != "" {
				require.Nil(t, accounts)
				require.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Len(t, accounts, test.expectedAccountsLen)
			}
		})
	}
}
