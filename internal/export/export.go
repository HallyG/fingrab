package export

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/HallyG/fingrab/internal/domain"
	"github.com/HallyG/fingrab/internal/format"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rs/zerolog"
)

type ExportType string
type ExporterConstructor func(opts Options) (Exporter, error)

type Exporter interface {
	Type() ExportType
	MaxDateRange() time.Duration
	ExportTransactions(ctx context.Context, opts Options) ([]*domain.Transaction, error) // Export(ctx context.Context, opts Options) ([]T, error)
}

type Options struct {
	AccountID string
	EndDate   time.Time
	StartDate time.Time
	AuthToken string
	Timeout   time.Duration
	Format    format.FormatType
}

func (o Options) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &o,
		validation.Field(&o.StartDate, validation.Required.Error("start time is required")),
		validation.Field(&o.EndDate, validation.Required.Error("end time is required")),
		validation.Field(&o.AuthToken, validation.Required.Error("auth token is required")),
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
	constructor, exists := registry[exportType]
	if !exists {
		return nil, fmt.Errorf("unsupported export type: %s", exportType)
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

func Transactions(ctx context.Context, exportType ExportType, opts Options, formatter format.Formatter) error {
	if err := opts.Validate(ctx); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	exporter, err := NewExporter(exportType, opts)
	if err != nil {
		return fmt.Errorf("failed to create %s exporter: %w", exportType, err)
	}

	maxDateRange := exporter.MaxDateRange()
	if maxDateRange > 0 && opts.EndDate.Sub(opts.StartDate) > maxDateRange {
		return fmt.Errorf("date range is too long, max is %d days", int(maxDateRange/time.Hour*24))
	}

	ctx = zerolog.Ctx(ctx).With().
		Str("exporter.type", string(exportType)).
		Logger().
		WithContext(ctx)

	transactions, err := exporter.ExportTransactions(ctx, opts)
	if err != nil {
		return err
	}

	if err := formatter.WriteHeader(); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, t := range transactions {
		if err := formatter.WriteTransaction(t); err != nil {
			return fmt.Errorf("failed to write transaction: %w", err)
		}
	}

	if err := formatter.Flush(); err != nil {
		return fmt.Errorf("failed to flush formatter: %w", err)
	}

	return nil
}
