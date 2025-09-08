package format

import (
	"io"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
)

const (
	ynabTimeFormat            = "01/02/2006" //date format expected by YNAB (MM/DD/YYYY).
	FormatTypeYNAB FormatType = "ynab"
)

func init() {
	register(FormatTypeYNAB, func(w io.Writer, location *time.Location) (Formatter, error) {
		return &YNABFormatter{
			CSVFormatter: NewCSVFormatter(w),
			location:     location,
		}, nil
	})
}

// YNABFormatter formats transactions for import into You Need A Budget (YNAB).
// It outputs CSV with columns: Date, Payee, Memo, Amount.
type YNABFormatter struct {
	*CSVFormatter
	location *time.Location
}

func (y *YNABFormatter) WriteHeader() error {
	return y.writer.Write([]string{"Date", "Payee", "Memo", "Amount"})
}

func (y *YNABFormatter) WriteTransaction(t *domain.Transaction) error {
	return y.writer.Write([]string{
		t.CreatedAt.In(y.location).Format(ynabTimeFormat),
		t.Reference,
		t.Notes,
		t.Amount.String(),
	})
}
