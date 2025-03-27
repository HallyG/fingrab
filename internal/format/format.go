package format

import (
	"fmt"
	"io"
	"slices"

	"github.com/HallyG/fingrab/internal/domain"
)

type FormatType string

type Formatter interface {
	WriteHeader() error
	WriteTransaction(t *domain.Transaction) error
	Flush() error
}

var registry = make(map[FormatType]func(io.Writer) Formatter)

func register(format FormatType, constructor func(io.Writer) Formatter) {
	registry[format] = constructor
}

func NewFormatter(format FormatType, w io.Writer) (Formatter, error) {
	constructor, exists := registry[format]
	if !exists {
		return nil, fmt.Errorf("unsupported format type: %s", format)
	}

	return constructor(w), nil
}

func All() []FormatType {
	formats := make([]FormatType, 0, len(registry))
	for format := range registry {
		formats = append(formats, format)
	}

	slices.Sort(formats)

	return formats
}
