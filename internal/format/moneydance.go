package format

import (
	"io"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
)

const (
	moneyDanceTimeFormat            = "2006-01-02" // date format expected by MoneyDance (YYYY-MM-DD)
	FormatTypeMoneyDance FormatType = "moneydance"
)

func init() {
	register(FormatTypeMoneyDance, func(w io.Writer, location *time.Location) (Formatter, error) {
		return &MoneyDanceFormatter{
			CSVFormatter: NewCSVFormatter(w),
			location:     location,
		}, nil
	})
}

// MoneyDanceFormatter formats transactions for import into MoneyDance.
// It outputs CSV with columns: check number, date, description, category, amount, memo.
// Transactions are marked as "Trn" and deposits as "Dep" in the check number field.
type MoneyDanceFormatter struct {
	*CSVFormatter
	location *time.Location
}

func (m *MoneyDanceFormatter) WriteHeader() error {
	return m.writer.Write([]string{"check number", "date", "description", "category", "amount", "memo"})
}

func (m *MoneyDanceFormatter) WriteTransaction(t *domain.Transaction) error {
	checkNumber := "Trn"
	if t.IsDeposit {
		checkNumber = "Dep"
	}

	return m.writer.Write([]string{
		checkNumber,
		t.CreatedAt.In(m.location).Format(moneyDanceTimeFormat),
		t.Reference,
		t.Category,
		t.Amount.String(),
		t.Notes,
	})
}
