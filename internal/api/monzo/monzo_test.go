package monzo_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/HallyG/fingrab/internal/api/monzo"
	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/util/testutil"
	"github.com/stretchr/testify/require"
)

const (
	token         = "mock-token"
	accountId     = monzo.AccountID("acc_56789")
	transactionId = monzo.TransactionID("tx_000099999999")
	potId         = monzo.PotID("pot_00009")
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		api := monzo.New(nil, nil)
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

			route := testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    "/accounts",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "accounts.json")(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := monzo.New(&http.Client{},
				monzo.WithBaseURL(server.URL),
				monzo.WithAuthToken(token),
			)

			accounts, err := client.FetchAccounts(t.Context())
			require.NoError(t, err, "failed to fetch accounts")
			require.Len(t, accounts, test.expectedLength, "unexpected number of accounts")

			if len(accounts) > 0 {
				account := accounts[0]
				require.Equal(t, monzo.AccountID("acc_56789"), account.ID, "account ID should match")
				require.Equal(t, "user_123456", account.Description, "description should match")
				require.Equal(t, "2020-02-02T02:02:22Z", account.CreatedAt.Format(time.RFC3339), "created at time should match")
				require.False(t, account.Closed, "closed flag should match")
				require.Equal(t, "GBP", account.Currency, "currency should match")
				require.Equal(t, "uk_retail", account.Type, "type should match")
				require.Equal(t, "personal", account.OwnerType, "owner type should match")
				require.Equal(t, "GB", account.CountryCode, "country code should match")
				require.Equal(t, "GBR", account.CountryCodeAlpha3, "country code alpha 3 should match")
				require.Equal(t, "12345678", account.AccountNumber, "account number should match")
				require.Equal(t, "040004", account.SortCode, "sort code should match")
			}
		})
	}

	t.Run("returns API error", func(t *testing.T) {
		t.Parallel()

		route := testutil.HTTPTestRoute{
			Method: http.MethodGet,
			URL:    "/accounts",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertRequest(t, r, http.MethodGet, nil, nil)
				testutil.ServeJSONTestDataHandler(t, http.StatusUnauthorized, "error.json")(w, r)
			},
		}

		server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
		client := monzo.New(&http.Client{},
			monzo.WithBaseURL(server.URL),
			monzo.WithAuthToken(token),
		)

		ctx := t.Context()
		accounts, err := client.FetchAccounts(ctx)

		require.Error(t, err)
		require.Nil(t, accounts)
		require.Contains(t, err.Error(), "/a not found (http status=401)")

		var monzoErr monzo.Error
		ok := errors.As(errors.Unwrap(err), &monzoErr)
		require.True(t, ok)
		require.Equal(t, "not_found", monzoErr.Code, "code should match")
		require.Equal(t, http.StatusUnauthorized, monzoErr.HTTPStatus, "http status should match")
		require.Equal(t, "/a not found", monzoErr.Message, "message should match")
	})
}

func TestFetchTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
	}{
		{
			name:                "successful fetch",
			expectedQueryParams: map[string]string{},
			expectedHeaders: map[string]string{
				"Authorization": token,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			route := testutil.HTTPTestRoute{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("/transactions/%s", transactionId),
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "transaction.json")(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := monzo.New(&http.Client{},
				monzo.WithBaseURL(server.URL),
				monzo.WithAuthToken(token),
			)

			txn, err := client.FetchTransaction(t.Context(), transactionId)
			require.NoError(t, err, "failed to fetch transaction")

			require.NotNil(t, txn)
			require.Equal(t, monzo.TransactionID("tx_000099999999"), txn.ID, "transaction ID should match")
			require.Equal(t, monzo.AccountID("acc_0000912345"), txn.AccountID, "account ID should match")
			require.Equal(t, "TfL Travel Charge      TFL.gov.uk/CP GBR", txn.Description, "description should match")
			require.Equal(t, domain.Money{MinorUnit: -280, Currency: "GBP"}, txn.Amount, "amount should match")
			require.Equal(t, domain.Money{MinorUnit: -280, Currency: "GBP"}, txn.LocalAmount, "local amount should match")
			require.Equal(t, "Travel charge for Friday, 24 Jan", txn.UserNotes, "user notes should match")
			require.Equal(t, "transport", txn.CategoryName, "category name should match")
			require.Equal(t, "2025-01-25T10:00:00Z", txn.CreatedAt.Format(time.RFC3339), "created at time should match")
			require.Equal(t, "2025-01-25T11:00:00Z", txn.SettledAt.Format(time.RFC3339), "settled at time should match")
			require.Equal(t, "2025-01-25T12:00:00Z", txn.UpdatedAt.Format(time.RFC3339), "updated at time should match")
			require.False(t, txn.AmountIsPending, "amount is pending should be false")
			require.Equal(t, "mastercard", txn.Scheme, "scheme should match")
			require.NotNil(t, txn.Merchant, "merchant should not be nil")
			require.Nil(t, txn.CounterParty, "counter party should be nil")
		})
	}
}

func TestFetchPots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
		expectedLength      int
	}{
		{
			name: "successful fetch",
			expectedQueryParams: map[string]string{
				"current_account_id": string(accountId),
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
				URL:    "/pots",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "pots.json")(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := monzo.New(&http.Client{},
				monzo.WithBaseURL(server.URL),
				monzo.WithAuthToken(token),
			)

			pots, err := client.FetchPots(t.Context(), accountId)
			require.NoError(t, err, "failed to fetch pots")
			require.Len(t, pots, test.expectedLength, "unexpected number of pots")

			if len(pots) > 0 {
				pot := pots[0]
				require.Equal(t, monzo.PotID("pot_00009"), pot.ID, "pot ID should match")
				require.True(t, pot.Deleted, "deleted flag should match")
				require.Equal(t, "holiday", pot.Name, "name should match")
				require.Equal(t, "GBP", pot.Currency, "currency should match")
			}
		})
	}
}

func TestFetchTransactions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		opts                monzo.FetchTransactionOptions
		expectedQueryParams map[string]string
		expectedHeaders     map[string]string
		expectedLength      int
	}{
		{
			name: "successful fetch",
			opts: monzo.FetchTransactionOptions{
				AccountID: accountId,
			},
			expectedQueryParams: map[string]string{
				"account_id": string(accountId),
				"expand[]":   "merchant",
				"limit":      "100",
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
				URL:    "/transactions",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertRequest(t, r, http.MethodGet, test.expectedHeaders, test.expectedQueryParams)
					testutil.ServeJSONTestDataHandler(t, http.StatusOK, "transactions.json")(w, r)
				},
			}

			server := testutil.NewHTTPTestServer(t, []testutil.HTTPTestRoute{route})
			client := monzo.New(&http.Client{},
				monzo.WithBaseURL(server.URL),
				monzo.WithAuthToken(token),
			)

			txns, err := client.FetchTransactionsSince(t.Context(), test.opts)
			require.NoError(t, err, "failed to fetch transactions")
			require.Len(t, txns, test.expectedLength, "unexpected number of transactions")

			if len(txns) > 0 {
				require.Equal(t, transactionId, txns[0].ID, "transaction ID should match")
			}
		})
	}

	t.Run("returns error when invalid options", func(t *testing.T) {
		t.Parallel()

		client := monzo.New(&http.Client{},
			monzo.WithBaseURL(""),
			monzo.WithAuthToken(token),
		)

		now := time.Now()
		_, err := client.FetchTransactionsSince(t.Context(), monzo.FetchTransactionOptions{
			End:       now.Add(-24 * time.Hour),
			Start:     now,
			AccountID: "acc_12345",
		})
		require.ErrorContains(t, err, "start time must be before end time")
	})
}
