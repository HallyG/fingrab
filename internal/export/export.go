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

type ExportType string
type ExporterConstructor func(opts Options) (Exporter, error)

type Exporter interface {
	Type() ExportType
	// MaxDateRange returns the maximum allowed date range for fetching transactions.
	// A zero duration indicates no limit.
	MaxDateRange() time.Duration
	ExportTransactions(ctx context.Context, opts Options) ([]*domain.Transaction, error)
}

type Options struct {
	AccountID string
	EndDate   time.Time
	StartDate time.Time
	AuthToken string
	Timeout   time.Duration
}

func (o Options) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &o,
		validation.Field(&o.StartDate, validation.Required.Error("is required")),
		validation.Field(&o.EndDate, validation.Required.Error("is required")),
		validation.Field(&o.AuthToken, validation.Required.Error("is required")),
	)
}

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

	return constructor(opts)
}

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

func Transactions(ctx context.Context, exportType ExportType, opts Options) ([]*domain.Transaction, error) {
	if err := opts.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	exporter, err := NewExporter(exportType, opts)
	if err != nil {
		return nil, fmt.Errorf("create exporter: %w", err)
	}

	maxDateRange := exporter.MaxDateRange()
	days := (opts.EndDate.Sub(opts.StartDate).Hours()) / 24
	if maxDateRange > 0 && opts.EndDate.Sub(opts.StartDate) > maxDateRange {
		hours := maxDateRange.Hours()
		maxDays := hours / 24
		return nil, fmt.Errorf("date range %d days is too long, max is %d days", int(days), int(maxDays))
	}

	transactions, err := exporter.ExportTransactions(ctx, opts)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}
