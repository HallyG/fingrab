package export

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type (
	ExportType          string
	ExporterConstructor func(opts Options) (Exporter, error)
	Exporter            interface {
		Type() ExportType
		// MaxDateRange returns the maximum allowed date range for fetching transactions.
		// A zero duration indicates no limit.
		MaxDateRange() time.Duration
		ExportTransactions(ctx context.Context, opts TransactionOptions) ([]*domain.Transaction, error)
		ExportAccounts(ctx context.Context) ([]*domain.Account, error)
	}
)

type Options struct {
	AuthToken string
	Timeout   time.Duration
}

func (o Options) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &o,
		validation.Field(&o.AuthToken, validation.Required.Error("is required")),
	)
}

// BearerAuthToken formats the AuthToken as a Bearer token by adding the "Bearer " prefix if not already present.
// It trims whitespace from the token for consistency.
// Example:
//
//	opts := Options{AuthToken: "abc123"}
//	token := opts.BearerAuthToken() // Returns "Bearer abc123"
func (o Options) BearerAuthToken() string {
	token := strings.TrimSpace(o.AuthToken)
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}

	return token
}

var (
	registry     = make(map[ExportType]ExporterConstructor)
	registryLock = sync.RWMutex{}
)

// Register adds a new exporter constructor to the registry for the given export type.
// It is thread-safe and overwrites any existing constructor for the same ExportType.
func Register(exportType ExportType, constructor ExporterConstructor) {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry[exportType] = constructor
}

func NewExporter(exportType ExportType, opts Options) (Exporter, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	constructor, exists := registry[exportType]
	if !exists {
		return nil, fmt.Errorf("unsupported type: %s", exportType)
	}

	exporter, err := constructor(opts)
	if err != nil {
		return nil, fmt.Errorf("constructor: %w", err)
	}

	return exporter, nil
}

// All returns a sorted slice (by name) of all registered export types.
func All() []ExportType {
	registryLock.RLock()
	defer registryLock.RUnlock()

	exportTypes := make([]ExportType, 0, len(registry))
	for exportType := range registry {
		exportTypes = append(exportTypes, exportType)
	}

	slices.Sort(exportTypes)

	return exportTypes
}
