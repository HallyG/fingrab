package exporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/log"
	"github.com/HallyG/fingrab/internal/monzo"
	"github.com/samber/lo"
)

const (
	Monzo                 = "Monzo"
	ExportTypeMonzo       = export.ExportType(Monzo)
	monzoTransactionBatch = uint16(100)
	monzoTimeFormat       = "2006-01-02"
	monzoMaxDateRange     = 90 * 24 * time.Hour
)

var _ export.Exporter = (*TransactionExporter)(nil)

type TransactionExporter struct {
	api monzo.Client
}

func New(api monzo.Client) (*TransactionExporter, error) {
	if api == nil {
		return nil, errors.New("monzo client is required")
	}
	return &TransactionExporter{
		api: api,
	}, nil
}

func (m *TransactionExporter) Type() export.ExportType {
	return ExportTypeMonzo
}

func (m *TransactionExporter) MaxDateRange() time.Duration {
	return monzoMaxDateRange
}

func (m *TransactionExporter) ExportTransactions(ctx context.Context, opts export.Options) ([]*domain.Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	log.FromContext(ctx).InfoContext(ctx, "starting export of transactions",
		slog.String("export.start", opts.StartDate.Format(monzoTimeFormat)),
		slog.String("export.end", opts.EndDate.Format(monzoTimeFormat)),
	)

	account, err := m.fetchAccount(ctx, opts.AccountID)
	if err != nil {
		return nil, err
	}

	transactions, err := m.fetchTransactions(ctx, account.ID, opts.StartDate, opts.EndDate)
	if err != nil {
		return nil, err
	}

	// todo: move into fetch
	err = m.enrichTransactionDescriptions(ctx, account.ID, transactions)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).InfoContext(ctx, "successfully exported transactions",
		slog.Int("transaction.count", len(transactions)),
	)

	return lo.Map(transactions, func(txn *monzo.Transaction, _ int) *domain.Transaction {
		reference, notes := m.determineReference(txn)
		if txn.UserNotes != "" {
			notes = txn.UserNotes
		}

		return &domain.Transaction{
			Amount:    txn.Amount,
			Reference: reference,
			Category:  txn.CategoryName,
			CreatedAt: txn.CreatedAt,
			IsDeposit: txn.LocalAmount.MinorUnit > 0,
			BankName:  Monzo,
			Notes:     notes,
		}
	}), nil
}

func (m *TransactionExporter) fetchAccount(ctx context.Context, accountID string) (*monzo.Account, error) {
	accounts, err := m.api.FetchAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, errors.New("no accounts found, exiting")
	}

	selectedAccount := accounts[0]
	accountIDs := make([]monzo.AccountID, 0)
	for _, account := range accounts {
		if accountID == string(account.ID) {
			selectedAccount = account
		}

		accountIDs = append(accountIDs, account.ID)
	}

	log.FromContext(ctx).InfoContext(ctx, "found accounts",
		slog.Int("account.total", len(accountIDs)),
	)

	log.FromContext(ctx).InfoContext(ctx, "selected account",
		slog.String("account.id", string(selectedAccount.ID)),
	)

	return selectedAccount, nil
}

func (m *TransactionExporter) fetchTransactions(ctx context.Context, accountID monzo.AccountID, startDate time.Time, endDate time.Time) ([]*monzo.Transaction, error) {
	var transactions []*monzo.Transaction
	var sinceID monzo.TransactionID

	endDateExclusive := endDate.AddDate(0, 0, 1)
	limit := monzoTransactionBatch

	log.FromContext(ctx).InfoContext(ctx, "fetching transactions",
		slog.String("account.id", string(accountID)),
		slog.String("start", startDate.Format(monzoTimeFormat)),
		slog.String("end", endDate.Format(monzoTimeFormat)),
		slog.Int("limit", int(limit)),
	)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("fetch transactions: %w", ctx.Err())
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
			return nil, fmt.Errorf("fetch transactions: %w", err)
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
			inDesiredDateRange := endDate.IsZero() || !transaction.CreatedAt.After(endDateExclusive)

			if inDesiredDateRange && isNotDeclined {
				transactions = append(transactions, transaction)
			}
		}

		sinceID = latest.ID

		// Monzo doesn't support cursor AND time pagination
		if latest.CreatedAt.After(endDateExclusive) {
			break
		}
	}

	log.FromContext(ctx).InfoContext(ctx, "fetched transactions",
		slog.String("account.id", string(accountID)),
		slog.Int("transaction.total", len(transactions)),
	)

	return transactions, nil
}

// Enrich Transaction Descriptions with Pot Name.
func (m *TransactionExporter) enrichTransactionDescriptions(ctx context.Context, accountID monzo.AccountID, transactions []*monzo.Transaction) error {
	log.FromContext(ctx).DebugContext(ctx, "enriching transaction descriptions",
		slog.String("account.id", string(accountID)),
	)

	pots, err := m.api.FetchPots(ctx, accountID)
	if err != nil {
		return fmt.Errorf("fetch pots: %w", err)
	}

	log.FromContext(ctx).DebugContext(ctx, "fetched pots",
		slog.String("account.id", string(accountID)),
		slog.Int("pots.total", len(pots)),
	)

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

func (m *TransactionExporter) determineReference(txn *monzo.Transaction) (string, string) {
	reference := txn.Description
	notes := ""

	switch {
	// When splitting payment with someone
	case txn.CounterParty != nil && txn.CounterParty.Name != "" && txn.Merchant != nil && txn.Merchant.Name != "":
		reference = txn.CounterParty.Name
		notes = txn.Merchant.Name
	case txn.CounterParty != nil && txn.CounterParty.Name != "":
		reference = txn.CounterParty.Name
	case txn.Merchant != nil && txn.Merchant.Name != "":
		reference = txn.Merchant.Name
	default:
	}

	return strings.TrimSpace(reference), strings.TrimSpace(notes)
}
