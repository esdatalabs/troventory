package items

import (
	"errors"
	"sync"

	"github.com/esdatalabs/troventory/internal/items/services/catalog"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
)

// Dispatcher is the items feature's single inbound entry point. It routes
// commands to the correct service and owns the shutdown cascade.
type Dispatcher interface {
	// ManageItem submits cmd to the catalog service. It only returns an
	// error if the Dispatcher has been closed; the command's own outcome is
	// reported asynchronously via entities.AuditGateway.
	ManageItem(cmd catalog.Command) error

	// EnrichItem submits cmd to the enrich service. It only returns an
	// error if the Dispatcher has been closed; the command's own outcome is
	// reported asynchronously via entities.AuditGateway.
	EnrichItem(cmd enrich.Command) error

	// Close refuses further sends and drains every service before
	// returning. It is idempotent.
	Close()
}

// ErrDispatcherClosed is returned by ManageItem once the Dispatcher has been
// closed.
var ErrDispatcherClosed = errors.New("items: dispatcher is closed")

type dispatcher struct {
	catalog *catalog.Service
	enrich  *enrich.Service

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// NewDispatcher constructs the items feature's Dispatcher over the given
// catalog.Service and enrich.Service.
func NewDispatcher(catalog *catalog.Service, enrich *enrich.Service) Dispatcher {
	return &dispatcher{catalog: catalog, enrich: enrich}
}

func (d *dispatcher) ManageItem(cmd catalog.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}
	d.catalog.Send(cmd)
	return nil
}

func (d *dispatcher) EnrichItem(cmd enrich.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}
	d.enrich.Send(cmd)
	return nil
}

func (d *dispatcher) Close() {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.mu.Unlock()

		d.catalog.Close()
		d.enrich.Close()
	})
}
