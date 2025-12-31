package exporter_test

import (
	"context"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/starling"
	starlingexporter "github.com/HallyG/fingrab/internal/starling/exporter"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type StubClient struct {
	Accounts         []*starling.Account
	SavingsGoals     []*starling.SavingsGoal
	Transactions     []*starling.FeedItem
	FetchAccountsErr error
	FetchGoalsErr    error
	FetchTxnsErr     error
}

var _ starling.Client = (*StubClient)(nil)

func (c *StubClient) FetchTransactionsSince(ctx context.Context, opts starling.FetchTransactionOptions) ([]*starling.FeedItem, error) {
	if c.FetchTxnsErr != nil {
		return nil, c.FetchTxnsErr
	}

	return c.Transactions, nil
}

func (c *StubClient) FetchFeedItem(ctx context.Context, accountID starling.AccountID, categoryID starling.CategoryID, feedItemID starling.FeedItemID) (*starling.FeedItem, error) {
	return &starling.FeedItem{}, nil
}

func (c *StubClient) FetchAccounts(ctx context.Context) ([]*starling.Account, error) {
	if c.FetchAccountsErr != nil {
		return nil, c.FetchAccountsErr
	}

	return c.Accounts, nil
}

func (c *StubClient) FetchSavingsGoals(ctx context.Context, accountID starling.AccountID) ([]*starling.SavingsGoal, error) {
	if c.FetchGoalsErr != nil {
		return nil, c.FetchGoalsErr
	}

	return c.SavingsGoals, nil
}

func TestNewTransactionExport(t *testing.T) {
	t.Parallel()

	t.Run("error when nil client success", func(t *testing.T) {
		t.Parallel()

		exporter, err := starlingexporter.New(nil)

		require.Nil(t, exporter)
		require.ErrorContains(t, err, "starling client is required")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exporter, err := starlingexporter.New(&StubClient{})

		require.NoError(t, err)
		require.NotNil(t, exporter)
		require.Equal(t, time.Duration(0), exporter.MaxDateRange())
		require.Equal(t, starlingexporter.ExportTypeStarling, exporter.Type())
	})
}

func TestExportAccounts(t *testing.T) {
	t.Parallel()

	accountID := starling.AccountID(uuid.New())
	categoryID := starling.CategoryID(uuid.New())
	now := time.Now()

	setup := func(t *testing.T) export.Exporter {
		t.Helper()

		accounts := []*starling.Account{
			{
				ID:                accountID,
				DefaultCategoryID: categoryID,
				Type:              "PRIMARY",
				Currency:          "GBP",
				CreatedAt:         now,
			},
		}

		client := &StubClient{
			Accounts: accounts,
		}

		exporter, err := starlingexporter.New(client)
		require.NoError(t, err)

		return exporter
	}

	t.Run("returns accounts", func(t *testing.T) {
		t.Parallel()

		accounts, err := setup(t).ExportAccounts(t.Context())
		require.NoError(t, err)

		require.Len(t, accounts, 1)
		require.Equal(t, accountID.String(), accounts[0].ID)
		require.Equal(t, "PRIMARY", accounts[0].Type)
		require.WithinDuration(t, now, accounts[0].CreatedAt, time.Second)
	})
}

func TestExportTransactions(t *testing.T) {
	t.Parallel()

	accountID := starling.AccountID(uuid.New())
	categoryID := starling.CategoryID(uuid.New())

	setup := func(t *testing.T) export.Exporter {
		t.Helper()

		accounts := []*starling.Account{
			{
				ID:                accountID,
				DefaultCategoryID: categoryID,
			},
		}
		savingsGoals := []*starling.SavingsGoal{}
		transactions := []*starling.FeedItem{
			{
				CategoryID:  categoryID,
				Status:      starling.StatusDeclined,
				Direction:   starling.DirectionOUT,
				Description: "declined",
				Amount: domain.Money{
					MinorUnit: 118,
					Currency:  "GBP",
				},
			},
			{
				CategoryID:  categoryID,
				Status:      starling.StatusSettled,
				Direction:   starling.DirectionIN,
				Description: "interest",
				Amount: domain.Money{
					MinorUnit: 123,
					Currency:  "GBP",
				},
			},
			{
				CategoryID:  categoryID,
				Status:      starling.StatusSettled,
				Direction:   starling.DirectionOUT,
				Description: "settled",
				Amount: domain.Money{
					MinorUnit: 276,
					Currency:  "GBP",
				},
			},
		}

		client := &StubClient{
			Accounts:     accounts,
			SavingsGoals: savingsGoals,
			Transactions: transactions,
		}

		exporter, err := starlingexporter.New(client)
		require.NoError(t, err)

		return exporter
	}

	t.Run("excludes declined transactions", func(t *testing.T) {
		t.Parallel()

		res, err := setup(t).ExportTransactions(
			t.Context(),
			export.Options{
				StartDate: time.Now().Add(-24 * time.Hour),
				EndDate:   time.Now(),
				AccountID: accountID.String(),
				Timeout:   10 * time.Second,
				AuthToken: "test-token",
			},
		)
		require.NoError(t, err)

		require.Len(t, res, 2)
		require.Equal(t, "settled", res[1].Reference)
		require.Equal(t, domain.Money{
			MinorUnit: -276,
			Currency:  "GBP",
		}, res[1].Amount)
		require.Equal(t, "interest", res[0].Reference)
		require.Equal(t, domain.Money{
			MinorUnit: 123,
			Currency:  "GBP",
		}, res[0].Amount)
	})
}
