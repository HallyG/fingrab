package format

import (
	"io"

	"github.com/HallyG/fingrab/internal/domain"
)

const (
	moneyDanceTimeFormat            = "2006-01-02"
	FormatTypeMoneyDance FormatType = "moneydance"
)

func init() {
	register(FormatTypeMoneyDance, func(w io.Writer) Formatter {
		return newMoneyDanceFormatter(w)
	})
}

type MoneyDanceFormatter struct {
	*CSVFormatter
}

func newMoneyDanceFormatter(w io.Writer) *MoneyDanceFormatter {
	return &MoneyDanceFormatter{
		CSVFormatter: NewCSVFormatter(w),
	}
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
		t.CreatedAt.Format(moneyDanceTimeFormat),
		t.Reference,
		t.Category,
		t.Amount.String(),
		t.Notes,
	})
}
