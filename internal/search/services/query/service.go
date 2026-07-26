package query

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/search/entities"
)

// Service orchestrates searching and filtering items across the catalog,
// location hierarchy, and current-value data. A Command is the only entry
// point — Send has no return value, so every outcome (success or failure)
// is reported through entities.AuditGateway (ARCHITECTURE.md §4).
type Service struct {
	items     ItemGateway
	locations LocationGateway
	values    ValueGateway
	audit     entities.AuditGateway
	clock     entities.Clock
	log       *slog.Logger

	commands  chan Command
	timeout   time.Duration
	wg        sync.WaitGroup
	closeOnce sync.Once

	// resultsByRef memoizes the first Result produced for a given
	// non-empty Command.Reference, so a repeat submission replays the
	// identical Result without calling ItemGateway.FindAll again. It is
	// confined to this Service's own single consumer goroutine (loop),
	// per ARCHITECTURE.md §5 (Concurrency), so it needs no mutex.
	resultsByRef map[string]entities.Result
}

// New constructs a Service, creates its command channel, and starts its
// consumer loop.
func New(
	items ItemGateway,
	locations LocationGateway,
	values ValueGateway,
	audit entities.AuditGateway,
	clock entities.Clock,
	log *slog.Logger,
	buffer int,
	timeout time.Duration,
) *Service {
	s := &Service{
		items:        items,
		locations:    locations,
		values:       values,
		audit:        audit,
		clock:        clock,
		log:          log,
		commands:     make(chan Command, buffer),
		timeout:      timeout,
		resultsByRef: make(map[string]entities.Result),
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
// AuditGateway — the only place any error or match list surfaces, since
// Send returns nothing. A repeat submission under a non-empty Reference
// replays the memoized Result without carrying out the search again.
func (s *Service) handle(ctx context.Context, cmd Command) {
	if cmd.Reference != "" {
		if result, ok := s.resultsByRef[cmd.Reference]; ok {
			s.record(ctx, result)
			return
		}
	}

	s.log.Debug("processing search query command",
		"correlation_id", cmd.CorrelationID,
		"reference", cmd.Reference,
		"at", s.clock.Now(),
	)

	matches, err := s.search(ctx, cmd)

	result := entities.Result{
		CorrelationID: cmd.CorrelationID,
		Reference:     cmd.Reference,
		Err:           err,
		Matches:       matches,
	}

	if cmd.Reference != "" {
		s.resultsByRef[cmd.Reference] = result
	}

	s.record(ctx, result)
}

// record delivers result via the AuditGateway.
func (s *Service) record(ctx context.Context, result entities.Result) {
	if err := s.audit.RecordResult(ctx, result); err != nil {
		s.log.Error("record search query result",
			"correlation_id", result.CorrelationID,
			"error", err,
		)
	}
}
