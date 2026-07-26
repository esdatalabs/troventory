package enrich

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/items/entities"
)

// Service orchestrates enriching a draft or existing item's catalog data
// from a barcode lookup. A Command is the only entry point — Send has no
// return value, so every outcome (success or failure) is reported through
// entities.AuditGateway (ARCHITECTURE.md §4).
type Service struct {
	items  ItemGateway
	lookup ProductLookupGateway
	audit  entities.AuditGateway
	clock  entities.Clock
	log    *slog.Logger

	commands  chan Command
	timeout   time.Duration
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// New constructs a Service, creates its command channel, and starts its
// consumer loop.
func New(
	items ItemGateway,
	lookup ProductLookupGateway,
	audit entities.AuditGateway,
	clock entities.Clock,
	log *slog.Logger,
	buffer int,
	timeout time.Duration,
) *Service {
	s := &Service{
		items:    items,
		lookup:   lookup,
		audit:    audit,
		clock:    clock,
		log:      log,
		commands: make(chan Command, buffer),
		timeout:  timeout,
	}

	s.wg.Add(1)
	go s.loop()

	return s
}

// Send is the only way to submit a Command to this Service. It returns
// nothing; the caller learns the outcome via entities.AuditGateway.
func (s *Service) Send(cmd Command) {
	s.commands <- cmd
}

// Close stops accepting further work by closing the command channel and
// blocks until every in-flight command has finished processing. It is
// idempotent and must not be called concurrently with Send.
func (s *Service) Close() {
	s.closeOnce.Do(func() {
		close(s.commands)
	})
	s.wg.Wait()
}

func (s *Service) loop() {
	defer s.wg.Done()

	for cmd := range s.commands {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		s.handle(ctx, cmd)
		cancel()
	}
}

// handle processes a single Command and reports its outcome via the
// AuditGateway — the only place any error surfaces, since Send returns
// nothing.
func (s *Service) handle(ctx context.Context, cmd Command) {
	s.log.Debug("processing enrich command",
		"correlation_id", cmd.CorrelationID,
		"at", s.clock.Now(),
	)

	err := s.process(ctx, cmd)

	result := entities.Result{
		CorrelationID: cmd.CorrelationID,
		Reference:     cmd.Reference,
		Err:           err,
	}

	if auditErr := s.audit.RecordResult(ctx, result); auditErr != nil {
		s.log.Error("record enrich result",
			"correlation_id", cmd.CorrelationID,
			"error", auditErr,
		)
	}
}

// process runs the enrich pipeline for cmd, each step short-circuiting the
// rest on error:
//  1. validate the barcode's format (pure, no I/O) — before any Gateway call
//  2. locate the target item
//  3. reject an already-archived target
//  4. look up product details for the barcode
//  5. fill gaps only, never overwriting an existing field
//  6. persist the (possibly unchanged) item
func (s *Service) process(ctx context.Context, cmd Command) error {
	if err := validateBarcode(cmd.Barcode); err != nil {
		return err
	}

	item, err := s.locateItem(ctx, cmd)
	if err != nil {
		return err
	}
	if item.Archived {
		return fmt.Errorf("enrich item %q: %w", item.Description, entities.ErrItemArchived)
	}

	details, err := s.lookup.Lookup(ctx, cmd.Barcode)
	if err != nil {
		return fmt.Errorf("look up barcode %q: %w", cmd.Barcode, err)
	}

	item = applyProductDetails(item, details)

	if err := s.items.Save(ctx, item); err != nil {
		return fmt.Errorf("save item: %w", err)
	}
	return nil
}

// locateItem finds the target item: by TargetDescription if it's non-empty,
// else by Barcode.
func (s *Service) locateItem(ctx context.Context, cmd Command) (entities.Item, error) {
	if cmd.TargetDescription != "" {
		item, err := s.items.FindByDescription(ctx, cmd.TargetDescription)
		if err != nil {
			return entities.Item{}, fmt.Errorf("find item %q: %w", cmd.TargetDescription, err)
		}
		return item, nil
	}

	item, err := s.items.FindByBarcode(ctx, cmd.Barcode)
	if err != nil {
		return entities.Item{}, fmt.Errorf("find item by barcode %q: %w", cmd.Barcode, err)
	}
	return item, nil
}
