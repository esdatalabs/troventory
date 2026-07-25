# Architecture Guide

This document defines the architecture, naming conventions, and folder structure for this
project. It is written for an agent (human or AI) starting a **greenfield** codebase. Follow
it exactly unless a rule explicitly says otherwise — consistency here matters more than any
individual preference.

The project follows the principles of **Hexagonal Architecture**: dependencies always point
**inward**, toward business logic, never outward toward frameworks or infrastructure. It does
**not** use the standard hexagonal vocabulary ("ports", "adapters") — see the Vocabulary
section for the terms this project uses instead.

Examples throughout use a **payments** feature (funds transfers and account limit changes) as
the reference domain.

---

## 1. The Hierarchy

```
App
 └── Feature          (e.g. payments, onboarding, statements)
      ├── Dispatcher   (ONE per feature — routes inbound messages to its services)
      └── Service      (e.g. transfer, limitchange — one per business "verb")
           └── uses Entities + Gateways (its own outbound interfaces)
```

- **App** — the whole binary/deployable. Composed of independent features under `/internal`.
- **Feature** — a bounded slice of business capability (e.g. `payments`). Owns a `dispatcher`,
  its `entities`, one or more `services`, and the `providers` that back them. Features do
  **not** import each other's `services` or `providers` packages directly.
- **Service** — one unit of orchestration for one specific use case (e.g. `transfer`,
  `limitchange`). Owns its own command, its own gateways where not shared, and its own
  step-by-step orchestration logic. Services within the same feature do **not** call each
  other directly.
- **Entity** — a pure business object or rule. No I/O, no framework imports.
- **Gateway** — an interface describing something the app needs from the outside world.
- **Provider** — the concrete implementation of a Gateway.

---

## 2. Vocabulary (memorize this — it's the whole naming system)

| Term | Direction | Definition | Example |
|---|---|---|---|
| **Entity** | — | Pure business object or rule. No I/O. Lives in `entities`. | `Account`, `Money`, `TransferPlan`, `CalculateFee()` |
| **Gateway** | Outbound (app calls **out**) | Interface for something external the app depends on: ledger DB, core banking system, fraud screening, notifications. Implemented by Providers, faked in tests. | `LedgerGateway`, `FraudGateway`, `AuditGateway` |
| **Provider** | Outbound (implements a Gateway) | The concrete implementation of a Gateway — talks to the real database, core banking host, or vendor API. Lives in `providers/`. | `postgres.Provider`, `corebank.Provider` |
| **Service** | — | Orchestrates entities + gateways to complete one use case. Owns its own command channel and consumer loop — messages in, no return value. No interface unless a second caller genuinely needs one — default to a concrete struct. | `transfer.Service`, `limitchange.Service` |
| **Dispatcher** | Inbound (something calls **in**) | The single entry point per feature. Routes an inbound message to the correct service's channel, and owns the shutdown cascade. | `payments.Dispatcher` |

**Directional pairing to remember:** `Gateway` = we call out (the promise). `Provider` = the
concrete thing fulfilling that promise. `Dispatcher` = something calls in.

**Do not use:**
- "Adapter" — superseded by Provider.
- "Repository" — superseded by Gateway.
- "Port" — superseded by Gateway (outbound) and Dispatcher (inbound).
- "Application" as a folder or package name — a feature contains `services`, not an
  "application layer".
- "Handler" for anything in `services/` — reserve "Handler" for driving-side message handlers
  only (e.g. `kafka.MessageHandler`), which are not Providers since they don't implement a
  Gateway.

---

## 3. Folder Structure

```
/internal/payments
  dispatcher.go              -> Dispatcher interface + concrete impl. The ONLY inbound entry
                                point for this feature; routes to every service below.

  /entities
    account.go               -> Account, AccountTier
    money.go                   -> Money value type
    transfer.go                  -> TransferPlan, FeeBreakdown
    transfer_rules.go              -> CalculateFee, ValidateDailyLimit (pure, no I/O)
    limit_rules.go                   -> ValidateLimitChange (pure, no I/O)
    result.go                          -> Result — shared outcome type for every service
    errors.go                            -> business sentinel errors (ErrInsufficientFunds, ...)
    gateways.go                            -> SHARED gateways: AuditGateway, NotificationGateway

  /services
    /transfer
      command.go             -> Command (input DTO: from, to, amount, reference)
      service.go               -> Service (channel + loop + orchestration)
      gateways.go                -> LedgerGateway, FraudGateway (used ONLY by transfer)

    /limitchange
      command.go             -> Command (account, new daily limit)
      service.go
      gateways.go                -> AccountGateway (used ONLY by limitchange)

  /providers
    /postgres
      ledger_provider.go     -> implements transfer.LedgerGateway
      audit_provider.go        -> implements entities.AuditGateway
      provider_test.go           -> integration tests + compile-time interface assertions

    /corebank                -> the bank's system of record (ISO 8583 / vendor SDK / host API)
      provider.go              -> implements limitchange.AccountGateway

    /fraudnet                -> third-party transaction screening vendor
      provider.go              -> implements transfer.FraudGateway

    /notify                  -> customer comms vendor (SMS/email/push)
      provider.go              -> implements entities.NotificationGateway

    /kafka                   -> the DRIVING side — inbound payment instructions
      message_handler.go       -> unpacks inbound message, calls Dispatcher.
                                  NOT a Provider — it doesn't implement a Gateway, it
                                  consumes the Dispatcher.

/cmd
  /worker
    main.go                  -> wires providers -> services -> dispatcher -> driving side
```

### Rules

1. **Interfaces are defined by the consumer, not the implementer.** A Gateway interface lives
   in the `services/<service>` package that calls it (or `entities`, if shared) — never in
   `providers/`. Providers define zero interfaces; they only define concrete types and
   methods, and satisfy Gateway interfaces structurally.
2. **`entities` imports nothing but the standard library.** No DB drivers, no HTTP, no JSON
   tags, no vendor SDK types. If you're tempted to import anything else into `entities`, the
   logic belongs in `services` or `providers` instead.
3. **`main.go` (or equivalent in `/cmd`) is the only place that imports every provider
   package.** All wiring — constructing providers, injecting them into services, injecting
   services into the dispatcher, injecting the dispatcher into the driving side — happens
   here and only here.
4. **A Gateway interface is sized to exactly what its consumer needs**, not to everything the
   underlying system can do. The core banking host may expose a hundred operations; if
   `limitchange` only sets a daily limit, its `AccountGateway` has exactly one method. Promote
   an interface to `entities/gateways.go` only once 2+ services need the *identical* operation
   for the *same reason* (not merely the same external system).

---

## 4. Service Lifecycle & Messaging

**A message is the only entry point to a Service.** Services do not expose callable
orchestration methods; they expose a channel send.

### Rules

1. **A Service creates its own channel in its constructor** and starts its consumer loop
   there. The channel's lifetime is exactly the Service's lifetime — it is never passed in
   from outside, and never exposed as a raw `chan` for others to close.
2. **`Send(cmd)` is the only way in.** It returns nothing. Because a channel send cannot
   return an error, outcomes — success *and* failure — are reported through the
   `AuditGateway` and `NotificationGateway`, never to the caller.
3. **`Close()` is a separate, explicit method.** It closes the channel, waits for in-flight
   work to drain (`sync.WaitGroup`), and is idempotent (`sync.Once`). It must not be called
   concurrently with `Send`.
4. **The Dispatcher is the sole sender.** This is the safety rule that makes closing possible:
   a send on a closed channel panics, so the Dispatcher sets a `closed` flag first and refuses
   further sends, *then* closes each Service. Nothing else in the codebase may hold a
   reference to a Service and call `Send`.
5. **Shutdown order is fixed:** stop the inbound driving side (consumer/server) →
   `Dispatcher.Close()` (refuse new sends, drain every Service) → close Providers. Closing
   Providers before Services have drained will fail in-flight work — in a payments context
   that means a transfer that has debited but not credited.
6. **Buffer size and worker count are explicit constructor arguments.** Unbuffered gives
   natural backpressure; a buffer absorbs bursts but loses queued work on a hard kill. For
   parallelism, start N goroutines over the same channel in the constructor — the WaitGroup
   drain logic is unchanged. Be deliberate here: for financial instructions, durability of the
   inbound queue matters more than in-process buffering.
7. **Never send the inbound message's `context.Context` across the channel.** It is cancelled
   as soon as the driving-side handler returns, which would kill work that hasn't started yet.
   The Service derives a **fresh** context per command (with its own timeout), carrying
   forward only the correlation ID. See §5.

### Skeleton

```go
type Service struct {
	// ...gateways...
	commands  chan Command
	timeout   time.Duration
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func New(/* gateways */, buffer int, timeout time.Duration) *Service {
	s := &Service{ /* ... */ commands: make(chan Command, buffer), timeout: timeout}
	s.wg.Add(1)
	go s.loop()
	return s
}

func (s *Service) Send(cmd Command) { s.commands <- cmd }

func (s *Service) Close() {
	s.closeOnce.Do(func() { close(s.commands) })
	s.wg.Wait()
}

func (s *Service) loop() {
	defer s.wg.Done()
	for cmd := range s.commands {
		ctx, cancel := context.WithTimeout(
			ContextWithCorrelationID(context.Background(), cmd.CorrelationID),
			s.timeout,
		)
		s.handle(ctx, cmd) // records result via audit + notification gateways
		cancel()
	}
}
```

---

## 5. Go Conventions

Non-negotiable baseline. These are standard-library-era idioms, not preferences.

### Packages & naming

- Package names are **short, lowercase, single-word, no underscores or plurals** where natural:
  `transfer`, `postgres`, `corebank`. The two structural exceptions in this guide are
  `entities`, `services`, and `providers`, which are directories that group packages, not
  packages themselves.
- **Avoid stutter.** `transfer.Service`, not `transfer.TransferService`. `postgres.Provider`,
  not `postgres.PostgresProvider`.
- Exported identifiers get doc comments starting with the identifier name.
- File names are `snake_case.go` and describe contents (`ledger_provider.go`,
  `transfer_rules.go`).

### Interfaces

- **Accept interfaces, return structs.** Constructors take Gateway interfaces and return a
  concrete `*Service` / `*Provider`.
- **Keep interfaces small.** One to three methods is typical. An interface with one
  implementation and one caller usually shouldn't exist at all — don't add it "for testing"
  unless something is actually faked.
- **Never define an interface preemptively.** Add it when a second implementation or a test
  fake genuinely requires it.

### Errors

- Return errors; **do not panic**. The only acceptable panics are in `main` during startup
  wiring, where failing fast is correct.
- Wrap with context using `%w`: `fmt.Errorf("post double entry: %w", err)`. Lowercase, no
  trailing punctuation, no "failed to" prefix.
- **Never inspect `err.Error()`.** The string is for humans — logs and screens. Any code that
  branches on error text is a bug waiting to happen.
- **Handle each error exactly once.** Handling means inspecting it and making a decision.
  Logging an error *and* returning it is handling it twice: you get duplicate log lines at
  every level of the stack, plus a context-free error at the top. Either annotate and return,
  or log and stop — never both.
- Providers translate vendor/driver errors into domain errors at the boundary. A `pq.Error`,
  gRPC status, or vendor SDK error type must never escape a Provider.

There are **two distinct kinds of error** in this codebase, and they get different treatment:

**1. Business conditions → sentinel errors.** Conditions a caller must be able to distinguish
in order to make a business decision. Define these in `entities` and check with `errors.Is`:

```go
// entities/errors.go
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrLimitExceeded     = errors.New("daily limit exceeded")
	ErrAccountFrozen     = errors.New("account frozen")
)

if errors.Is(err, entities.ErrInsufficientFunds) { /* ... */ }
```

Sentinel errors are **public API** — adding one is an API change, and every package checking
it takes a dependency on `entities`. That's acceptable here precisely because `entities`
already sits at the centre and imports nothing back, so no cycle is possible. Keep the set
small and deliberate; do not add a sentinel for something no caller branches on.

**2. Infrastructure conditions → assert on behaviour, not type.** Whether an operation can be
retried is not a business fact; it's a property of the failure. Do **not** create a shared
error catalogue for this — define a small unexported interface at the point of use and let any
Provider's error satisfy it:

```go
// The interface is unexported and duplicated where needed. No Provider needs to
// import a shared errors package, and no error type needs to be exported.
type retryable interface {
	Retryable() bool
}

// IsRetryable reports whether err indicates a transient failure worth retrying.
func IsRetryable(err error) bool {
	var r retryable
	return errors.As(err, &r) && r.Retryable()
}
```

Each Provider decides for itself what "retryable" means for its own system (a connection reset
from the ledger DB, a 503 from the fraud vendor, a host timeout from core banking) and returns
an error implementing `Retryable() bool`. The caller learns retryability without importing the
Provider, knowing its error types, or matching on strings.

> **Rule of thumb:** if the answer changes a *business* outcome, use a sentinel in `entities`.
> If it changes an *operational* one (retry, back off, dead-letter), assert on behaviour.
> When in doubt, prefer behaviour assertion — it couples less.

### Context

- **Every Gateway method takes `ctx context.Context` as its first parameter.** No exceptions —
  they all perform I/O.
- Entity rules take **no** context. They are pure and do no I/O.
- Never store a context in a struct; pass it through call chains.
- Every outbound call gets a timeout. Providers must respect `ctx` cancellation.

```go
type LedgerGateway interface {
	PostDoubleEntry(ctx context.Context, plan entities.TransferPlan) error
}
```

### Construction & state

- Constructors are named `New` (or `NewX` when a package exports several) and return a pointer.
- **No package-level mutable state, no `init()`, no singletons.** Everything is constructed in
  `main` and injected. This is what makes the whole architecture testable.
- No global loggers, clocks, or config — inject them.

### Concurrency

- The Service channel pattern in §4 is the only place goroutines are started in business code.
- Any shared mutable state needs a mutex or must be confined to a single goroutine; prefer
  confinement.
- CI runs tests with `-race`. Any data race is a build failure, not a warning.

---

## 6. Cross-Cutting Requirements

### Money

- **Never use `float64` for monetary amounts.** Use an integer minor-unit representation
  (`int64` cents/pence) wrapped in an `entities.Money` type carrying its currency.
- `Money` operations must reject mixed-currency arithmetic rather than silently coercing.
- Rounding rules are explicit, documented functions in `entities` — never implicit.

### Time

- **Inject a clock**; never call `time.Now()` inside `entities` or `services`. Define a
  `Clock` interface (`Now() time.Time`) in `entities` and inject a real one in `main`, a fixed
  one in tests. Time-dependent business rules must be deterministically testable.
- All timestamps are **UTC** internally; convert to local only at display boundaries.

### Idempotency & retries

- Every inbound message carries an **idempotency key** (client-supplied or derived). Services
  must be safe to process the same message twice — an at-least-once delivery guarantee on the
  inbound queue means duplicates *will* happen.
- Retries with backoff belong in **Providers**, not Services. A Service's orchestration should
  not know that the ledger driver retries three times.
- Distinguish retryable from terminal errors using the **behaviour assertion** pattern in §5
  (`Retryable() bool`), not a shared error catalogue. The driving side uses this to decide
  between nack-and-redeliver and dead-lettering.

### Correlation & observability

- Every command carries a **correlation ID**, propagated from the inbound message through the
  context into every Gateway call and every log line. Without it, an async channel-based
  system is untraceable.
- Use `log/slog` with structured key/value fields. No `fmt.Println`, no unstructured logging.
- Inject the logger; no package-level logger.
- Log at the boundaries (driving side, Provider calls, Result recording). Don't log inside
  entity rules.

### Sensitive data

- **Never log full account numbers, card PANs, customer names, or amounts tied to an identity.**
  Log account references in a masked or tokenized form, and the correlation ID.
- Implement `LogValue()` (`slog.LogValuer`) on entity types holding sensitive fields so they
  redact themselves by default rather than relying on every call site remembering.
- The same applies to error messages — never interpolate a full account identifier into an
  error that will be logged.

### Configuration

- Read configuration from environment variables **only in `main`**, into an explicit `Config`
  struct. Nothing below `main` reads `os.Getenv`.
- **Fail fast at startup** on missing or invalid config, before any consumer starts.
- No config file parsing scattered through packages; no config lookups at request time.

---

## 7. Cross-Boundary Communication Rules

| Boundary | Rule |
|---|---|
| Service → Service (same feature) | **Never call directly.** If logic must be shared, either pull it down into `entities` (if it's a pure rule) or expose it via a Gateway (if it's a side effect). |
| Feature → Feature | **Never import another feature's `services` or `providers` packages directly.** Communicate via events published/consumed through a Gateway, or via a small shared-kernel package (see below). |
| Shared kernel | A `/internal/shared` package may hold **pure data types** used across features (e.g. `Money`, `AccountID`, `CustomerID`). It must never contain business logic, services, or gateways. |

---

## 8. Testing Strategy

| Layer | Test type | Speed | Real infra? |
|---|---|---|---|
| `entities` (entities + rules) | Plain table tests, pure input → output | microseconds | No |
| `services/<service>` | Unit tests using hand-written fakes of that service's Gateways | milliseconds | No |
| `providers/<system>` | Integration tests against the real system (or `testcontainers`) | seconds | Yes |

- Prefer **table-driven tests** with named subtests (`t.Run`). Use `t.Helper()` in assertion
  helpers, `t.Cleanup()` over `defer` for teardown.
- Fakes for a service's Gateways are hand-written and colocated with the service's test file
  (`_test.go` suffix keeps them out of production builds) unless a Gateway is complex enough
  to warrant a generated mock (`mockgen`), in which case put generated mocks in a sibling
  `mocks/` package.
- Every Provider should have a compile-time assertion (in a `_test.go` file) proving it
  satisfies every Gateway interface it's meant to implement:
  ```go
  var _ transfer.LedgerGateway = (*Provider)(nil)
  var _ entities.AuditGateway  = (*Provider)(nil)
  ```
  Keeping this in a `_test.go` file means the Provider's production build never imports the
  services packages — preserving the inward-only dependency direction.
- Unit tests for services must explicitly cover **partial-failure behavior**. This is the
  highest-value test category in a financial system: if fraud screening passes but the ledger
  post fails, prove the account state was not updated and the failure was still audited and
  notified. Money moving halfway is the failure mode that matters.
- Cover **idempotency**: send the same command twice, prove the side effect happened once.
- Because `Send` is asynchronous and returns nothing, service tests must **`Send` then
  `Close`** — `Close` blocks until the queue is drained, which gives the test a deterministic
  point at which to assert on the fake Gateways. Never assert immediately after `Send`; the
  loop may not have run yet. **Never use `time.Sleep` to wait for async work.**
- Test `Close` itself: that it is idempotent (calling twice does not panic) and that commands
  sent before it are still fully processed rather than dropped.
- Integration tests are gated behind a build tag or `testing.Short()` so the default
  `go test ./...` stays fast.

---

## 9. Tooling & Enforcement

Conventions that aren't mechanically enforced will erode. Set these up before the first
feature is written.

- **`golangci-lint`** with at least: `errcheck`, `govet`, `staticcheck`, `revive`, `ineffassign`,
  `errorlint` (catches `%v`-instead-of-`%w` and bad error comparisons), `bodyclose`,
  `contextcheck`, `noctx`.
- **`depguard`** (or an equivalent import-boundary linter) configured to enforce the
  architecture mechanically, not by review:
  - `entities` may import **stdlib only**.
  - `services/**` may **not** import `providers/**`.
  - Feature A may **not** import Feature B's `services` or `providers`.
- **`go test -race ./...`** in CI. Mandatory given the channel/goroutine design.
- **`gofmt`/`gofumpt`** enforced in CI; no formatting debates.
- **`go vet`** on every build.
- Pin the Go version in `go.mod` and CI. Keep dependencies minimal — prefer the standard
  library, and justify every third-party module in review.
- Record non-obvious architectural decisions as short **ADRs** in `/docs/adr/`. This document
  is the starting point, not the full record.

---

## 10. Worked Example Skeleton

```go
// internal/payments/entities/gateways.go
package entities

import "context"

type AuditGateway interface {
	RecordResult(ctx context.Context, result Result) error
}

type NotificationGateway interface {
	NotifyResult(ctx context.Context, result Result) error
}

type Clock interface {
	Now() time.Time
}

// internal/payments/entities/transfer_rules.go
package entities

// Pure — no I/O, no context, no clock calls. Testable with plain table tests.
func CalculateFee(amount Money, tier AccountTier) (Money, error) { /* ... */ }
func ValidateDailyLimit(acct Account, amount Money) error        { /* ... */ }

// internal/payments/services/transfer/gateways.go
package transfer

type LedgerGateway interface {
	PostDoubleEntry(ctx context.Context, plan entities.TransferPlan) error
}

type FraudGateway interface {
	Screen(ctx context.Context, plan entities.TransferPlan) (approved bool, err error)
}

// internal/payments/services/transfer/service.go
package transfer

type Service struct {
	ledger    LedgerGateway
	fraud     FraudGateway
	audit     entities.AuditGateway
	notifier  entities.NotificationGateway
	clock     entities.Clock
	log       *slog.Logger
	commands  chan Command
	timeout   time.Duration
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func New(
	ledger LedgerGateway,
	fraud FraudGateway,
	audit entities.AuditGateway,
	notifier entities.NotificationGateway,
	clock entities.Clock,
	log *slog.Logger,
	buffer int,
	timeout time.Duration,
) *Service { /* creates channel, starts loop */ }

func (s *Service) Send(cmd Command) { s.commands <- cmd } // the ONLY entry point

func (s *Service) Close() {
	s.closeOnce.Do(func() { close(s.commands) })
	s.wg.Wait()
}

func (s *Service) handle(ctx context.Context, cmd Command) {
	// 1. pure entity rules: ValidateDailyLimit, CalculateFee, build TransferPlan
	// 2. gateways in order: FraudGateway.Screen -> LedgerGateway.PostDoubleEntry
	// 3. record + notify the Result — no error returned to any caller
}

// internal/payments/dispatcher.go
package payments

type Dispatcher interface {
	Transfer(cmd transfer.Command) error       // errors only on "shutting down"
	ChangeLimit(cmd limitchange.Command) error
	Close()                                     // refuses further sends, then drains services
}

// internal/payments/providers/postgres/ledger_provider.go
// No interface declared here — satisfies the service's Gateway structurally.
package postgres

type Provider struct {
	db  *sql.DB
	log *slog.Logger
}

func (p *Provider) PostDoubleEntry(ctx context.Context, plan entities.TransferPlan) error {
	// translate driver errors into entities.Err* sentinels before returning
	return nil
}
```

---

## 11. Checklist Before Adding a New Service or Feature

- [ ] Does this belong in an **existing feature** (shares entities/rules) or is it a **new
      feature** (distinct business capability)?
- [ ] Define the `Command` first — including its correlation ID and idempotency key.
- [ ] Write the pure `entities` rules first, with table tests, before touching any Gateway.
- [ ] Define only the Gateway methods this service actually calls, each taking `ctx` first —
      resist the urge to make an interface "complete" relative to the underlying system.
- [ ] Add the new service's method to the feature's `Dispatcher` interface and its concrete
      implementation.
- [ ] Give the Service a constructor that creates its channel and starts its loop, a `Send`
      method, and an idempotent `Close` — and add its `Close` to the Dispatcher's cascade.
- [ ] Confirm the Service derives a fresh context per command; never reuse the inbound one.
- [ ] Wire the new service in `main.go` — construct its Providers, inject into the `Service`,
      register with the `Dispatcher`, confirm it drains on shutdown.
- [ ] Write unit tests using fakes — happy path, at least one partial-failure path, and a
      duplicate-message idempotency case, asserting after `Close` rather than after `Send`.
- [ ] Write/extend an integration test for any new Provider method, plus a compile-time
      interface assertion.
- [ ] Confirm no sensitive data reaches logs or error strings.

---

## Appendix A. Migrating an Existing Codebase

This document describes a target state. Applying it to a codebase that already exists is a
staged migration, not a single refactor. Read this appendix before starting one.

### A.1 Two rules are not refactors

Most of this document is structural — renames, moves, interface reshaping — and is
behaviour-preserving. Two rules are not:

- **§4 (channel-only entry point)** converts a synchronous call into fire-and-forget. If
  existing callers receive a result or an error, adopting this **changes your API contract**.
  It is a redesign, not a refactor. Decide deliberately whether you want it; it is entirely
  reasonable to adopt everything else and skip §4.
- **§6 (money as integer minor units)** changes stored and transmitted representations. It
  requires a data migration and a compatibility plan for anything already persisted or
  in-flight.

Everything else can be adopted incrementally without changing behaviour.

### A.2 Assess before changing anything

The first task is a gap analysis with no code changes. Ask for: where the codebase already
conforms, where it diverges, the cost of closing each divergence (mechanical rename /
structural move / semantic change), and — importantly — **which rules in this document are a
poor fit for this codebase and why**. A document written for a blank slate will contain rules
that are wrong for a system with history. Amend the document rather than force the fit.

### A.3 Order of operations

Sequenced lowest-risk first. Each stage is its own PR, scoped to **one feature or package at a
time** — never the whole repository at once. Stages 1–3 should be provably behaviour-preserving.

| Stage | Work | Risk |
|---|---|---|
| 1 | **Characterization tests.** Get tests around current behaviour before touching anything, even ugly ones that hit real infrastructure. | — |
| 2 | **Renames only.** Apply §2 vocabulary: `Repository`→`Gateway`, `adapters/`→`providers/`, `domain/`→`entities/`, application-layer `Handler`→`Service`. | Low |
| 3 | **Extract pure rules into `entities`.** Pull business logic out of services and providers into pure, I/O-free functions with table tests. | Low |
| 4 | **Invert the interfaces.** Move Gateway definitions from provider packages to consumer packages; shrink each to only the methods that consumer calls. | Medium |
| 5 | **Introduce the Dispatcher.** Give each feature one inbound entry point. | Medium |
| 6 | **Context, errors, observability.** Thread `ctx` through Gateways, apply the §5 error split, add correlation IDs. | Medium |
| 7 | **Channel lifecycle (§4).** Only if §A.1 was answered yes. | High |

**Never mix a rename commit with a behaviour change.** A stage-2 diff is large and mechanical;
a stage-4 diff is small and semantic. Combined, neither is reviewable, and that is where
behaviour silently changes.

### A.4 Ratchet the linters, don't big-bang them

Enabling `depguard` (§9) against a legacy codebase produces thousands of failures and gets
switched off within a day. Instead, configure the architectural linters to run against an
**allowlist of migrated packages only**. Seed it with the first migrated package; every
subsequent migration PR adds its package to the list.

The property that matters is the ratchet: violations can only decrease, and the build stays
green, so nobody learns to ignore it. Formatting and correctness linters (`gofmt`, `govet`,
`errcheck`) can usually go repository-wide from day one.

### A.5 Keep this document honest during migration

While migrating, this document describes a state that does not yet exist. That is confusing
for humans and actively misleading for agents. Maintain a short status block at the top of the
file recording:

- which packages have migrated, and to which stage;
- which rules have been **deliberately declined**, with the reason;
- which rules are deferred, and what triggers revisiting them.

Delete the block once migration completes.

### A.6 Prompting an agent through a stage

Give the agent one stage and one package, and require a plan before edits:

```
Stage <N> of the ARCHITECTURE.md migration, scoped to internal/<package> only.

Read ARCHITECTURE.md, then Appendix A.3 for what this stage does and does
not cover.

Before changing anything, show me:
  - the exact list of changes you intend to make
  - anything you found that this stage's scope does not cover

Then wait for approval.

Constraints:
  - do not make changes belonging to a later stage
  - no opportunistic improvements outside the stated scope
  - the test suite must pass unchanged; no test assertions may be edited
```

For long sessions, periodically ask the agent to re-read the relevant sections and **list**
violations in its own work without fixing them. Separating detection from correction produces
better results than asking for both at once.
