package monzo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/HallyG/fingrab/internal/api"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"resty.dev/v3"
)

const (
	prodAPI              = "https://api.monzo.com"
	getAccountsRoute     = "/accounts"
	getPotsRoute         = "/pots"
	getTransactionsRoute = "/transactions"
	getTransactionRoute  = getTransactionsRoute + "/%s"
	maxResultPerPage     = 100
)

type Client interface {
	FetchAccounts(ctx context.Context) ([]*Account, error)
	FetchPots(ctx context.Context, accountID AccountID) ([]*Pot, error)
	FetchTransaction(ctx context.Context, transactionID TransactionID) (*Transaction, error)
	FetchTransactionsSince(ctx context.Context, opts FetchTransactionOptions) ([]*Transaction, error)
}

var _ Client = (*client)(nil)

type client struct {
	api *resty.Client
}

type Option func(*client)

func New(httpClient *http.Client, opts ...Option) *client {
	c := &client{
		api: api.New(
			prodAPI,
			httpClient,
			api.WithError[Error](),
		),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(c)
	}

	return c
}

func WithAuthToken(authToken string) Option {
	return func(c *client) {
		api.WithAuthToken(authToken)(c.api)
	}
}

func WithBaseURL(baseURL string) Option {
	return func(c *client) {
		api.WithBaseURL(baseURL)(c.api)
	}
}

func (c *client) FetchAccounts(ctx context.Context) ([]*Account, error) {
	result, err := api.ExecuteRequest[struct {
		Accounts []*Account `json:"accounts"`
	}](ctx, c.api, http.MethodGet, getAccountsRoute, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("failed to accounts: %w", err)
	}

	return result.Accounts, nil
}

func (c *client) FetchPots(ctx context.Context, accountID AccountID) ([]*Pot, error) {
	values := url.Values{}
	values.Add("current_account_id", string(accountID))

	result, err := api.ExecuteRequest[struct {
		Pots []*Pot `json:"pots"`
	}](ctx, c.api, http.MethodGet, getPotsRoute, values)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pots: %w", err)
	}

	return result.Pots, nil
}

func (c *client) FetchTransaction(ctx context.Context, transactionID TransactionID) (*Transaction, error) {
	result, err := api.ExecuteRequest[struct {
		Transaction *Transaction `json:"transaction"`
	}](ctx, c.api, http.MethodGet, fmt.Sprintf(getTransactionRoute, string(transactionID)), url.Values{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return result.Transaction, nil
}

func (c *client) FetchTransactionsSince(ctx context.Context, opts FetchTransactionOptions) ([]*Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	values := url.Values{}
	values.Add("account_id", string(opts.AccountID))
	values.Add("expand[]", "merchant")

	if opts.Limit == 0 {
		opts.Limit = maxResultPerPage
	}

	if opts.Limit != 0 {
		values.Add("limit", strconv.Itoa(opts.Limit))
	}

	if !opts.End.IsZero() {
		values.Add("before", opts.End.Format(time.RFC3339))
	}

	if opts.SinceID == "" {
		values.Add("since", opts.Start.Format(time.RFC3339))
	} else {
		values.Add("since", string(opts.SinceID))
	}

	result, err := api.ExecuteRequest[struct {
		Transactions []*Transaction `json:"transactions"`
	}](ctx, c.api, http.MethodGet, getTransactionsRoute, values)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return result.Transactions, nil
}

type FetchTransactionOptions struct {
	AccountID AccountID
	Start     time.Time
	End       time.Time
	SinceID   TransactionID
	Limit     int
}

func (fto FetchTransactionOptions) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &fto,
		validation.Field(&fto.AccountID, validation.Required.Error("account ID is required")),
		validation.Field(&fto.Limit, validation.Min(0).Error("limit must be non-negative")),
		validation.Field(&fto.Start, validation.When(!fto.End.IsZero(), validation.By(func(value any) error {
			start, ok := value.(time.Time)
			if !ok {
				return validation.NewError("validation_invalid_type", "start time must be a valid time")
			}

			if !start.Before(fto.End) {
				return validation.NewError("validation_invalid_time_range", "start time must be before end time")
			}

			return nil
		}))),
	)
}
