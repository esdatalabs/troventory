# Troventory

A home inventory application — track everything you own, where it lives, and what it's worth.

## Status

🚧 Early development. This repository currently contains project scaffolding; no features are
implemented yet.

## What it does (planned)

- Catalog personal belongings with photos, descriptions, and purchase details
- Organize items by location (room, container, storage unit) and category
- Track valuation for insurance and estate-planning purposes
- Look up items by barcode/UPC
- Search and filter across your entire inventory
- Export inventory data (e.g. for insurance claims)

This list is a starting point, not a spec — expect it to firm up as the project takes shape.

## Architecture

This project follows the conventions defined in [`ARCHITECTURE.md`](./ARCHITECTURE.md): a
hexagonal architecture where dependencies point inward toward business logic, using a small,
consistent vocabulary — Entity, Gateway, Provider, Service, Dispatcher — instead of the usual
ports/adapters terminology. Read it before implementing a new feature.

`ARCHITECTURE.md`'s worked examples use a `payments` domain for illustration; for this project,
the equivalent features are things like `items`, `locations`, and `valuation`.

## Tech stack

- **Language:** Go
- **Architecture:** Hexagonal (see `ARCHITECTURE.md`)
- **Acceptance testing:** [godog](https://github.com/cucumber/godog) (Cucumber/Gherkin for Go)
- **Everything else** (storage, frontend, deployment) — not yet decided

## Getting started

This section will fill in as the project takes shape. For now:

```bash
git clone https://github.com/<you>/troventory.git
cd troventory
go mod tidy
go test ./...
```

## Development workflow

New features go through a red/green TDD pipeline: a plain-language prompt becomes Gherkin
scenarios, then a failing (compile-error) acceptance test, then an implementation built against
`ARCHITECTURE.md`, then a passing test. If the pipeline scaffold has been copied into this repo,
run a full cycle for a new feature with:

```
/tdd-cycle <feature> <service> "<description of the behavior>"
```

See `.claude/commands/tdd-cycle.md` and the scaffold's own README for details.

## Contributing

Not yet open for contributions — this is a personal project in early scaffolding. That may
change.

## License

TBD.