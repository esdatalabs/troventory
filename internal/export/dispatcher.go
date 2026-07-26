package export

import (
	"errors"
	"sync"

	report "github.com/esdatalabs/troventory/internal/export/services/insurance-report"
)

// Dispatcher is the export feature's single inbound entry point. It routes
// commands to the correct service and owns the shutdown cascade.
type Dispatcher interface {
	// ExportInsuranceReport submits cmd to the insurance-report service.
	// It only returns an error if the Dispatcher has been closed; the
	// command's own outcome is reported asynchronously via
	// entities.AuditGateway.
	ExportInsuranceReport(cmd report.Command) error

	// Close refuses further sends and drains every service before
	// returning. It is idempotent.
	Close()
}

// ErrDispatcherClosed is returned by ExportInsuranceReport once the
// Dispatcher has been closed.
var ErrDispatcherClosed = errors.New("export: dispatcher is closed")

type dispatcher struct {
	report *report.Service

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// NewDispatcher constructs the export feature's Dispatcher over the given
// report.Service.
func NewDispatcher(report *report.Service) Dispatcher {
	return &dispatcher{report: report}
}

func (d *dispatcher) ExportInsuranceReport(cmd report.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}
	d.report.Send(cmd)
	return nil
}

func (d *dispatcher) Close() {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.mu.Unlock()

		d.report.Close()
	})
}
