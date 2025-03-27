package format

import (
	"io"

	"github.com/HallyG/fingrab/internal/domain"
)

const (
	ynabTimeFormat            = "01/02/2006" // MM/DD/YYYY
	FormatTypeYNAB FormatType = "ynab"
)

func init() {
	register(FormatTypeYNAB, func(w io.Writer) Formatter {
		return newYNABFormatter(w)
	})
}

type YNABFormatter struct {
	*CSVFormatter
}

func newYNABFormatter(w io.Writer) *YNABFormatter {
	return &YNABFormatter{
		CSVFormatter: NewCSVFormatter(w),
	}
}

func (y *YNABFormatter) WriteHeader() error {
	return y.writer.Write([]string{"Date", "Payee", "Memo", "Amount"})
}

func (y *YNABFormatter) WriteTransaction(t *domain.Transaction) error {
	return y.writer.Write([]string{
		t.CreatedAt.Format(ynabTimeFormat),
		t.Reference,
		t.Notes,
		t.Amount.String(),
	})
}
