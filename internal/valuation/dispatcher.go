package valuation

import (
	"errors"
	"sync"

	"github.com/esdatalabs/troventory/internal/valuation/services/assess"
)

// Dispatcher is the valuation feature's single inbound entry point. It
// routes commands to the correct service and owns the shutdown cascade.
type Dispatcher interface {
	// AssessValue submits cmd to the assess service. It only returns an
	// error if the Dispatcher has been closed; the command's own outcome
	// is reported asynchronously via entities.AuditGateway.
	AssessValue(cmd assess.Command) error

	// Close refuses further sends and drains every service before
	// returning. It is idempotent.
	Close()
}

// ErrDispatcherClosed is returned by AssessValue once the Dispatcher has
// been closed.
var ErrDispatcherClosed = errors.New("valuation: dispatcher is closed")

type dispatcher struct {
	assess *assess.Service

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// NewDispatcher constructs the valuation feature's Dispatcher over the given
// assess.Service.
func NewDispatcher(assess *assess.Service) Dispatcher {
	return &dispatcher{assess: assess}
}

func (d *dispatcher) AssessValue(cmd assess.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}
	d.assess.Send(cmd)
	return nil
}

func (d *dispatcher) Close() {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.mu.Unlock()

		d.assess.Close()
	})
}
