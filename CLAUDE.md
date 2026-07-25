# Troventory

<!-- Maintainer note: this file is a first draft based on project conversations, not `/init`
output. Once the repo has real Go code, run `/init` and merge its findings into this file
rather than replacing it — the workflow and architecture context here won't come from a
codebase scan. -->

## About this project

Troventory is a home inventory application: tracking personal belongings, their locations,
categories, and value. Currently in early scaffolding — most of the codebase doesn't exist yet.

## Architecture

Read `ARCHITECTURE.md` before touching any code. It defines this project's hexagonal
architecture and vocabulary: Entity, Gateway, Provider, Service, Dispatcher. Do not use
"adapter," "repository," "port," or "handler" for service-layer code — see `ARCHITECTURE.md` §2
for the full "do not use" list.

`ARCHITECTURE.md`'s worked examples use a `payments` domain; for this project the equivalent
features are things like `items`, `locations`, and `valuation`.

## Development workflow

New features go through a red/green TDD pipeline, not ad-hoc implementation:

1. Gherkin scenarios first (`internal/<feature>/acceptance/features/<service>.feature`)
2. A failing, compile-error-red test (step defs + fakes + a symbol manifest)
3. Implementation against the manifest, following `ARCHITECTURE.md`
4. Full regression

Run a cycle with `/tdd-cycle <feature> <service> "<description>"` rather than writing a new
service's entities and tests by hand in a single pass. See `.claude/commands/tdd-cycle.md` and
the scaffold's own README for exactly what each stage does.

**Never edit anything under `acceptance/` or a `.feature` file to make a test pass.** If a test
looks wrong, say so and stop instead of weakening it.

## Commands

<!-- TBD: fill in once go.mod exists — run /init to detect these automatically. -->
- `go build ./...` — build everything
- `go test ./...` — unit + acceptance tests
- `go test -race ./...` — full regression, required before considering a feature done
- `golangci-lint run ./...` — lint, including the `depguard` hexagon-boundary check

## Code style

Follow `ARCHITECTURE.md` §5 (Go conventions): short lowercase package names, no stutter
(`transfer.Service`, not `transfer.TransferService`), accept interfaces / return structs, errors
wrapped with `%w` and handled exactly once, sentinel errors in `entities` for business
conditions, behavior-assertion (`Retryable() bool`) for infrastructure conditions.

## Repository etiquette

<!-- TBD: no team conventions yet — this is a solo/early project. Update once decided
(branch naming, merge vs. rebase, PR process). -->

## Status

Pre-alpha. Expect parts of this file to be wrong until there's enough real code for `/init` to
cross-check it against.