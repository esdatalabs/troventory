// Command api is the HTTP driving side for Troventory: it wires the same
// in-memory/JSON-file Providers as cmd/cli into each feature's Services and
// Dispatchers (ARCHITECTURE.md §3 rule 3 — this is the only place in this
// binary that imports every Provider package) and serves them over a
// RESTful JSON API instead of a REPL.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/esdatalabs/troventory/internal/export"
	exportentities "github.com/esdatalabs/troventory/internal/export/entities"
	exportaudit "github.com/esdatalabs/troventory/internal/export/providers/audit"
	exportitems "github.com/esdatalabs/troventory/internal/export/providers/items"
	exportstore "github.com/esdatalabs/troventory/internal/export/providers/store"
	exportvaluation "github.com/esdatalabs/troventory/internal/export/providers/valuation"
	report "github.com/esdatalabs/troventory/internal/export/services/insurance-report"

	"github.com/esdatalabs/troventory/internal/items"
	itementities "github.com/esdatalabs/troventory/internal/items/entities"
	itemsaudit "github.com/esdatalabs/troventory/internal/items/providers/audit"
	itemslookup "github.com/esdatalabs/troventory/internal/items/providers/lookup"
	itemsstore "github.com/esdatalabs/troventory/internal/items/providers/store"
	"github.com/esdatalabs/troventory/internal/items/services/catalog"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"

	"github.com/esdatalabs/troventory/internal/locations"
	locationentities "github.com/esdatalabs/troventory/internal/locations/entities"
	locationsaudit "github.com/esdatalabs/troventory/internal/locations/providers/audit"
	locationsstore "github.com/esdatalabs/troventory/internal/locations/providers/store"
	"github.com/esdatalabs/troventory/internal/locations/services/manage"

	"github.com/esdatalabs/troventory/internal/search"
	searchentities "github.com/esdatalabs/troventory/internal/search/entities"
	searchaudit "github.com/esdatalabs/troventory/internal/search/providers/audit"
	searchitems "github.com/esdatalabs/troventory/internal/search/providers/items"
	searchlocations "github.com/esdatalabs/troventory/internal/search/providers/locations"
	searchvaluation "github.com/esdatalabs/troventory/internal/search/providers/valuation"
	"github.com/esdatalabs/troventory/internal/search/services/query"

	"github.com/esdatalabs/troventory/internal/valuation"
	valentities "github.com/esdatalabs/troventory/internal/valuation/entities"
	valuationaudit "github.com/esdatalabs/troventory/internal/valuation/providers/audit"
	valuationitems "github.com/esdatalabs/troventory/internal/valuation/providers/items"
	valuationstore "github.com/esdatalabs/troventory/internal/valuation/providers/store"
	"github.com/esdatalabs/troventory/internal/valuation/services/assess"

	"github.com/esdatalabs/troventory/internal/platform/clock"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

const (
	serviceBuffer  = 8
	serviceTimeout = 5 * time.Second
	awaitTimeout   = 10 * time.Second
)

// App holds every feature's Dispatcher plus the audit Providers the API
// blocks on to learn each command's outcome, and the shared store the read
// endpoints query directly for listings — the same shape as cmd/cli's App,
// wired for a driving side that answers HTTP requests instead of REPL
// lines.
type App struct {
	store *jsonstore.Store

	itemsDispatcher     items.Dispatcher
	locationsDispatcher locations.Dispatcher
	valuationDispatcher valuation.Dispatcher
	searchDispatcher    search.Dispatcher
	exportDispatcher    export.Dispatcher

	itemsAudit     *itemsaudit.Provider
	locationsAudit *locationsaudit.Provider
	valuationAudit *valuationaudit.Provider
	searchAudit    *searchaudit.Provider
	exportAudit    *exportaudit.Provider
}

// NewApp opens the data file at dataPath (and product catalog file at
// productsPath, which may be "") and wires up every feature.
func NewApp(dataPath, productsPath string, log *slog.Logger) (*App, error) {
	store, err := jsonstore.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("open data store: %w", err)
	}

	realClock := clock.Real{}

	app := &App{store: store}

	// items: catalog + enrich
	catalogStorage := itemsstore.NewCatalogStorage(store)
	catalogLocations := itemsstore.NewCatalogLocations(store)
	enrichItems := itemsstore.NewEnrichItems(store)
	productLookup, err := itemslookup.New(productsPath)
	if err != nil {
		return nil, fmt.Errorf("load product catalog: %w", err)
	}
	app.itemsAudit = itemsaudit.New()

	catalogService := catalog.New(catalogStorage, catalogLocations, app.itemsAudit, realClock, log, serviceBuffer, serviceTimeout)
	enrichService := enrich.New(enrichItems, productLookup, app.itemsAudit, realClock, log, serviceBuffer, serviceTimeout)
	app.itemsDispatcher = items.NewDispatcher(catalogService, enrichService)

	// locations: manage
	locationStorage := locationsstore.New(store)
	app.locationsAudit = locationsaudit.New()
	manageService := manage.New(locationStorage, app.locationsAudit, realClock, log, serviceBuffer, serviceTimeout)
	app.locationsDispatcher = locations.NewDispatcher(manageService)

	// valuation: assess
	valuationStorage := valuationstore.New(store)
	valuationItemsGateway := valuationitems.New(store)
	app.valuationAudit = valuationaudit.New()
	assessService := assess.New(valuationItemsGateway, valuationStorage, app.valuationAudit, realClock, log, serviceBuffer, serviceTimeout)
	app.valuationDispatcher = valuation.NewDispatcher(assessService)

	// search: query
	searchItemsGateway := searchitems.New(store)
	searchLocationsGateway := searchlocations.New(store)
	searchValueGateway := searchvaluation.New(store, realClock)
	app.searchAudit = searchaudit.New()
	queryService := query.New(searchItemsGateway, searchLocationsGateway, searchValueGateway, app.searchAudit, realClock, log, serviceBuffer, serviceTimeout)
	app.searchDispatcher = search.NewDispatcher(queryService)

	// export: insurance-report
	exportItemsGateway := exportitems.New(store)
	exportValuationGateway := exportvaluation.New(store, realClock)
	exportStorage := exportstore.New(store)
	app.exportAudit = exportaudit.New()
	reportService := report.New(exportItemsGateway, exportValuationGateway, exportStorage, app.exportAudit, realClock, log, serviceBuffer, serviceTimeout)
	app.exportDispatcher = export.NewDispatcher(reportService)

	return app, nil
}

// Close drains and stops every feature's Dispatcher.
func (a *App) Close() {
	a.itemsDispatcher.Close()
	a.locationsDispatcher.Close()
	a.valuationDispatcher.Close()
	a.searchDispatcher.Close()
	a.exportDispatcher.Close()
}

// submitLocation sends cmd to the locations Dispatcher and blocks for its
// Result.
func (a *App) submitLocation(ctx context.Context, cmd manage.Command) (locationentities.Result, error) {
	if err := a.locationsDispatcher.ManageLocation(cmd); err != nil {
		return locationentities.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, awaitTimeout)
	defer cancel()
	return a.locationsAudit.Await(ctx, cmd.CorrelationID)
}

// submitCatalog sends cmd to the items Dispatcher's catalog service and
// blocks for its Result.
func (a *App) submitCatalog(ctx context.Context, cmd catalog.Command) (itementities.Result, error) {
	if err := a.itemsDispatcher.ManageItem(cmd); err != nil {
		return itementities.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, awaitTimeout)
	defer cancel()
	return a.itemsAudit.Await(ctx, cmd.CorrelationID)
}

// submitEnrich sends cmd to the items Dispatcher's enrich service and
// blocks for its Result.
func (a *App) submitEnrich(ctx context.Context, cmd enrich.Command) (itementities.Result, error) {
	if err := a.itemsDispatcher.EnrichItem(cmd); err != nil {
		return itementities.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, awaitTimeout)
	defer cancel()
	return a.itemsAudit.Await(ctx, cmd.CorrelationID)
}

// submitValue sends cmd to the valuation Dispatcher and blocks for its
// Result.
func (a *App) submitValue(ctx context.Context, cmd assess.Command) (valentities.Result, error) {
	if err := a.valuationDispatcher.AssessValue(cmd); err != nil {
		return valentities.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, awaitTimeout)
	defer cancel()
	return a.valuationAudit.Await(ctx, cmd.CorrelationID)
}

// submitSearch sends cmd to the search Dispatcher and blocks for its
// Result.
func (a *App) submitSearch(ctx context.Context, cmd query.Command) (searchentities.Result, error) {
	if err := a.searchDispatcher.Search(cmd); err != nil {
		return searchentities.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, awaitTimeout)
	defer cancel()
	return a.searchAudit.Await(ctx, cmd.CorrelationID)
}

// submitExport sends cmd to the export Dispatcher and blocks for its
// Result.
func (a *App) submitExport(ctx context.Context, cmd report.Command) (exportentities.Result, error) {
	if err := a.exportDispatcher.ExportInsuranceReport(cmd); err != nil {
		return exportentities.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, awaitTimeout)
	defer cancel()
	return a.exportAudit.Await(ctx, cmd.CorrelationID)
}
