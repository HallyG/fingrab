package starling_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/api/starling"
	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/util/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	token = "mock-token"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		api := starling.New(nil)
		require.NotNil(t, api)
	})
}

func TestFetchAccounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
		expectedLength      int
	}{
		{
			name:                "successful fetch",
			expectedQueryParams: map[string]string{},
			expectedHeaders: map[string]string{
				"Authorization": token,
			},
			expectedLength: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertHandler := testutil.ServeJSONTestDataHandler(t, http.StatusOK, "accounts.json")
			route := testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/api/v2/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					assertHandler(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := starling.New(&http.Client{},
				starling.WithBaseURL(server.URL),
				starling.WithAuthToken(token),
			)

			accounts, err := client.FetchAccounts(t.Context())
			require.NoError(t, err, "failed to fetch accounts")
			require.Len(t, accounts, test.expectedLength, "unexpected number of accounts")

			if len(accounts) > 0 {
				account := accounts[0]
				require.Equal(t, starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033")), account.ID, "account ID should match")
				require.Equal(t, starling.CategoryID(uuid.MustParse("00000000-0000-4000-0000-000000000099")), account.DefaultCategoryID, "category ID should match")
				require.Equal(t, "PRIMARY", account.Type, "type should match")
				require.Equal(t, "GBP", account.Currency, "currency should match")
				require.Equal(t, "2020-02-02T02:02:22Z", account.CreatedAt.Format(time.RFC3339), "created at time should match")
				require.Equal(t, "Personal", account.Name, "name should match")
			}
		})
	}

	t.Run("returns API error", func(t *testing.T) {
		t.Parallel()

		route := testutil.HTTPTestRoute{
			Method: http.MethodGet,
			URL:    "/api/v2/accounts",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertRequest(t, r, http.MethodGet, nil, nil)
				testutil.ServeJSONTestDataHandler(t, http.StatusUnauthorized, "error.json")(w, r)
			},
		}

		server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
		client := starling.New(&http.Client{},
			starling.WithBaseURL(server.URL),
			starling.WithAuthToken(token),
		)

		ctx := t.Context()
		accounts, err := client.FetchAccounts(ctx)

		require.Error(t, err)
		require.Nil(t, accounts)
		require.Contains(t, err.Error(), "Header: Authorization` must be set  (http status=401)")
	})

	t.Run("returns API error array", func(t *testing.T) {
		t.Parallel()

		route := testutil.HTTPTestRoute{
			Method: http.MethodGet,
			URL:    "/api/v2/accounts",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertRequest(t, r, http.MethodGet, nil, nil)
				testutil.ServeJSONTestDataHandler(t, http.StatusUnauthorized, "error-array.json")(w, r)
			},
		}

		server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
		client := starling.New(&http.Client{},
			starling.WithBaseURL(server.URL),
			starling.WithAuthToken(token),
		)

		ctx := t.Context()
		accounts, err := client.FetchAccounts(ctx)

		require.Error(t, err)
		require.Nil(t, accounts)

		var starlingErr starling.Error
		ok := errors.As(errors.Unwrap(err), &starlingErr)
		require.True(t, ok)

		require.Empty(t, starlingErr.Code, "code should be empty")
		require.Empty(t, starlingErr.Message, "message should be empty")
		require.Equal(t, http.StatusUnauthorized, starlingErr.HTTPStatus, "http status should match")
		require.Equal(t, []starling.ErrorMessage{
			{"MAX_TRANSACTION_TIMESTAMP_must not be null"},
			{"MIN_TRANSACTION_TIMESTAMP_must not be null"},
		}, starlingErr.ErrorMessages)
	})
}

func TestFetchSavingsGoals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
		expectedLength      int
	}{
		{
			name:                "successful fetch",
			expectedQueryParams: map[string]string{},
			expectedHeaders: map[string]string{
				"Authorization": token,
			},
			expectedLength: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			accountId := starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033"))

			assertHandler := testutil.ServeJSONTestDataHandler(t, http.StatusOK, "savings-goals.json")
			route := testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/account/%s/savings-goals", accountId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					assertHandler(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := starling.New(&http.Client{},
				starling.WithBaseURL(server.URL),
				starling.WithAuthToken(token),
			)

			goals, err := client.FetchSavingsGoals(t.Context(), accountId)
			require.NoError(t, err, "failed to fetch savings goals")
			require.Len(t, goals, test.expectedLength, "unexpected number of savings goals")

			if len(goals) > 0 {
				goal := goals[0]
				require.Equal(t, starling.SavingsGoalID(uuid.MustParse("77887788-7788-7788-7788-778877887788")), goal.ID, "savings goal ID should match")
				require.Equal(t, "Trip to Paris", goal.Name, "name should match")
				require.Equal(t, "ACTIVE", goal.State, "status should match")
				require.Equal(t, domain.Money{MinorUnit: 123456, Currency: "GBP"}, goal.Target, "target should match")
				require.Equal(t, domain.Money{MinorUnit: 123456, Currency: "GBP"}, goal.TotalSaved, "total saved should match")
			}
		})
	}
}

func TestFetchFeedItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
		pending             bool
		datafilename        string
	}{
		{
			name:                "successful fetch",
			expectedQueryParams: map[string]string{},
			expectedHeaders: map[string]string{
				"Authorization": token,
			},
			pending:      false,
			datafilename: "feed-item.json",
		},
		{
			name:                "successful fetch pending item",
			expectedQueryParams: map[string]string{},
			expectedHeaders: map[string]string{
				"Authorization": token,
			},
			pending:      true,
			datafilename: "feed-item-pending.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			feedItemId := starling.FeedItemID(uuid.MustParse("11221122-1122-1122-1122-112211221122"))
			accountId := starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033"))
			categoryId := starling.CategoryID(uuid.MustParse("ccddccdd-ccdd-ccdd-ccdd-ccddccddccdd"))

			assertHandler := testutil.ServeJSONTestDataHandler(t, http.StatusOK, test.datafilename)
			route := testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/%s", accountId.String(), categoryId.String(), feedItemId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					assertHandler(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := starling.New(&http.Client{},
				starling.WithBaseURL(server.URL),
				starling.WithAuthToken(token),
			)

			feedItem, err := client.FetchFeedItem(t.Context(), accountId, categoryId, feedItemId)
			require.NoError(t, err, "failed to fetch feed item")

			require.NotNil(t, feedItem)
			require.Equal(t, starling.FeedItemID(uuid.MustParse("11221122-1122-1122-1122-112211221122")), feedItem.ID, "feed item ID should match")
			require.Equal(t, starling.CategoryID(uuid.MustParse("ccddccdd-ccdd-ccdd-ccdd-ccddccddccdd")), feedItem.CategoryID, "category ID should match")
			require.Equal(t, "GROCERIES", feedItem.CategoryName, "category name should match")
			require.Equal(t, domain.Money{MinorUnit: 123456, Currency: "GBP"}, feedItem.Amount, "amount should match")
			require.Equal(t, "Tax deductable, submit me to payroll", feedItem.UserNote, "user note should match")
			require.Equal(t, "TESCO-STORES-6148      SOUTHAMPTON   GBR", feedItem.Description, "description should match")
			require.Equal(t, "2025-02-19T16:37:59Z", feedItem.TransactedAt.Format(time.RFC3339), "transcated at time should match")
			require.Equal(t, starling.DirectionIN, feedItem.Direction, "direction should match")
			require.Equal(t, "MASTER_CARD", feedItem.Source, "source should match")
			require.Equal(t, "CONTACTLESS", feedItem.SourceSubType, "source sub type should match")
			require.Equal(t, "MERCHANT", feedItem.CounterPartyType, "counter party type should match")
			require.Equal(t, "Tesco", feedItem.CounterPartyName, "counter party name should match")
			require.Equal(t, starling.CounterPartyID(uuid.MustParse("68e16af4-c2c3-413b-bf93-1056b90097fa")), feedItem.CounterPartyID, "counter party ID should match")

			if test.pending {
				require.Equal(t, starling.StatusPending, feedItem.Status, "status should match")
				require.Empty(t, feedItem.SettledAt, "settled at time should match")
			} else {
				require.Equal(t, starling.StatusSettled, feedItem.Status, "status should match")
				require.Equal(t, "2025-02-19T16:38:59Z", feedItem.SettledAt.Format(time.RFC3339), "settled at time should match")
			}
		})
	}
}

func TestFetchTransactionsSince(t *testing.T) {
	t.Parallel()

	accountId := starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033"))
	categoryId := starling.CategoryID(uuid.MustParse("ccddccdd-ccdd-ccdd-ccdd-ccddccddccdd"))
	feedItemId := starling.FeedItemID(uuid.MustParse("11221122-1122-1122-1122-112211221122"))

	startTime, err := time.Parse(time.RFC3339, "2025-02-19T00:00:00Z")
	require.NoError(t, err)

	endTime, err := time.Parse(time.RFC3339, "2025-02-20T00:00:00Z")
	require.NoError(t, err)

	tests := []struct {
		name                string
		opts                starling.FetchTransactionOptions
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
		expectedLength      int
	}{
		{
			name: "successful fetch",
			opts: starling.FetchTransactionOptions{
				AccountID:  accountId,
				CategoryID: categoryId,
				Start:      startTime,
				End:        endTime,
			},
			expectedQueryParams: map[string]string{
				"minTransactionTimestamp": "2025-02-19T00:00:00Z",
				"maxTransactionTimestamp": "2025-02-20T00:00:00Z",
			},
			expectedHeaders: map[string]string{
				"Authorization": token,
			},
			expectedLength: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			route := testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/transactions-between", accountId.String(), categoryId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-items.json")(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := starling.New(&http.Client{},
				starling.WithBaseURL(server.URL),
				starling.WithAuthToken(token),
			)

			feedItems, err := client.FetchTransactionsSince(t.Context(), test.opts)
			require.NoError(t, err, "failed to fetch feed items")
			require.Len(t, feedItems, test.expectedLength, "unexpected number of feed items")

			if len(feedItems) > 0 {
				require.Equal(t, feedItemId, feedItems[0].ID, "feed item ID should match")
			}
		})
	}

	t.Run("returns error when invalid options", func(t *testing.T) {
		t.Parallel()

		client := starling.New(&http.Client{},
			starling.WithBaseURL(""),
			starling.WithAuthToken(token),
		)

		now := time.Now()
		_, err := client.FetchTransactionsSince(t.Context(), starling.FetchTransactionOptions{
			End:       now.Add(-24 * time.Hour),
			Start:     now,
			AccountID: accountId,
		})
		require.ErrorContains(t, err, "start time must be before end time")
	})
}
