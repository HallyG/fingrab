package format

import (
	"encoding/csv"
	"io"
)

// CSVFormatter provides CSV output functionality for transaction formatters.
// It wraps the standard csv.Writer.
type CSVFormatter struct {
	writer *csv.Writer
}

// NewCSVFormatter creates a new CSV formatter that writes to the provided io.Writer.
func NewCSVFormatter(w io.Writer) *CSVFormatter {
	return &CSVFormatter{
		writer: csv.NewWriter(w),
	}
}

func (f *CSVFormatter) Flush() error {
	f.writer.Flush()

	return f.writer.Error()
}
