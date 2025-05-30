package export

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/api/monzo"
	"github.com/HallyG/fingrab/internal/domain"
	"github.com/rs/zerolog"
)

const (
	ExportTypeMonzo       = ExportType("Monzo")
	Monzo                 = "Monzo"
	monzoTransactionBatch = 100
	monzoTimeFormat       = "2006-01-02"
	monzoMaxDateRange     = 90 * 24 * time.Hour // 90 days
)

func init() {
	Register(ExportTypeMonzo, func(opts Options) (Exporter, error) {
		client := &http.Client{
			Timeout: opts.Timeout,
		}

		api := monzo.New(client, monzo.WithAuthToken(opts.BearerAuthToken()))

		return NewMonzoTransactionExporter(api), nil
	})
}

var _ Exporter = (*monzoTransactionExporter)(nil)

type monzoTransactionExporter struct {
	api monzo.Client
}

func NewMonzoTransactionExporter(api monzo.Client) *monzoTransactionExporter {
	return &monzoTransactionExporter{
		api: api,
	}
}

func (m *monzoTransactionExporter) Type() ExportType {
	return ExportTypeMonzo
}

func (m *monzoTransactionExporter) MaxDateRange() time.Duration {
	return monzoMaxDateRange
}

func (m *monzoTransactionExporter) ExportTransactions(ctx context.Context, opts Options) ([]*domain.Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid opts: %w", err)
	}

	zerolog.Ctx(ctx).Info().
		Str("bank", Monzo).
		Str("export.start", opts.StartDate.Format(monzoTimeFormat)).
		Str("export.end", opts.EndDate.Format(monzoTimeFormat)).
		Msg("starting export of Monzo transactions")

	account, err := m.fetchAccount(ctx, opts.AccountID)
	if err != nil {
		return nil, err
	}

	transactions, err := m.fetchTransactions(ctx, account.ID, opts.StartDate, opts.EndDate)
	if err != nil {
		return nil, err
	}

	err = m.enrichTransactionDescriptions(ctx, account.ID, transactions)
	if err != nil {
		return nil, err
	}

	zerolog.Ctx(ctx).Info().
		Str("bank", Monzo).
		Int("transaction.count", len(transactions)).
		Msg("successfully exported Monzo transactions")

	return m.ToDomainTransactions(transactions)
}

func (m *monzoTransactionExporter) ToDomainTransactions(monzoTxns []*monzo.Transaction) ([]*domain.Transaction, error) {
	domainTxns := make([]*domain.Transaction, 0)

	for _, txn := range monzoTxns {
		domainTxn := &domain.Transaction{
			Amount:    txn.Amount,
			Reference: m.determineReference(txn),
			Category:  txn.CategoryName,
			CreatedAt: txn.CreatedAt,
			IsDeposit: txn.LocalAmount.MinorUnit > 0,
			BankName:  Monzo,
			Notes:     txn.UserNotes,
		}

		domainTxns = append(domainTxns, domainTxn)
	}

	return domainTxns, nil
}

// Enrich Transaction Descriptions with Pot Name.
func (m *monzoTransactionExporter) enrichTransactionDescriptions(ctx context.Context, accountID monzo.AccountID, transactions []*monzo.Transaction) error {
	zerolog.Ctx(ctx).Debug().
		Ctx(ctx).
		Str("bank", Monzo).
		Str("account.id", string(accountID)).
		Msg("enriching transaction descriptions")

	pots, err := m.api.FetchPots(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to fetch pots while enriching transactions: %w", err)
	}

	zerolog.Ctx(ctx).Debug().
		Ctx(ctx).
		Str("bank", Monzo).
		Str("account.id", string(accountID)).
		Int("pots.total", len(pots)).
		Msg("fetched pots")

	potMap := make(map[string]string)
	for _, pot := range pots {
		potMap[string(pot.ID)] = pot.Name
	}

	for _, transaction := range transactions {
		if potName, exists := potMap[transaction.Description]; exists {
			transaction.Description = potName + " Pot"
		}
	}

	return nil
}

func (m *monzoTransactionExporter) fetchAccount(ctx context.Context, accountID string) (*monzo.Account, error) {
	accounts, err := m.api.FetchAccounts(ctx)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no accounts found, exiting")
	}

	accountsArr := zerolog.Arr()
	selectedAccount := accounts[0]

	accountIDs := make([]monzo.AccountID, 0)

	for _, account := range accounts {
		if accountID == string(account.ID) {
			selectedAccount = account
		}

		accountIDs = append(accountIDs, account.ID)
		accountsArr.Str(string(account.ID))
	}

	zerolog.Ctx(ctx).Debug().
		Ctx(ctx).
		Str("bank", Monzo).
		Int("account.total", len(accountIDs)).
		Array("account.all", accountsArr).
		Msg("found Monzo accounts")

	zerolog.Ctx(ctx).Info().
		Ctx(ctx).
		Str("bank", Monzo).
		Str("account.id", string(selectedAccount.ID)).
		Msg("selected Monzo account")

	return selectedAccount, nil
}

func (m *monzoTransactionExporter) fetchTransactions(ctx context.Context, accountID monzo.AccountID, startDate time.Time, endDate time.Time) ([]*monzo.Transaction, error) {
	var transactions []*monzo.Transaction

	var sinceID monzo.TransactionID

	limit := monzoTransactionBatch
	zerolog.Ctx(ctx).Debug().
		Ctx(ctx).
		Str("bank", Monzo).
		Str("account.id", string(accountID)).
		Time("start", startDate).
		Time("end", endDate).
		Int("limit", limit).
		Msg("fetching Monzo transactions")

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to fetch transactions: %w", ctx.Err())
		default:
		}

		transactionDtos, err := m.api.FetchTransactionsSince(ctx, monzo.FetchTransactionOptions{
			AccountID: accountID,
			Start:     startDate,
			End:       endDate,
			SinceID:   sinceID,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}

		if len(transactionDtos) == 0 {
			break
		}

		latest := transactionDtos[0]
		for _, transaction := range transactionDtos {
			if transaction.CreatedAt.After(latest.CreatedAt) {
				latest = transaction
			}

			isActiveCardCheck := transaction.Amount.MinorUnit == 0 && transaction.Metadata["notes"] == "Active card check"
			if isActiveCardCheck {
				continue
			}

			isNotDeclined := transaction.DeclineReason == ""
			inDesiredDateRange := endDate.IsZero() || !transaction.CreatedAt.After(endDate)

			if inDesiredDateRange && isNotDeclined {
				transactions = append(transactions, transaction)
			}
		}

		sinceID = latest.ID

		// Monzo doesn't support cursor AND time pagination
		if latest.CreatedAt.After(endDate) {
			break
		}
	}

	zerolog.Ctx(ctx).Debug().
		Str("bank", Monzo).
		Str("account.id", string(accountID)).
		Int("transaction.total", len(transactions)).
		Msg("fetched Monzo transactions")

	return transactions, nil
}

func (m *monzoTransactionExporter) determineReference(txn *monzo.Transaction) string {
	reference := txn.Description

	switch {
	case txn.Merchant != nil && txn.Merchant.Name != "":
		reference = txn.Merchant.Name
	case txn.CounterParty != nil && txn.CounterParty.Name != "":
		reference = txn.CounterParty.Name
	default:
	}

	return strings.TrimSpace(reference)
}
