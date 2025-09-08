package starling

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/HallyG/fingrab/internal/api"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"resty.dev/v3"
)

const (
	prodAPI              = "https://api.starlingbank.com"
	getAccountsRoute     = "/api/v2/accounts"
	getTransactionsRoute = "/api/v2/feed/account/%s/category/%s/transactions-between"
	getFeedItemRoute     = "/api/v2/feed/account/%s/category/%s/%s"
	getSavingsRoute      = "/api/v2/account/%s/savings-goals"
)

var _ Client = (*client)(nil)

type (
	Client interface {
		FetchTransactionsSince(ctx context.Context, opts FetchTransactionOptions) ([]*FeedItem, error)
		FetchFeedItem(ctx context.Context, accountID AccountID, categoryID CategoryID, feedItemID FeedItemID) (*FeedItem, error)
		FetchAccounts(ctx context.Context) ([]*Account, error)
		FetchSavingsGoals(ctx context.Context, accountID AccountID) ([]*SavingsGoal, error)
	}
	client struct {
		api *resty.Client
	}
)

func New(httpClient *http.Client, opts ...api.Option) *client {
	c := api.New(
		prodAPI,
		httpClient,
		api.WithError[Error](),
	)

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(c)
	}

	return &client{
		api: c,
	}
}

func (c *client) FetchFeedItem(ctx context.Context, accountID AccountID, categoryID CategoryID, feedItemID FeedItemID) (*FeedItem, error) {
	result, err := api.ExecuteRequest[FeedItem](ctx, c.api,
		http.MethodGet,
		fmt.Sprintf(getFeedItemRoute, accountID, categoryID, feedItemID),
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) FetchAccounts(ctx context.Context) ([]*Account, error) {
	result, err := api.ExecuteRequest[struct {
		Accounts []*Account `json:"accounts"`
	}](
		ctx, c.api,
		http.MethodGet,
		getAccountsRoute,
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	return result.Accounts, nil
}

func (c *client) FetchSavingsGoals(ctx context.Context, accountID AccountID) ([]*SavingsGoal, error) {
	result, err := api.ExecuteRequest[struct {
		SavingsGoals []*SavingsGoal `json:"savingsGoalList"`
	}](
		ctx, c.api,
		http.MethodGet,
		fmt.Sprintf(getSavingsRoute, accountID.String()),
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	return result.SavingsGoals, nil
}

func (c *client) FetchTransactionsSince(ctx context.Context, opts FetchTransactionOptions) ([]*FeedItem, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	values := url.Values{}
	values.Add("minTransactionTimestamp", opts.Start.Format(time.RFC3339))
	if !opts.End.IsZero() {
		values.Add("maxTransactionTimestamp", opts.End.Format(time.RFC3339))
	}

	result, err := api.ExecuteRequest[struct {
		FeedItems []*FeedItem `json:"feedItems"`
	}](ctx, c.api,
		http.MethodGet,
		fmt.Sprintf(getTransactionsRoute, opts.AccountID, opts.CategoryID),
		values,
	)
	if err != nil {
		return nil, err
	}

	return result.FeedItems, nil
}

type FetchTransactionOptions struct {
	AccountID  AccountID
	CategoryID CategoryID
	Start      time.Time
	End        time.Time
}

func (fto FetchTransactionOptions) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &fto,
		validation.Field(&fto.AccountID, validation.By(func(value any) error {
			uid, _ := value.(AccountID)
			if uid == AccountID(uuid.Nil) {
				return validation.NewError("validation_required", "is required")
			}

			return nil
		})),
		validation.Field(&fto.CategoryID, validation.By(func(value any) error {
			uid, _ := value.(CategoryID)
			if uid == CategoryID(uuid.Nil) {
				return validation.NewError("validation_required", "is required")
			}

			return nil
		})),
		validation.Field(&fto.End, validation.Required.Error("is required")),
		validation.Field(&fto.Start, validation.When(!fto.End.IsZero(), validation.By(func(value any) error {
			start, _ := value.(time.Time)
			if !start.Before(fto.End) {
				return validation.NewError("validation_invalid_time_range", "must be before End")
			}

			return nil
		}))),
	)
}
