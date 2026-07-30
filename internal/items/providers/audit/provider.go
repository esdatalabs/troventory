// Package audit implements the items feature's entities.AuditGateway by
// handing each Result to whichever CLI goroutine is awaiting its
// correlation ID — the driving-side bridge from this feature's
// fire-and-forget Service.Send back to a synchronous CLI prompt.
package audit

import (
	"context"

	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/platform/resultwait"
)

// Provider backs entities.AuditGateway.
type Provider struct {
	waiter *resultwait.Waiter[entities.Result]
}

// New returns an empty Provider.
func New() *Provider {
	return &Provider{waiter: resultwait.New[entities.Result]()}
}

// RecordResult delivers result to whatever call is awaiting its
// CorrelationID.
func (p *Provider) RecordResult(_ context.Context, result entities.Result) error {
	p.waiter.Deliver(result.CorrelationID, result)
	return nil
}

// Await blocks until a Result is recorded for correlationID, or ctx is
// done.
func (p *Provider) Await(ctx context.Context, correlationID string) (entities.Result, error) {
	return p.waiter.Await(ctx, correlationID)
}
