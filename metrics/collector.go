package metrics

import (
	"context"
	"errors"
)

// Collector receives metric events and routes them to a backend
type Collector interface {
	// Collect receives an event for processing
	Collect(ctx context.Context, event Event) error

	// Flush ensures all buffered events are sent
	Flush(ctx context.Context) error

	// Close shuts down the collector
	Close() error
}

// MultiCollector sends events to multiple collectors
type MultiCollector struct {
	collectors []Collector
}

// NewMultiCollector creates a collector that fans out to multiple collectors
func NewMultiCollector(collectors ...Collector) *MultiCollector {
	return &MultiCollector{collectors: collectors}
}

// Collect sends an event to all collectors
func (m *MultiCollector) Collect(ctx context.Context, event Event) error {
	var errs []error
	for _, c := range m.collectors {
		if err := c.Collect(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return combineErrors(errs)
}

// Flush flushes all collectors
func (m *MultiCollector) Flush(ctx context.Context) error {
	var errs []error
	for _, c := range m.collectors {
		if err := c.Flush(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return combineErrors(errs)
}

// Close closes all collectors
func (m *MultiCollector) Close() error {
	var errs []error
	for _, c := range m.collectors {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return combineErrors(errs)
}

func combineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return errors.Join(errs...)
}
