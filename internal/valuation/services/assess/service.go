package assess

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/valuation/entities"
)

// Service orchestrates recording purchase prices and appraisals,
// configuring depreciation, and computing an item's current value. A
// Command is the only entry point — Send has no return value, so every
// outcome (success or failure) is reported through entities.AuditGateway
// (ARCHITECTURE.md §4).
type Service struct {
	items   ItemGateway
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
	items ItemGateway,
	storage StorageGateway,
	audit entities.AuditGateway,
	clock entities.Clock,
	log *slog.Logger,
	buffer int,
	timeout time.Duration,
) *Service {
	s := &Service{
		items:    items,
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
// AuditGateway — the only place any error or computed value surfaces,
// since Send returns nothing.
func (s *Service) handle(ctx context.Context, cmd Command) {
	s.log.Debug("processing assess value command",
		"correlation_id", cmd.CorrelationID,
		"action", cmd.Action,
		"at", s.clock.Now(),
	)

	value, err := s.process(ctx, cmd)

	result := entities.Result{
		CorrelationID: cmd.CorrelationID,
		Reference:     cmd.Reference,
		Err:           err,
		Value:         value,
	}

	if auditErr := s.audit.RecordResult(ctx, result); auditErr != nil {
		s.log.Error("record assess value result",
			"correlation_id", cmd.CorrelationID,
			"error", auditErr,
		)
	}
}

// process dispatches cmd to the handler for its Action. Only
// ActionComputeCurrentValue returns a non-nil *entities.Money.
func (s *Service) process(ctx context.Context, cmd Command) (*entities.Money, error) {
	switch cmd.Action {
	case ActionRecordPurchasePrice:
		return nil, s.recordPurchasePrice(ctx, cmd)
	case ActionRecordAppraisal:
		return nil, s.recordAppraisal(ctx, cmd)
	case ActionConfigureDepreciation:
		return nil, s.configureDepreciation(ctx, cmd)
	case ActionComputeCurrentValue:
		return s.computeCurrentValue(ctx, cmd)
	default:
		return nil, fmt.Errorf("assess: unknown action %d", cmd.Action)
	}
}
