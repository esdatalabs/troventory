Mapping the README's six features onto `ARCHITECTURE.md`'s Feature/Service hierarchy first, since that's what determines sequencing — some of these are one Feature each, some are Services within a shared Feature, and a couple need a cross-Feature boundary decision before anyone writes code.

## Decisions to lock in before Phase 1

These aren't backend logic decisions, they're infrastructure choices that change how services are structured — worth an ADR each per `ARCHITECTURE.md` §9 rather than deciding implicitly mid-implementation:

| Decision | Why it matters now |
|---|---|
| Primary datastore (Postgres assumed, not confirmed) | Every `Provider` in Phase 1+ targets it |
| Photo storage (local disk vs. S3-compatible object store) | Affects `items`' `PhotoGateway` shape |
| Barcode/UPC data provider (which third-party API) | Affects `items`' enrichment `Gateway` shape |
| Search backend (Postgres full-text vs. a dedicated engine like Meilisearch/Typesense) | Determines whether `search` needs its own read-model/projector or can query items' store directly |

## Feature/Service decomposition

| Phase | Feature | Service(s) | Depends on | Covers README feature |
|---|---|---|---|---|
| 1 | `locations` | `locations/manage` — create/rename/move/archive a location (room → container → shelf nesting) | shared kernel only | "Organize by location" (half) |
| 2 | `items` | `items/catalog` — create/update/archive an item: photos, description, purchase details, category, location assignment | `locations` (by ID only, via shared kernel — see boundary note below) | "Catalog belongings" + "Organize by location/category" |
| 3 | `items` | `items/enrich` — populate a draft item from a barcode/UPC lookup | `items/catalog` | "Barcode/UPC lookup" |
| 4 | `valuation` | `valuation/assess` — record purchase price, appraisals, depreciation; compute current value | `items` (read-only) | "Valuation for insurance/estate" |
| 5 | `search` | `search/query` — search/filter across items by name, category, location, value | `items`, `locations`, `valuation` (read-only) | "Search and filter" |
| 6 | `export` | `export/insurance-report` — compile item + valuation + location data into CSV/PDF | `items`, `locations`, `valuation`, `search` | "Export inventory data" |

Two decomposition calls worth flagging as I made them, since a team might reasonably want it differently: **categories** are folded into `items/catalog` rather than given their own Feature — they don't have enough independent lifecycle (no rename/merge workflow implied by the README) to justify a Dispatcher of their own yet. And **export** is its own Feature rather than a service tacked onto `valuation`, since it's genuinely a different use case (reporting) reading across three other Features' data.

## The one real architectural wrinkle: cross-Feature reads

`ARCHITECTURE.md` §7 is explicit that a Feature can't import another Feature's `services` or `providers` packages directly. `valuation`, `search`, and `export` all need to read data `items` owns, so this needs a deliberate answer before Phase 4, not an improvised one:

- **Cheapest**: each downstream Feature defines its own narrow read-only Gateway (e.g. `valuation` defines `ItemLookupGateway` with exactly the method it needs), and a Provider implements it by querying the same Postgres database `items` writes to. Simple, but means two Features share a schema implicitly.
- **More decoupled**: `items` publishes domain events (`ItemCreated`, `ItemUpdated`) that `valuation` and `search` consume to build their own local read models. Costs more upfront, avoids schema coupling, and is what a search read-model basically requires anyway once you're past Postgres full-text.

Given `search` likely needs its own optimized read-model regardless of which way you go, I'd lean toward the event approach and reuse the same mechanism for `valuation`'s read access — but this is a real trade-off the team should decide, not something to default silently.

## How each phase actually runs

Every Service in the table goes through the pipeline already built: `/tdd-cycle <feature> <service> "<description>"` for Gherkin → failing test → implementation against fakes. Real Providers (Postgres, object storage, the barcode API, the search engine) are a deliberately separate pass per Feature, built only once that Feature's scenarios are green against fakes — this stays true to how the pipeline's `feature-implementer` is scoped (it never touches `providers/`).

Exit criteria per phase, straight from `ARCHITECTURE.md` §11: entities table-tested, service unit tests cover the happy path + partial failure + idempotency, compile-time Gateway assertions exist, and `main.go` wiring drains cleanly on shutdown before moving to the next phase.

Want me to turn the cross-Feature boundary decision into an ADR draft, or start Phase 1 (`locations`) with an actual `/tdd-cycle` prompt?