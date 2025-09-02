package monzo_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/api/monzo"
	"github.com/HallyG/fingrab/internal/util/testutil"
	"github.com/stretchr/testify/require"
)

const (
	token         = "mock-token"
	accountId     = monzo.AccountID("acc_56789")
	transactionId = monzo.TransactionID("tx_000099999999")
	potId         = monzo.PotID("pot_00009")
)

func setup(t *testing.T, routes ...testutil.HTTPTestRoute) monzo.Client {
	t.Helper()

	server := testutil.NewHTTPTestServer(t, routes)
	client := monzo.New(&http.Client{},
		monzo.WithBaseURL(server.URL),
		monzo.WithAuthToken(token),
	)

	return client
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		api := monzo.New(nil, nil)
		require.NotNil(t, api)
	})
}

func TestFetchAccounts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		route            testutil.HTTPTestRoute
		expectedAccounts []*monzo.Account
		expectedErr      *monzo.Error
		assertFn         func(t *testing.T, items []*monzo.Account)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "accounts.json")(w, r)
				},
			},
			expectedAccounts: testutil.MarshalTestDataFile[struct {
				Accounts []*monzo.Account `json:"accounts"`
			}](t, "accounts.json").Accounts,
			assertFn: func(t *testing.T, items []*monzo.Account) {
				t.Helper()

				require.Len(t, items, 1)
				require.Equal(t, accountId, items[0].ID)
			},
		},
		"returns API error": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, nil, nil)
					testutil.ServeJSONTestDataHandler(t, http.StatusUnauthorized, "error.json")(w, r)
				},
			},
			expectedErr: &monzo.Error{
				Code:    "not_found",
				Message: "/a not found",
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)
			accounts, err := client.FetchAccounts(t.Context())

			if test.expectedErr != nil {
				require.Empty(t, accounts)
				requireErrorEqual(t, *test.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, accounts, test.expectedAccounts)
			}
		})
	}
}

func TestFetchTransaction(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		route        testutil.HTTPTestRoute
		expectedItem *monzo.Transaction
		expectedErr  *monzo.Error
		assertFn     func(t *testing.T, item *monzo.Transaction)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/transactions/%s", transactionId),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "transaction.json")(w, r)
				},
			},
			expectedItem: testutil.MarshalTestDataFile[struct {
				Transaction *monzo.Transaction `json:"transaction"`
			}](t, "transaction.json").Transaction,
			assertFn: func(t *testing.T, item *monzo.Transaction) {
				t.Helper()

				require.Equal(t, transactionId, item.ID)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)
			item, err := client.FetchTransaction(t.Context(), transactionId)

			if test.expectedErr != nil {
				require.Nil(t, item)
				require.ErrorContains(t, err, test.expectedErr.Error())
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

func TestFetchPots(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		route        testutil.HTTPTestRoute
		expectedPots []*monzo.Pot
		expectedErr  *monzo.Error
		assertFn     func(t *testing.T, items []*monzo.Pot)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/pots",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)
					query.Add("current_account_id", string(accountId))

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "pots.json")(w, r)
				},
			},
			expectedPots: testutil.MarshalTestDataFile[struct {
				Pots []*monzo.Pot `json:"pots"`
			}](t, "pots.json").Pots,
			assertFn: func(t *testing.T, items []*monzo.Pot) {
				t.Helper()

				require.Len(t, items, 1)
				require.Equal(t, potId, items[0].ID)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)
			items, err := client.FetchPots(t.Context(), accountId)

			if test.expectedErr != nil {
				require.Empty(t, items)
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, items, test.expectedPots)
				if test.assertFn != nil {
					test.assertFn(t, items)
				}
			}
		})
	}
}

func TestFetchTransactions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name          string
		route         testutil.HTTPTestRoute
		opts          monzo.FetchTransactionOptions
		expectedItems []*monzo.Transaction
		expectedErr   error
		assertFn      func(t *testing.T, items []*monzo.Transaction)
	}{
		"successful fetch": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/transactions",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}
					header.Add("Authorization", token)
					query.Add("account_id", string(accountId))
					query.Add("expand[]", "merchant")
					query.Add("limit", "100")

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "transactions.json")(w, r)
				},
			},
			opts: monzo.FetchTransactionOptions{
				AccountID: accountId,
			},
			expectedItems: testutil.MarshalTestDataFile[struct {
				Transactions []*monzo.Transaction `json:"transactions"`
			}](t, "transactions.json").Transactions,
			assertFn: func(t *testing.T, items []*monzo.Transaction) {
				t.Helper()

				require.Len(t, items, 1)
				require.Equal(t, transactionId, items[0].ID)
			},
		},
		"returns error when invalid options": {
			route: testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/transactions",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					header := http.Header{}
					query := url.Values{}

					testutil.AssertRequest(t, r, http.MethodGet, header, query)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "transactions.json")(w, r)
				},
			},
			opts: monzo.FetchTransactionOptions{
				End:       time.Now().Add(-24 * time.Hour),
				Start:     time.Now(),
				AccountID: "acc_12345",
			},
			expectedErr: errors.New("start time must be before end time"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := setup(t, test.route)

			items, err := client.FetchTransactionsSince(t.Context(), test.opts)
			if test.expectedErr != nil {
				require.Empty(t, items)
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, items, test.expectedItems)
				if test.assertFn != nil {
					test.assertFn(t, items)
				}
			}
		})
	}
}

func requireErrorEqual(t *testing.T, expectedErr monzo.Error, err error) {
	t.Helper()

	require.Error(t, err)

	var monzoErr *monzo.Error
	ok := errors.As(err, &monzoErr)
	require.True(t, ok)

	require.Equal(t, expectedErr.Code, monzoErr.Code)
	require.Equal(t, expectedErr.Message, monzoErr.Message)
}
