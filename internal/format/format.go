package format

import (
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
)

type FormatType string

type Formatter interface {
	// WriteHeader writes any format-specific header information.
	// This is called once before any transactions are written.
	WriteHeader() error
	WriteTransaction(t *domain.Transaction) error
	// Flush finalizes the output and ensures all buffered data is written.
	// This is called once after all transactions have been written.
	Flush() error
}

type constructor func(io.Writer, *time.Location) Formatter

var registry = make(map[FormatType]constructor)

func register(format FormatType, constructor constructor) {
	registry[format] = constructor
}

func NewFormatter(format FormatType, w io.Writer) (Formatter, error) {
	constructor, exists := registry[format]
	if !exists {
		return nil, fmt.Errorf("unsupported type: %s", format)
	}

	location := time.UTC
	return constructor(w, location), nil
}

func All() []FormatType {
	formats := make([]FormatType, 0, len(registry))
	for format := range registry {
		formats = append(formats, format)
	}

	slices.Sort(formats)

	return formats
}

// WriteCollection writes a complete collection of transactions using the specified formatter.
// This is a convenience function that handles the full three-phase writing process:
// writing the header, writing all transactions, and flushing the output.
func WriteCollection(formatter Formatter, transactions []*domain.Transaction) error {
	if err := formatter.WriteHeader(); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, t := range transactions {
		if err := formatter.WriteTransaction(t); err != nil {
			return fmt.Errorf("write transaction: %w", err)
		}
	}

	if err := formatter.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	return nil
}
