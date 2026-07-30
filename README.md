# Troventory

A home inventory application — track everything you own, where it lives, and what it's worth.

## Status

🚧 Early development. The five features below (`items`, `locations`, `valuation`, `search`,
`export`) are implemented and covered by acceptance tests, and there's now a runnable
[interactive CLI](#using-the-interactive-cli) wired up in front of them, backed by a JSON file
on disk. Still expect rough edges — this is a personal project taking shape incrementally.

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

```bash
git clone https://github.com/<you>/troventory.git
cd troventory
go mod tidy
go test ./...
```

## Using the interactive CLI

`cmd/cli` wires every feature's Service and Dispatcher up to a small set of Providers backed by
a single JSON file on disk, and puts an interactive prompt in front of them. It's the fastest way
to actually exercise the app.

### Run it

```bash
go run ./cmd/cli
```

By default this reads/writes `~/.troventory/data.json`, so your inventory persists between runs.
Override any of the paths with flags:

```bash
go run ./cmd/cli \
  --data     ~/.troventory/data.json \    # inventory data (JSON)
  --log      ~/.troventory/cli.log \      # structured debug logs (kept out of the terminal)
  --products ~/.troventory/products.json  # optional: extra barcode/UPC entries for "item enrich"
```

You'll get a `>` prompt. Type `help` at any time for the full command reference, and `exit` (or
`quit`) to leave.

### A quick walkthrough

```
> location add Garage
> location add "Workbench" --parent Garage
> item add "Cordless Drill" --category Tools --date 2022-03-01 --price 129.99 --currency USD --vendor "Home Depot" --location Workbench
> value price "Cordless Drill" 129.99 --date 2022-03-01
> value depreciate "Cordless Drill" 10
> value current "Cordless Drill"
> search --category Tools
> export CSV
```

- **Locations:** `location add|rename|move|archive|list` — a nested room → container → shelf
  hierarchy.
- **Items:** `item add|update|archive|show` for the catalog, plus `item scan <barcode>` +
  `item enrich <barcode>` to simulate scanning a not-yet-described item and filling in its
  description/category/photo from a barcode lookup. The lookup is a small offline sample catalog
  (see `internal/items/providers/lookup`) — extend it by pointing `--products` at your own JSON
  file of `{barcode, description, category, photo}` entries.
- **Valuation:** `value price|appraise|depreciate|current` — record a purchase price, record
  appraisals over time, configure a straight-line depreciation rate, and compute an item's value
  as of any date.
- **Search:** `search [--desc SUBSTRING] [--category CAT] [--location NAME] [--min DOLLARS]
  [--max DOLLARS]` — no flags lists every active item.
- **Export:** `export CSV` (or `export PDF`, which renders the same content as a plain-text
  stand-in — no PDF renderer is wired up yet) writes an insurance report under `./exports`.

Commands that create or change something (`add`, `appraise`, `export`, ...) accept an optional
`--ref <value>` idempotency key; resubmitting the same one is a no-op rather than a duplicate.

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