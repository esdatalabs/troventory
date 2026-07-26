package search

import (
	"errors"
	"sync"

	"github.com/esdatalabs/troventory/internal/search/services/query"
)

// Dispatcher is the search feature's single inbound entry point. It routes
// commands to the correct service and owns the shutdown cascade.
type Dispatcher interface {
	// Search submits cmd to the query service. It only returns an error
	// if the Dispatcher has been closed; the command's own outcome is
	// reported asynchronously via entities.AuditGateway.
	Search(cmd query.Command) error

	// Close refuses further sends and drains every service before
	// returning. It is idempotent.
	Close()
}

// ErrDispatcherClosed is returned by Search once the Dispatcher has been
// closed.
var ErrDispatcherClosed = errors.New("search: dispatcher is closed")

type dispatcher struct {
	query *query.Service

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// NewDispatcher constructs the search feature's Dispatcher over the given
// query.Service.
func NewDispatcher(query *query.Service) Dispatcher {
	return &dispatcher{query: query}
}

func (d *dispatcher) Search(cmd query.Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}
	d.query.Send(cmd)
	return nil
}

func (d *dispatcher) Close() {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.mu.Unlock()

		d.query.Close()
	})
}
