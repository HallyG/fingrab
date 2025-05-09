package export_test

import (
	"context"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/api/starling"
	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type StubStarlingClient struct {
	Accounts         []*starling.Account
	SavingsGoals     []*starling.SavingsGoal
	Transactions     []*starling.FeedItem
	FetchAccountsErr error
	FetchGoalsErr    error
	FetchTxnsErr     error
}

var _ starling.Client = (*StubStarlingClient)(nil)

func (s *StubStarlingClient) FetchTransactionsSince(ctx context.Context, opts starling.FetchTransactionOptions) ([]*starling.FeedItem, error) {
	if s.FetchTxnsErr != nil {
		return nil, s.FetchTxnsErr
	}

	return s.Transactions, nil
}

func (s *StubStarlingClient) FetchFeedItem(ctx context.Context, accountID starling.AccountID, categoryID starling.CategoryID, feedItemID starling.FeedItemID) (*starling.FeedItem, error) {
	return &starling.FeedItem{}, nil
}

func (s *StubStarlingClient) FetchAccounts(ctx context.Context) ([]*starling.Account, error) {
	if s.FetchAccountsErr != nil {
		return nil, s.FetchAccountsErr
	}

	return s.Accounts, nil
}

func (s *StubStarlingClient) FetchSavingsGoals(ctx context.Context, accountID starling.AccountID) ([]*starling.SavingsGoal, error) {
	if s.FetchGoalsErr != nil {
		return nil, s.FetchGoalsErr
	}

	return s.SavingsGoals, nil
}

func TestNewStarlingTransactionExport(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exporter := export.NewStarlingTransactionExporter(nil)

		require.NotNil(t, exporter)
		require.Equal(t, time.Duration(0), exporter.MaxDateRange())
		require.Equal(t, export.ExportTypeStarling, exporter.Type())
	})

	t.Run("registry", func(t *testing.T) {
		t.Parallel()

		exporter, err := export.NewExporter(export.ExportTypeStarling, export.Options{})

		require.NoError(t, err)
		require.NotNil(t, exporter)
		require.Equal(t, time.Duration(0), exporter.MaxDateRange())
		require.Equal(t, export.ExportTypeStarling, exporter.Type())
	})
}

func TestExportStarlingTransactions(t *testing.T) {
	t.Run("excludes declined transactions", func(t *testing.T) {
		t.Parallel()

		accountID := starling.AccountID(uuid.New())
		categoryID := starling.CategoryID(uuid.New())

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

		client := &StubStarlingClient{
			Accounts:     accounts,
			SavingsGoals: savingsGoals,
			Transactions: transactions,
		}

		res, err := export.NewStarlingTransactionExporter(client).
			ExportTransactions(
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
