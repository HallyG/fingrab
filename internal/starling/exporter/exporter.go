package exporter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/starling"
	"github.com/HallyG/fingrab/internal/util/sliceutil"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	ExportTypeStarling   = export.ExportType("Starling")
	Starling             = "Starling"
	starlingTimeFormat   = "2006-01-02"
	starlingMaxDateRange = 0
)

var _ export.Exporter = (*TransactionExporter)(nil)

type TransactionExporter struct {
	api starling.Client
}

func New(api starling.Client) (*TransactionExporter, error) {
	if api == nil {
		return nil, errors.New("starling client is required")
	}
	return &TransactionExporter{
		api: api,
	}, nil
}

func (s *TransactionExporter) Type() export.ExportType {
	return ExportTypeStarling
}

func (s *TransactionExporter) MaxDateRange() time.Duration {
	return starlingMaxDateRange
}

func (s *TransactionExporter) ExportTransactions(ctx context.Context, opts export.Options) ([]*domain.Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid opts: %w", err)
	}

	accountID := starling.AccountID(uuid.Nil)

	if opts.AccountID != "" {
		uuid, err := uuid.Parse(opts.AccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account id: %w", err)
		}

		accountID = starling.AccountID(uuid)
	}

	zerolog.Ctx(ctx).Info().
		Str("bank", Starling).
		Str("export.start", opts.StartDate.Format(starlingTimeFormat)).
		Str("export.end", opts.EndDate.Format(starlingTimeFormat)).
		Msg("starting export of Starling transactions")

	account, err := s.fetchAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	categoryID := account.DefaultCategoryID

	transactions, err := s.fetchTransactionsSince(ctx, account.ID, categoryID, opts.StartDate, opts.EndDate)
	if err != nil {
		return nil, err
	}

	zerolog.Ctx(ctx).Info().
		Str("bank", Starling).
		Int("transaction.count", len(transactions)).
		Msg("successfully exported Starling transactions")

	return sliceutil.Map(transactions, func(txn *starling.FeedItem) *domain.Transaction {
		reference := s.determineReference(txn)

		depositSignum := int64(-1)
		isDeposit := starling.DirectionIN == txn.Direction

		if isDeposit {
			depositSignum = 1
		}

		return &domain.Transaction{
			Amount: domain.Money{
				MinorUnit: txn.Amount.MinorUnit * depositSignum,
				Currency:  txn.Amount.Currency,
			},
			Reference: reference,
			Category:  txn.CategoryName,
			CreatedAt: txn.TransactedAt,
			IsDeposit: isDeposit,
			BankName:  Starling,
			Notes:     txn.UserNote,
		}
	}), nil
}

func (s *TransactionExporter) fetchAccount(ctx context.Context, accountID starling.AccountID) (*starling.Account, error) {
	accounts, err := s.api.FetchAccounts(ctx)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no accounts found, exiting")
	}

	selectedAccount := accounts[0]
	accountsArr := zerolog.Arr()

	accountIDs := make([]starling.AccountID, 0)

	for _, account := range accounts {
		if accountID == account.ID {
			selectedAccount = account
		}

		accountIDs = append(accountIDs, account.ID)
		accountsArr.Str(account.ID.String())
	}

	zerolog.Ctx(ctx).Debug().
		Str("bank", Starling).
		Int("account.total", len(accountIDs)).
		Array("account.all", accountsArr).
		Msg("found Starling accounts")

	zerolog.Ctx(ctx).Info().
		Str("bank", Starling).
		Str("account.id", selectedAccount.ID.String()).
		Str("category.id", selectedAccount.DefaultCategoryID.String()).
		Msg("selected Starling account")

	return selectedAccount, nil
}

func (s *TransactionExporter) fetchTransactionsSince(ctx context.Context, accountID starling.AccountID, categoryID starling.CategoryID, start time.Time, end time.Time) ([]*starling.FeedItem, error) {
	zerolog.Ctx(ctx).Info().
		Str("bank", Starling).
		Str("account.id", accountID.String()).
		Str("category.id", categoryID.String()).
		Time("start", start).
		Time("end", end).
		Msg("fetching Starling transactions")

	transactions, err := s.api.FetchTransactionsSince(ctx, starling.FetchTransactionOptions{
		AccountID:  accountID,
		CategoryID: categoryID,
		Start:      start,
		End:        end,
	})
	if err != nil {
		return nil, err
	}

	transactionsWithRoundUp := sliceutil.Filter(transactions, func(txn *starling.FeedItem) bool {
		return txn.RoundUp != nil
	})

	roundUpTransactions, err := s.fetchRoundUpTransactions(ctx, accountID, start, end, transactionsWithRoundUp)
	if err != nil {
		return nil, err
	}

	transactions = append(transactions, roundUpTransactions...)
	filteredTransactions := sliceutil.Filter(transactions, func(txn *starling.FeedItem) bool {
		return txn.Status != starling.StatusDeclined
	})

	zerolog.Ctx(ctx).Info().
		Str("bank", Starling).
		Str("account.id", accountID.String()).
		Str("category.id", categoryID.String()).
		Int("transaction.total", len(filteredTransactions)).
		Msg("fetched Starling transactions")

	return filteredTransactions, nil
}

func (s *TransactionExporter) determineReference(txn *starling.FeedItem) string {
	if txn.CategoryName == "TRANSFERS" && txn.CounterPartyType == "CATEGORY" && txn.Source == "INTERNAL_TRANSFER" && txn.SourceSubType == "" {
		return "Savings Pot"
	}

	// Interest
	if txn.CategoryName == "INCOME" && txn.CounterPartyType == "STARLING" && txn.Source == "INTEREST_PAYMENT" && txn.SourceSubType == "DEPOSIT" {
		return "Interest Capitalisation"
	}

	// Merchant
	if txn.CounterPartyName != "" && txn.CounterPartyType == "MERCHANT" {
		return txn.CounterPartyName
	}

	// Sender
	if txn.CounterPartyName != "" && txn.CounterPartyType == "SENDER" {
		return fmt.Sprintf("%s (%s)", txn.Description, txn.CounterPartyName)
	}

	return strings.TrimSpace(txn.Description)
}

func (s *TransactionExporter) fetchRoundUpTransactions(ctx context.Context, accountID starling.AccountID, start time.Time, end time.Time, transactionsWithRoundUp []*starling.FeedItem) ([]*starling.FeedItem, error) {
	seenCategoryIDs := make(map[starling.CategoryID]struct{})
	roundUpTransactions := make([]*starling.FeedItem, 0)

	for _, txn := range transactionsWithRoundUp {
		if txn.RoundUp == nil {
			continue
		}

		_, ok := seenCategoryIDs[txn.RoundUp.GoalCategoryID]
		if ok {
			continue
		}

		// Fetch related round-ups
		categoryTransactions, err := s.api.FetchTransactionsSince(ctx, starling.FetchTransactionOptions{
			AccountID:  accountID,
			CategoryID: txn.RoundUp.GoalCategoryID,
			Start:      start,
			End:        end,
		})
		if err != nil {
			return nil, err
		}

		seenCategoryIDs[txn.RoundUp.GoalCategoryID] = struct{}{}
		roundUps := sliceutil.Filter(categoryTransactions, func(item *starling.FeedItem) bool {
			return item.CounterPartyID == starling.CounterPartyID(txn.CategoryID)
		})

		for _, roundUp := range roundUps {
			roundUp.Direction = starling.DirectionOUT // a round-up moves an amount to another category, so we switch the direction
			roundUpTransactions = append(roundUpTransactions, roundUp)
		}
	}

	return roundUpTransactions, nil
}
