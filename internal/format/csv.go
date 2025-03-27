package format

import (
	"encoding/csv"
	"io"
)

type CSVFormatter struct {
	writer *csv.Writer
}

func NewCSVFormatter(w io.Writer) *CSVFormatter {
	return &CSVFormatter{
		writer: csv.NewWriter(w),
	}
}

func (f *CSVFormatter) Flush() error {
	f.writer.Flush()

	return f.writer.Error()
}
