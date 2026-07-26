package locations

import (
	"errors"
	"sync"

	"github.com/esdatalabs/troventory/internal/locations/services/manage"
)

// Dispatcher is the locations feature's single inbound entry point. It
// routes commands to the correct service and owns the shutdown cascade.
type Dispatcher interface {
	// ManageLocation submits cmd to the manage service. It only returns an
	// error if the Dispatcher has been closed; the command's own outcome is
	// reported asynchronously via entities.AuditGateway.
	ManageLocation(cmd manage.Command) error

	// Close refuses further sends and drains every service before
	// returning. It is idempotent.
	Close()
}

// ErrDispatcherClosed is returned by ManageLocation once the Dispatcher has
// been closed.
var ErrDispatcherClosed = errors.New("locations: dispatcher is closed")

type dispatcher struct {
	manage *manage.Service

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// NewDispatcher constructs the locations feature's Dispatcher over the given
// manage.Service.
func NewDispatcher(manage *manage.Service) Dispatcher {
	return &dispatcher{manage: manage}
}

func (d *dispatcher) ManageLocation(cmd manage.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}
	d.manage.Send(cmd)
	return nil
}

func (d *dispatcher) Close() {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.mu.Unlock()

		d.manage.Close()
	})
}
