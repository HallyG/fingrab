package format

import (
	"io"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
)

const (
	moneyDanceTimeFormat            = "2006-01-02"
	FormatTypeMoneyDance FormatType = "moneydance"
)

func init() {
	register(FormatTypeMoneyDance, func(w io.Writer, location *time.Location) Formatter {
		return newMoneyDanceFormatter(w, location)
	})
}

type MoneyDanceFormatter struct {
	*CSVFormatter
	location *time.Location
}

func newMoneyDanceFormatter(w io.Writer, location *time.Location) *MoneyDanceFormatter {
	return &MoneyDanceFormatter{
		CSVFormatter: NewCSVFormatter(w),
		location:     location,
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
		t.CreatedAt.In(m.location).Format(moneyDanceTimeFormat),
		t.Reference,
		t.Category,
		t.Amount.String(),
		t.Notes,
	})
}
