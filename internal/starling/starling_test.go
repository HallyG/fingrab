package starling_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/api"
	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/starling"
	"github.com/HallyG/fingrab/internal/util/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	token = "mock-token"
)

func setup(t *testing.T, routes ...testutil.HTTPTestRoute) starling.Client {
	t.Helper()

	server := testutil.NewHTTPTestServer(t, routes)
	client := starling.New(&http.Client{},
		api.WithBaseURL(server.URL),
		api.WithAuthToken(token),
	)

	return client
}

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

	tests := map[string]struct {
		route               testutil.HTTPTestRoute
		expectedAccounts    []*starling.Account
		expectedStarlingErr *starling.Error
		expectedErrMsg      string
		assertFn            func(t *testing.T, items []*starling.Account)
	}{
		"successful fetch": {
			expectedAccounts: []*starling.Account{
				{
					ID:                starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033")),
					DefaultCategoryID: starling.CategoryID(uuid.MustParse("00000000-0000-4000-0000-000000000099")),
					Type:              "PRIMARY",
					Currency:          "GBP",
					CreatedAt: testutil.MustParse(t, "2020-02-02T02:02:22.222Z", func(s string) (time.Time, error) {
						return time.Parse(time.RFC3339, s)
					}),
					Name: "Personal",
				},
			},
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/api/v2/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "accounts.json")(w, r)
				},
			},
			assertFn: func(t *testing.T, items []*starling.Account) {
				t.Helper()

				require.Len(t, items, 1)
			},
		},
		"returns API error": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/api/v2/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, nil, nil)
					testutil.ServeJSONTestDataHandler(t, http.StatusUnauthorized, "error.json")(w, r)
				},
			},
			expectedStarlingErr: &starling.Error{
				Code:    "invalid_token",
				Message: "No access token provided in request. `Header: Authorization` must be set",
			},
			expectedErrMsg: "No access token provided in request. `Header: Authorization` must be set",
		},
		"returns API error array": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/api/v2/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, nil, nil)
					testutil.ServeJSONTestDataHandler(t, http.StatusUnauthorized, "error-array.json")(w, r)
				},
			},
			expectedStarlingErr: &starling.Error{
				ErrorMessages: []starling.ErrorMessage{
					{"MAX_TRANSACTION_TIMESTAMP_must not be null"},
					{"MIN_TRANSACTION_TIMESTAMP_must not be null"},
				},
			},
			expectedErrMsg: "[MAX_TRANSACTION_TIMESTAMP_must not be null, MIN_TRANSACTION_TIMESTAMP_must not be null]",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)
			items, err := client.FetchAccounts(t.Context())

			if test.expectedStarlingErr != nil {
				require.Empty(t, items)
				requireStarlingErrorEqual(t, *test.expectedStarlingErr, test.expectedErrMsg, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, items, test.expectedAccounts)
				if test.assertFn != nil {
					test.assertFn(t, items)
				}
			}
		})
	}
}

func TestFetchSavingsGoals(t *testing.T) {
	t.Parallel()

	accountId := starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033"))

	tests := map[string]struct {
		route               testutil.HTTPTestRoute
		expectedGoals       []*starling.SavingsGoal
		expectedStarlingErr *starling.Error
		expectedErrMsg      string
		assertFn            func(t *testing.T, items []*starling.SavingsGoal)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/account/%s/savings-goals", accountId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "savings-goals.json")(w, r)
				},
			},
			expectedGoals: []*starling.SavingsGoal{
				{
					ID:         starling.SavingsGoalID(uuid.MustParse("77887788-7788-7788-7788-778877887788")),
					Name:       "Trip to Paris",
					State:      "ACTIVE",
					Target:     domain.Money{MinorUnit: 123457, Currency: "GBP"},
					TotalSaved: domain.Money{MinorUnit: 123456, Currency: "GBP"},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)
			items, err := client.FetchSavingsGoals(t.Context(), accountId)

			if test.expectedStarlingErr != nil {
				require.Empty(t, items)
				requireStarlingErrorEqual(t, *test.expectedStarlingErr, "", errors.Unwrap(err))
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, items, test.expectedGoals)
				require.Equal(t, "77887788-7788-7788-7788-778877887788", test.expectedGoals[0].ID.String())
				if test.assertFn != nil {
					test.assertFn(t, items)
				}
			}
		})
	}
}

func TestFetchFeedItem(t *testing.T) {
	t.Parallel()

	feedItemId := starling.FeedItemID(uuid.MustParse("11221122-1122-1122-1122-112211221122"))
	accountId := starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033"))
	categoryId := starling.CategoryID(uuid.MustParse("ccddccdd-ccdd-ccdd-ccdd-ccddccddccdd"))

	tests := map[string]struct {
		route               testutil.HTTPTestRoute
		expectedItem        *starling.FeedItem
		expectedStarlingErr *starling.Error
		expectedErrMsg      string
		assertFn            func(y *testing.T, item *starling.FeedItem)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/%s", accountId.String(), categoryId.String(), feedItemId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-item.json")(w, r)
				},
			},
			expectedItem: testutil.MarshalTestDataFile[starling.FeedItem](t, "feed-item.json"),
			assertFn: func(t *testing.T, item *starling.FeedItem) {
				t.Helper()

				require.Equal(t, starling.StatusSettled, item.Status)
				require.Equal(t, "2025-02-19T16:38:59Z", item.SettledAt.Format(time.RFC3339))
			},
		},
		"successful fetch pending item": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/%s", accountId.String(), categoryId.String(), feedItemId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-item-pending.json")(w, r)
				},
			},
			expectedItem: testutil.MarshalTestDataFile[starling.FeedItem](t, "feed-item-pending.json"),
			assertFn: func(t *testing.T, item *starling.FeedItem) {
				t.Helper()

				require.Equal(t, starling.StatusPending, item.Status)
				require.Empty(t, item.SettledAt)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)
			item, err := client.FetchFeedItem(t.Context(), accountId, categoryId, feedItemId)
			if test.expectedStarlingErr != nil {
				require.Nil(t, item)
				requireStarlingErrorEqual(t, *test.expectedStarlingErr, test.expectedErrMsg, errors.Unwrap(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedItem, item)
				if test.assertFn != nil {
					test.assertFn(t, item)
				}
			}
		})
	}
}

func TestFetchTransactionsSince(t *testing.T) {
	t.Parallel()

	accountId := starling.AccountID(uuid.MustParse("00000000-0000-4000-0000-000000000033"))
	categoryId := starling.CategoryID(uuid.MustParse("ccddccdd-ccdd-ccdd-ccdd-ccddccddccdd"))
	feedItemId := starling.FeedItemID(uuid.MustParse("11221122-1122-1122-1122-112211221122"))
	startTime := testutil.MustParse(t, "2025-02-19T00:00:00Z", func(s string) (time.Time, error) {
		return time.Parse(time.RFC3339, s)
	})
	endTime := testutil.MustParse(t, "2025-02-20T00:00:00Z", func(s string) (time.Time, error) {
		return time.Parse(time.RFC3339, s)
	})

	tests := map[string]struct {
		name           string
		route          testutil.HTTPTestRoute
		opts           starling.FetchTransactionOptions
		expectedItems  []*starling.FeedItem
		expectedErrMsg error
		assertFn       func(t *testing.T, items []*starling.FeedItem)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/transactions-between", accountId.String(), categoryId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)
					query.Add("minTransactionTimestamp", "2025-02-19T00:00:00Z")
					query.Add("maxTransactionTimestamp", "2025-02-20T00:00:00Z")

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-items.json")(w, r)
				},
			},
			opts: starling.FetchTransactionOptions{
				AccountID:  accountId,
				CategoryID: categoryId,
				Start:      startTime,
				End:        endTime,
			},
			expectedItems: testutil.MarshalTestDataFile[struct {
				FeedItems []*starling.FeedItem `json:"feedItems"`
			}](t, "feed-items.json").FeedItems,
			assertFn: func(t *testing.T, items []*starling.FeedItem) {
				t.Helper()

				require.Len(t, items, 1)
				require.Equal(t, feedItemId, items[0].ID)
			},
		},
		"returns error when start time before end time": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/transactions-between", accountId.String(), categoryId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)
					query.Add("minTransactionTimestamp", "2025-02-19T00:00:00Z")
					query.Add("maxTransactionTimestamp", "2025-02-20T00:00:00Z")

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-items.json")(w, r)
				},
			},
			opts: starling.FetchTransactionOptions{
				End:        time.Now().Add(-24 * time.Hour),
				Start:      time.Now(),
				AccountID:  accountId,
				CategoryID: categoryId,
			},
			expectedErrMsg: errors.New("invalid options: Start: must be before End"),
		},
		"returns error when invalid account id": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/transactions-between", accountId.String(), categoryId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)
					query.Add("minTransactionTimestamp", "2025-02-19T00:00:00Z")
					query.Add("maxTransactionTimestamp", "2025-02-20T00:00:00Z")

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-items.json")(w, r)
				},
			},
			opts: starling.FetchTransactionOptions{
				End:   time.Now().Add(24 * time.Hour),
				Start: time.Now(),
			},
			expectedErrMsg: errors.New("invalid options: AccountID: is required"),
		},
		"returns error when invalid category id": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/api/v2/feed/account/%s/category/%s/transactions-between", accountId.String(), categoryId.String()),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)
					query.Add("minTransactionTimestamp", "2025-02-19T00:00:00Z")
					query.Add("maxTransactionTimestamp", "2025-02-20T00:00:00Z")

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "feed-items.json")(w, r)
				},
			},
			opts: starling.FetchTransactionOptions{
				End:       time.Now().Add(24 * time.Hour),
				Start:     time.Now(),
				AccountID: accountId,
			},
			expectedErrMsg: errors.New("invalid options: CategoryID: is required"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)

			items, err := client.FetchTransactionsSince(t.Context(), test.opts)
			if test.expectedErrMsg != nil {
				require.Empty(t, items)
				require.ErrorContains(t, err, test.expectedErrMsg.Error())
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, items, test.expectedItems)
				require.Equal(t, "68e16af4-c2c3-413b-bf93-1056b90097fa", test.expectedItems[0].CounterPartyID.String())
				if test.assertFn != nil {
					test.assertFn(t, items)
				}
			}
		})
	}
}

func requireStarlingErrorEqual(t *testing.T, expectedErr starling.Error, expectedErrMsg string, err error) {
	t.Helper()

	require.Error(t, err)

	var starlingErr *starling.Error
	ok := errors.As(err, &starlingErr)
	require.True(t, ok)

	require.Equal(t, expectedErr.Code, starlingErr.Code)
	require.Equal(t, expectedErr.Message, starlingErr.Message)
	require.Equal(t, expectedErr.ErrorMessages, starlingErr.ErrorMessages)
	require.Equal(t, expectedErrMsg, starlingErr.Error())
}
