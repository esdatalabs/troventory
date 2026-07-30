# Troventory

A home inventory application — track everything you own, where it lives, and what it's worth.

## Status

🚧 Early development. The five features below (`items`, `locations`, `valuation`, `search`,
`export`) are implemented and covered by acceptance tests. There are now three driving sides in
front of them, all backed by the same JSON file on disk: a runnable
[interactive CLI](#using-the-interactive-cli), a [RESTful HTTP API](#using-the-rest-api), and a
[Vue dashboard](#using-the-web-dashboard) for demoing the whole thing in a browser. Still expect
rough edges — this is a personal project taking shape incrementally.

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
- **Persistence:** a single JSON file on disk (`internal/platform/jsonstore`) — a real database is
  still an open decision, see `PLAN.md`
- **Frontend:** [Vue 3](https://vuejs.org) + [Vite](https://vitejs.dev), under `web/` — talks to
  the REST API over HTTP; not yet decided whether it stays or a server-rendered approach replaces
  it
- **Everything else** (deployment) — not yet decided

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

## Using the REST API

`cmd/api` wires the same Providers, Services, and Dispatchers as `cmd/cli` — the same JSON data
file, the same business rules — behind a JSON HTTP API instead of a REPL, so anything you can do
from the CLI you can do from a script, a browser, or the [web dashboard](#using-the-web-dashboard).

### Run it

```bash
go run ./cmd/api
```

Same flags as the CLI (`--data`, `--log`), plus `--addr` (default `:8080`) for the listen
address. Requests/responses are JSON; money amounts are always integer cents, per
`ARCHITECTURE.md` §6. A permissive CORS policy is enabled so a locally-run frontend on a
different port can call it directly — this is a local demo server with no auth, so that trades
away nothing.

### Endpoints

| Method & path | Does |
|---|---|
| `POST /locations` | create a location (`{name, parent?}`) |
| `GET /locations` | list every location |
| `GET /locations/{name}` | show one location |
| `POST /locations/{name}/rename` | rename (`{name}`) |
| `POST /locations/{name}/move` | move (`{parent}`; omit/`null` = top level) |
| `POST /locations/{name}/archive` | archive |
| `POST /items` | create an item |
| `GET /items` | list every item (including archived) |
| `GET /items/{description}` | show one item, with its valuation if recorded |
| `PATCH /items/{description}` | update — omitted fields keep their current value |
| `POST /items/{description}/archive` | archive |
| `POST /items/scan` | seed a draft item from a barcode (`{barcode}`) |
| `POST /items/enrich` | fill gaps from a barcode lookup (`{barcode, target?}`) |
| `POST /items/{description}/value/price` | record purchase price (`{amount_cents, currency?, date?}`) |
| `POST /items/{description}/value/appraisals` | record an appraisal (same body, plus optional `reference`) |
| `PUT /items/{description}/value/depreciation-rate` | set the straight-line rate (`{rate_percent}`) |
| `GET /items/{description}/value/current?date=` | compute current value as of a date (defaults to today) |
| `GET /search?desc=&category=&location=&min=&max=` | filter active items; no params lists everything |
| `GET /export?format=CSV\|PDF` | stream a generated insurance report as a file download |
| `GET /healthz` | liveness check |

Every mutating endpoint accepts an optional `"reference"` field in its JSON body as an
idempotency key, the HTTP equivalent of the CLI's `--ref`. Business/validation failures come back
as a JSON `{"error": "..."}` body with a matching HTTP status (404 not found, 409 conflict, 400
validation, 502 for an unreachable barcode lookup).

## Using the web dashboard

`web/` is a small Vue 3 + Vite single-page app for demoing the app in a browser: a location
hierarchy, an item catalog with valuation, search/filter, and one-click CSV/PDF export. It talks
to `cmd/api` over HTTP and has no server of its own.

```bash
go run ./cmd/api                 # in one terminal — the API, default :8080
cd web && npm install && npm run dev   # in another — the dashboard, default :5173
```

Point it at a different API by copying `web/.env.example` to `web/.env` and setting
`VITE_API_BASE`. `npm run build` produces a static `web/dist` you can serve from anything.

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