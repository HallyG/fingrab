// Package format provides transaction formatting capabilities for different output formats.
// It supports pluggable formatters through a registry pattern, allowing transactions
// to be exported to various formats like CSV, YNAB, and MoneyDance.
package format

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
)

type (
	FormatType           string
	FormatterConstructor func(io.Writer, *time.Location) (Formatter, error)
	Formatter            interface {
		// WriteHeader writes any format-specific header.
		WriteHeader() error
		WriteTransaction(t *domain.Transaction) error
		// Flush finalizes the output and ensures all buffered data is written.
		Flush() error
	}
)

var (
	registry     = make(map[FormatType]FormatterConstructor)
	registryLock = sync.RWMutex{}
)

// Register adds a new formatter constructor to the registry for the given format type.
// It is thread-safe and overwrites any existing constructor for the same FormatType.
func register(formatType FormatType, constructor FormatterConstructor) {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry[formatType] = constructor
}

// NewFormatter creates a new formatter for the specified format type.
// Returns an error if the format type is not supported or if formatter creation fails.
func NewFormatter(formatType FormatType, w io.Writer) (Formatter, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	constructor, exists := registry[formatType]
	if !exists {
		return nil, fmt.Errorf("unsupported type: %s", formatType)
	}

	location := time.UTC

	formatter, err := constructor(w, location)
	if err != nil {
		return nil, fmt.Errorf("constructor: %w", err)
	}

	return formatter, nil
}

// All returns a sorted slice (by name) of all registered format types.
func All() []FormatType {
	formats := make([]FormatType, 0, len(registry))
	for format := range registry {
		formats = append(formats, format)
	}

	slices.Sort(formats)

	return formats
}

// WriteCollection writes a slice of transactions using the specified formatter.
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
