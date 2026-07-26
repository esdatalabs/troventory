package report

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/export/entities"
)

// Service orchestrates generating insurance reports from the catalog's
// active items and their recorded valuations. A Command is the only entry
// point — Send has no return value, so every outcome (success or failure)
// is reported through entities.AuditGateway (ARCHITECTURE.md §4).
type Service struct {
	items     ItemGateway
	valuation ValuationGateway
	storage   StorageGateway
	audit     entities.AuditGateway
	clock     entities.Clock
	log       *slog.Logger

	commands  chan Command
	timeout   time.Duration
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// New constructs a Service, creates its command channel, and starts its
// consumer loop.
func New(
	items ItemGateway,
	valuation ValuationGateway,
	storage StorageGateway,
	audit entities.AuditGateway,
	clock entities.Clock,
	log *slog.Logger,
	buffer int,
	timeout time.Duration,
) *Service {
	s := &Service{
		items:     items,
		valuation: valuation,
		storage:   storage,
		audit:     audit,
		clock:     clock,
		log:       log,
		commands:  make(chan Command, buffer),
		timeout:   timeout,
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
// AuditGateway — the only place any error or generated Report surfaces,
// since Send returns nothing.
func (s *Service) handle(ctx context.Context, cmd Command) {
	s.log.Debug("processing export insurance report command",
		"correlation_id", cmd.CorrelationID,
		"reference", cmd.Reference,
		"format", cmd.Format,
		"at", s.clock.Now(),
	)

	rpt, err := s.generateReport(ctx, cmd)

	result := entities.Result{
		CorrelationID: cmd.CorrelationID,
		Reference:     cmd.Reference,
		Err:           err,
		Report:        rpt,
	}

	if auditErr := s.audit.RecordResult(ctx, result); auditErr != nil {
		s.log.Error("record export insurance report result",
			"correlation_id", cmd.CorrelationID,
			"error", auditErr,
		)
	}
}
