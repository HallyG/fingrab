package export_test

import (
	"context"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/api/monzo"
	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/stretchr/testify/require"
)

type StubMonzoClient struct {
	Accounts         []*monzo.Account
	Pots             []*monzo.Pot
	Transactions     [][]*monzo.Transaction
	FetchAccountsErr error
	FetchPotErr      error
	FetchTxnsErr     error
	callCount        int
}

var _ monzo.Client = (*StubMonzoClient)(nil)

func (s *StubMonzoClient) FetchTransactionsSince(ctx context.Context, opts monzo.FetchTransactionOptions) ([]*monzo.Transaction, error) {
	if s.FetchTxnsErr != nil {
		return nil, s.FetchTxnsErr
	}

	s.callCount++
	index := s.callCount - 1

	// If we've exceeded the number of predefined responses, return empty
	if index >= len(s.Transactions) {
		return nil, nil
	}

	return s.Transactions[index], nil
}

func (s *StubMonzoClient) FetchTransaction(ctx context.Context, transactionID monzo.TransactionID) (*monzo.Transaction, error) {
	return &monzo.Transaction{}, nil
}

func (s *StubMonzoClient) FetchAccounts(ctx context.Context) ([]*monzo.Account, error) {
	if s.FetchAccountsErr != nil {
		return nil, s.FetchAccountsErr
	}

	return s.Accounts, nil
}

func (s *StubMonzoClient) FetchPots(ctx context.Context, accountID monzo.AccountID) ([]*monzo.Pot, error) {
	if s.FetchPotErr != nil {
		return nil, s.FetchPotErr
	}

	return s.Pots, nil
}

func TestNewMonzoTransactionExport(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exporter := export.NewMonzoTransactionExporter(nil)

		require.NotNil(t, exporter)
		require.Equal(t, 90*24*time.Hour, exporter.MaxDateRange())
		require.Equal(t, export.ExportTypeMonzo, exporter.Type())
	})

	t.Run("registry", func(t *testing.T) {
		t.Parallel()

		exporter, err := export.NewExporter(export.ExportTypeMonzo, export.Options{})

		require.NoError(t, err)
		require.NotNil(t, exporter)
		require.Equal(t, 90*24*time.Hour, exporter.MaxDateRange())
		require.Equal(t, export.ExportTypeMonzo, exporter.Type())
	})
}

func TestExportMonzoTransactions(t *testing.T) {
	setup := func(t *testing.T, txns []*monzo.Transaction) (*monzo.Account, export.Exporter) {
		t.Helper()

		accountID := "acc_12345"
		accounts := []*monzo.Account{
			{
				ID: monzo.AccountID(accountID),
			},
		}
		pots := []*monzo.Pot{}

		client := &StubMonzoClient{
			Accounts:     accounts,
			Pots:         pots,
			Transactions: [][]*monzo.Transaction{txns},
		}

		return accounts[0], export.NewMonzoTransactionExporter(client)
	}

	now := time.Now()
	tests := map[string]struct {
		transactions         []*monzo.Transaction
		expectedTransactions []*domain.Transaction
	}{
		"excludes declined transactions": {
			transactions: []*monzo.Transaction{
				{
					DeclineReason: "declined",
					Description:   "declined",
					Amount: domain.Money{
						MinorUnit: 118,
						Currency:  "GBP",
					},
					LocalAmount: domain.Money{
						MinorUnit: 140,
						Currency:  "EUR",
					},
				},
				{
					Description: "settled",
					SettledAt:   &now,
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
				},
			},
			expectedTransactions: []*domain.Transaction{
				{
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
					Reference: "settled",
					Category:  "",
					CreatedAt: time.Time{},
					IsDeposit: false,
					BankName:  "Monzo",
					Notes:     "",
				},
			},
		},
		"excludes active card checks": {
			transactions: []*monzo.Transaction{
				{
					Description: "active card check",
					Amount: domain.Money{
						MinorUnit: 0,
						Currency:  "GBP",
					},
					LocalAmount: domain.Money{
						MinorUnit: 0,
						Currency:  "USD",
					},
					SettledAt: nil,
					Metadata: map[string]string{
						"notes": "Active card check",
					},
				},
			},
		},
		"includes transactions created on today": {
			transactions: []*monzo.Transaction{
				{
					Description: "settled",
					SettledAt:   &now,
					CreatedAt:   now,
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
				},
			},
			expectedTransactions: []*domain.Transaction{
				{
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
					Reference: "settled",
					Category:  "",
					CreatedAt: now,
					IsDeposit: false,
					BankName:  "Monzo",
					Notes:     "",
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			account, exporter := setup(t, test.transactions)
			res, err := exporter.ExportTransactions(
				t.Context(),
				export.Options{
					StartDate: time.Now().Add(-24 * time.Hour),
					EndDate:   time.Now(),
					AccountID: string(account.ID),
					Timeout:   10 * time.Second,
					AuthToken: "test-token",
				},
			)
			require.NoError(t, err)

			require.ElementsMatch(t, res, test.expectedTransactions)
		})
	}
}
