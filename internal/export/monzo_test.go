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

	t.Run("excludes declined transactions", func(t *testing.T) {
		t.Parallel()
		now := time.Now()

		transactions := []*monzo.Transaction{
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
		}

		account, exporter := setup(t, transactions)

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

		require.Len(t, res, 1)
		require.Equal(t, "settled", res[0].Reference)
		require.Equal(t, domain.Money{
			MinorUnit: 276,
			Currency:  "GBP",
		}, res[0].Amount, "transaction amount should be in the account's currency")
	})

	t.Run("excludes active card checks", func(t *testing.T) {
		t.Parallel()

		now := time.Now()

		transactions := []*monzo.Transaction{
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
			{
				Description: "settled",
				SettledAt:   &now,
				Amount: domain.Money{
					MinorUnit: 276,
					Currency:  "GBP",
				},
			},
		}

		account, exporter := setup(t, transactions)

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

		require.Len(t, res, 1)
		require.Equal(t, "settled", res[0].Reference)
		require.Equal(t, domain.Money{
			MinorUnit: 276,
			Currency:  "GBP",
		}, res[0].Amount, "transaction amount should be in the account's currency")
	})
}
