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
	WriteHeader() error
	WriteTransaction(t *domain.Transaction) error
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
		return nil, fmt.Errorf("unsupported format type: %s", format)
	}

	location := time.Local
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
