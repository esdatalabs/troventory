package manage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/locations/entities"
)

// Service orchestrates creating, renaming, moving, and archiving locations.
// A Command is the only entry point — Send has no return value, so every
// outcome (success or failure) is reported through entities.AuditGateway
// (ARCHITECTURE.md §4).
type Service struct {
	storage StorageGateway
	audit   entities.AuditGateway
	clock   entities.Clock
	log     *slog.Logger

	commands  chan Command
	timeout   time.Duration
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// New constructs a Service, creates its command channel, and starts its
// consumer loop.
func New(
	storage StorageGateway,
	audit entities.AuditGateway,
	clock entities.Clock,
	log *slog.Logger,
	buffer int,
	timeout time.Duration,
) *Service {
	s := &Service{
		storage:  storage,
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
	s.log.Debug("processing manage location command",
		"correlation_id", cmd.CorrelationID,
		"action", cmd.Action,
		"at", s.clock.Now(),
	)

	err := s.process(ctx, cmd)

	result := entities.Result{
		CorrelationID: cmd.CorrelationID,
		Reference:     cmd.Reference,
		Err:           err,
	}

	if auditErr := s.audit.RecordResult(ctx, result); auditErr != nil {
		s.log.Error("record manage location result",
			"correlation_id", cmd.CorrelationID,
			"error", auditErr,
		)
	}
}

// process dispatches cmd to the handler for its Action.
func (s *Service) process(ctx context.Context, cmd Command) error {
	switch cmd.Action {
	case ActionCreate:
		return s.create(ctx, cmd)
	case ActionRename:
		return s.rename(ctx, cmd)
	case ActionMove:
		return s.move(ctx, cmd)
	case ActionArchive:
		return s.archive(ctx, cmd)
	default:
		return fmt.Errorf("manage: unknown action %d", cmd.Action)
	}
}
