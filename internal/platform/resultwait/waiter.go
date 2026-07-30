// Package resultwait bridges this project's fire-and-forget Service.Send
// design (ARCHITECTURE.md §4 — a Service's only feedback is an
// asynchronous AuditGateway.RecordResult call) back to a synchronous
// caller. It is driving-side infrastructure for the interactive CLI, not
// business logic: the CLI submits a Command and wants to print its
// outcome before prompting for the next one.
package resultwait

import (
	"context"
	"fmt"
	"sync"
)

// Waiter delivers a result of type R to whichever goroutine is waiting on
// the correlation ID it was produced for. Each feature's AuditGateway
// Provider wraps one Waiter and calls Deliver from RecordResult; the CLI
// calls Await right after Send with the same correlation ID.
type Waiter[R any] struct {
	mu      sync.Mutex
	pending map[string]chan R
}

// New returns an empty Waiter.
func New[R any]() *Waiter[R] {
	return &Waiter[R]{pending: make(map[string]chan R)}
}

// Await blocks until a result is delivered for correlationID, or ctx is
// done, whichever comes first.
func (w *Waiter[R]) Await(ctx context.Context, correlationID string) (R, error) {
	ch := w.register(correlationID)

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		w.abandon(correlationID)
		var zero R
		return zero, fmt.Errorf("await result for %q: %w", correlationID, ctx.Err())
	}
}

// Deliver hands result to whatever goroutine is awaiting correlationID. If
// nothing is currently awaiting it, the result is dropped — every
// correlation ID this project generates is awaited exactly once, by the
// same command loop that generated it.
func (w *Waiter[R]) Deliver(correlationID string, result R) {
	w.mu.Lock()
	ch, ok := w.pending[correlationID]
	if ok {
		delete(w.pending, correlationID)
	}
	w.mu.Unlock()

	if ok {
		ch <- result
	}
}

func (w *Waiter[R]) register(correlationID string) chan R {
	ch := make(chan R, 1)

	w.mu.Lock()
	w.pending[correlationID] = ch
	w.mu.Unlock()

	return ch
}

func (w *Waiter[R]) abandon(correlationID string) {
	w.mu.Lock()
	delete(w.pending, correlationID)
	w.mu.Unlock()
}
