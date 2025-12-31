package exporter_test

import (
	"context"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"

	"github.com/HallyG/fingrab/internal/monzo"
	monzoexporter "github.com/HallyG/fingrab/internal/monzo/exporter"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("error when nil client success", func(t *testing.T) {
		t.Parallel()

		exporter, err := monzoexporter.New(nil)

		require.Nil(t, exporter)
		require.ErrorContains(t, err, "monzo client is required")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exporter, err := monzoexporter.New(&StubClient{})

		require.NoError(t, err)
		require.NotNil(t, exporter)
		require.Equal(t, 90*24*time.Hour, exporter.MaxDateRange())
		require.Equal(t, monzoexporter.ExportTypeMonzo, exporter.Type())
	})
}

func TestExportAccounts(t *testing.T) {
	t.Parallel()

	accountID := "acc_12345"

	now := time.Now()

	setup := func(t *testing.T) export.Exporter {
		t.Helper()

		accounts := []*monzo.Account{
			{
				ID:        monzo.AccountID(accountID),
				CreatedAt: now,
				Type:      "uk_retail",
			},
		}

		client := &StubClient{
			Accounts: accounts,
		}

		exporter, err := monzoexporter.New(client)
		require.NoError(t, err)

		return exporter
	}

	t.Run("returns accounts", func(t *testing.T) {
		t.Parallel()

		accounts, err := setup(t).ExportAccounts(t.Context())
		require.NoError(t, err)

		require.Len(t, accounts, 1)
		require.Equal(t, string(accountID), accounts[0].ID)
		require.Equal(t, "uk_retail", accounts[0].Type)
		require.WithinDuration(t, now, accounts[0].CreatedAt, time.Second)
	})
}

func TestExportTransactions(t *testing.T) {
	t.Parallel()

	now := time.Now()

	setup := func(t *testing.T, txns []*monzo.Transaction) (*monzo.Account, *monzoexporter.TransactionExporter) {
		t.Helper()

		accountID := "acc_12345"
		accounts := []*monzo.Account{
			{
				ID: monzo.AccountID(accountID),
			},
		}
		pots := []*monzo.Pot{}

		client := &StubClient{
			Accounts:     accounts,
			Pots:         pots,
			Transactions: [][]*monzo.Transaction{txns},
		}

		exporter, err := monzoexporter.New(client)
		require.NoError(t, err)

		return accounts[0], exporter
	}

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
		"excludes transactions created tomorrow": {
			transactions: []*monzo.Transaction{
				{
					Description: "created tomorrow",
					Amount: domain.Money{
						MinorUnit: 0,
						Currency:  "GBP",
					},
					LocalAmount: domain.Money{
						MinorUnit: 0,
						Currency:  "USD",
					},
					SettledAt: nil,
					CreatedAt: now.AddDate(0, 0, 1),
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
		"includes split transaction": {
			transactions: []*monzo.Transaction{
				{
					Description: "settled",
					SettledAt:   &now,
					CreatedAt:   now,
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
					Merchant: &monzo.Merchant{
						Name: "Tesco",
					},
					CounterParty: &monzo.CounterParty{
						Name: "James",
					},
				},
				{
					Description: "settled",
					SettledAt:   &now,
					CreatedAt:   now,
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
					Merchant: &monzo.Merchant{
						Name: "Sainsburys",
					},
					CounterParty: &monzo.CounterParty{
						Name: "James",
					},
					UserNotes: "Beers",
				},
			},
			expectedTransactions: []*domain.Transaction{
				{
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
					Reference: "James",
					Category:  "",
					CreatedAt: now,
					IsDeposit: false,
					BankName:  "Monzo",
					Notes:     "Tesco",
				},
				{
					Amount: domain.Money{
						MinorUnit: 276,
						Currency:  "GBP",
					},
					Reference: "James",
					Category:  "",
					CreatedAt: now,
					IsDeposit: false,
					BankName:  "Monzo",
					Notes:     "Beers", // should not overwrite notes when we've set them in the app
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
				export.TransactionOptions{
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

var _ monzo.Client = (*StubClient)(nil)

type StubClient struct {
	Accounts         []*monzo.Account
	Pots             []*monzo.Pot
	Transactions     [][]*monzo.Transaction
	FetchAccountsErr error
	FetchPotErr      error
	FetchTxnsErr     error
	callCount        int
}

func (c *StubClient) FetchTransactionsSince(ctx context.Context, opts monzo.FetchTransactionOptions) ([]*monzo.Transaction, error) {
	if c.FetchTxnsErr != nil {
		return nil, c.FetchTxnsErr
	}

	c.callCount++
	index := c.callCount - 1

	// If we've exceeded the number of predefined responses, return empty
	if index >= len(c.Transactions) {
		return nil, nil
	}

	return c.Transactions[index], nil
}

func (c *StubClient) FetchTransaction(ctx context.Context, transactionID monzo.TransactionID) (*monzo.Transaction, error) {
	return &monzo.Transaction{}, nil
}

func (c *StubClient) FetchAccounts(ctx context.Context) ([]*monzo.Account, error) {
	if c.FetchAccountsErr != nil {
		return nil, c.FetchAccountsErr
	}

	return c.Accounts, nil
}

func (c *StubClient) FetchPots(ctx context.Context, accountID monzo.AccountID) ([]*monzo.Pot, error) {
	if c.FetchPotErr != nil {
		return nil, c.FetchPotErr
	}

	return c.Pots, nil
}
