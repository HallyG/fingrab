package monzo

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/HallyG/fingrab/internal/api"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	resty "resty.dev/v3"
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
	api *api.BaseClient
}

type Option func(*client)

func New(httpClient *http.Client, opts ...Option) *client {
	c := &client{
		api: api.New(
			prodAPI,
			httpClient,
			api.WithErrorUnmarshaller(func(r *resty.Response) error {
				return UnmarshalError(r.StatusCode(), r.Bytes())
			}),
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
	var result struct {
		Accounts []*Account `json:"accounts"`
	}

	_, err := c.api.ExecuteRequest(ctx, http.MethodGet, getAccountsRoute, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	return result.Accounts, nil
}

func (c *client) FetchPots(ctx context.Context, accountID AccountID) ([]*Pot, error) {
	var result struct {
		Pots []*Pot `json:"pots"`
	}

	queryParams := map[string]string{
		"current_account_id": string(accountID),
	}

	_, err := c.api.ExecuteRequest(ctx, http.MethodGet, getPotsRoute, queryParams, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pots: %w", err)
	}

	return result.Pots, nil
}

func (c *client) FetchTransaction(ctx context.Context, transactionID TransactionID) (*Transaction, error) {
	var result struct {
		Transaction *Transaction `json:"transaction"`
	}

	_, err := c.api.ExecuteRequest(ctx, http.MethodGet, fmt.Sprintf(getTransactionRoute, string(transactionID)), nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	return result.Transaction, nil
}

type FetchTransactionOptions struct {
	AccountID AccountID
	Start     time.Time
	End       time.Time
	SinceID   TransactionID
	Limit     int
}

func (fto *FetchTransactionOptions) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, fto,
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

func (c *client) FetchTransactionsSince(ctx context.Context, opts FetchTransactionOptions) ([]*Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	var result struct {
		Transactions []*Transaction `json:"transactions"`
	}

	queryParams := map[string]string{
		"account_id": string(opts.AccountID),
		"expand[]":   "merchant",
	}

	if opts.Limit == 0 {
		opts.Limit = maxResultPerPage
	}

	if opts.Limit != 0 {
		queryParams["limit"] = strconv.Itoa(opts.Limit)
	}

	if !opts.End.IsZero() {
		queryParams["before"] = opts.End.Format(time.RFC3339)
	}

	if opts.SinceID == "" {
		queryParams["since"] = opts.Start.Format(time.RFC3339)
	} else {
		queryParams["since"] = string(opts.SinceID)
	}

	_, err := c.api.ExecuteRequest(ctx, http.MethodGet, getTransactionsRoute, queryParams, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return result.Transactions, nil
}
