package main

import (
	"fmt"
	"net/http"

	itementities "github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/items/services/catalog"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// itemNotFoundError wraps entities.ErrItemNotFound with the description
// that wasn't found, so statusForError still maps it via errors.Is while
// the response body stays specific.
func itemNotFoundError(description string) error {
	return fmt.Errorf("%w: %q", itementities.ErrItemNotFound, description)
}

// itemRequest is the request body for POST /items.
type itemRequest struct {
	Reference          string   `json:"reference,omitempty"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	PurchaseDate       string   `json:"purchase_date,omitempty"`
	PurchasePriceCents int64    `json:"purchase_price_cents,omitempty"`
	Currency           string   `json:"currency,omitempty"`
	Vendor             string   `json:"vendor,omitempty"`
	LocationName       *string  `json:"location_name,omitempty"`
	Photos             []string `json:"photos,omitempty"`
}

// itemUpdateRequest is the request body for PATCH /items/{description}.
// Every field is a pointer so an absent JSON key means "keep the item's
// current value" — the same unset-flags-keep-current-value semantics the
// interactive CLI's "item update" implements.
type itemUpdateRequest struct {
	Description        *string   `json:"description,omitempty"`
	Category           *string   `json:"category,omitempty"`
	PurchaseDate       *string   `json:"purchase_date,omitempty"`
	PurchasePriceCents *int64    `json:"purchase_price_cents,omitempty"`
	Currency           *string   `json:"currency,omitempty"`
	Vendor             *string   `json:"vendor,omitempty"`
	LocationName       *string   `json:"location_name,omitempty"`
	Photos             *[]string `json:"photos,omitempty"`
}

// appraisalDTO is one recorded appraisal, as returned on an item response.
type appraisalDTO struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	AsOf        string `json:"as_of"`
	Reference   string `json:"reference,omitempty"`
}

// itemValuationDTO summarizes what's been recorded toward an item's value,
// included on an item response when anything has been recorded.
type itemValuationDTO struct {
	PurchasePrice           moneyDTO       `json:"purchase_price"`
	PurchaseDate            string         `json:"purchase_date"`
	DepreciationRatePercent int            `json:"depreciation_rate_percent"`
	Appraisals              []appraisalDTO `json:"appraisals"`
}

// itemResponse is the wire shape for one catalog item, as returned by
// GET/POST /items and GET /items/{description}.
type itemResponse struct {
	Description        string            `json:"description"`
	Category           string            `json:"category"`
	PurchaseDate       string            `json:"purchase_date"`
	PurchasePriceCents int64             `json:"purchase_price_cents"`
	Currency           string            `json:"currency"`
	Vendor             string            `json:"vendor"`
	LocationName       string            `json:"location_name"`
	Photos             []string          `json:"photos"`
	Archived           bool              `json:"archived"`
	Barcode            string            `json:"barcode"`
	Valuation          *itemValuationDTO `json:"valuation,omitempty"`
}

func newItemResponse(store *jsonstore.Store, item jsonstore.Item) itemResponse {
	resp := itemResponse{
		Description:        item.Description,
		Category:           item.Category,
		PurchaseDate:       item.PurchaseDate,
		PurchasePriceCents: item.PurchasePriceCents,
		Currency:           item.Currency,
		Vendor:             item.Vendor,
		LocationName:       item.LocationName,
		Photos:             item.Photos,
		Archived:           item.Archived,
		Barcode:            item.Barcode,
	}

	if val, ok := store.Valuation(item.Description); ok {
		appraisals := make([]appraisalDTO, 0, len(val.Appraisals))
		for _, a := range val.Appraisals {
			appraisals = append(appraisals, appraisalDTO{
				AmountCents: a.AmountCents,
				Currency:    a.Currency,
				AsOf:        a.AsOf,
				Reference:   a.Reference,
			})
		}
		resp.Valuation = &itemValuationDTO{
			PurchasePrice:           moneyDTO{AmountCents: val.PurchasePriceCents, Currency: val.PurchaseCurrency},
			PurchaseDate:            val.PurchaseDate,
			DepreciationRatePercent: val.DepreciationRatePercent,
			Appraisals:              appraisals,
		}
	}

	return resp
}

// createItem handles POST /items.
func createItem(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req itemRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		reference := req.Reference
		if reference == "" {
			reference = newID()
		}
		currency := req.Currency
		if currency == "" {
			currency = "USD"
		}

		cmd := catalog.Command{
			CorrelationID:      newID(),
			Reference:          reference,
			Action:             catalog.ActionCreate,
			Description:        req.Description,
			Category:           req.Category,
			PurchaseDate:       req.PurchaseDate,
			PurchasePriceCents: req.PurchasePriceCents,
			Currency:           currency,
			Vendor:             req.Vendor,
			LocationName:       req.LocationName,
			Photos:             req.Photos,
		}

		result, dispatchErr := app.submitCatalog(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}

		item, ok := app.store.Item(req.Description)
		if !ok {
			writeJSON(w, http.StatusCreated, outcomeResponse{CorrelationID: result.CorrelationID, Reference: result.Reference})
			return
		}
		writeJSON(w, http.StatusCreated, newItemResponse(app.store, item))
	}
}

// listItems handles GET /items.
func listItems(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		all := app.store.AllItems()
		out := make([]itemResponse, 0, len(all))
		for _, item := range all {
			out = append(out, newItemResponse(app.store, item))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// getItem handles GET /items/{description}.
func getItem(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		description := r.PathValue("description")
		item, ok := app.store.Item(description)
		if !ok {
			writeError(w, itemNotFoundError(description))
			return
		}
		writeJSON(w, http.StatusOK, newItemResponse(app.store, item))
	}
}

// updateItem handles PATCH /items/{description}.
func updateItem(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.PathValue("description")

		current, ok := app.store.Item(target)
		if !ok {
			writeError(w, itemNotFoundError(target))
			return
		}

		var req itemUpdateRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		description := current.Description
		if req.Description != nil {
			description = *req.Description
		}
		category := current.Category
		if req.Category != nil {
			category = *req.Category
		}
		purchaseDate := current.PurchaseDate
		if req.PurchaseDate != nil {
			purchaseDate = *req.PurchaseDate
		}
		priceCents := current.PurchasePriceCents
		if req.PurchasePriceCents != nil {
			priceCents = *req.PurchasePriceCents
		}
		currency := current.Currency
		if req.Currency != nil {
			currency = *req.Currency
		}
		vendor := current.Vendor
		if req.Vendor != nil {
			vendor = *req.Vendor
		}
		locationName := current.LocationName
		if req.LocationName != nil {
			locationName = *req.LocationName
		}
		photos := current.Photos
		if req.Photos != nil {
			photos = *req.Photos
		}

		cmd := catalog.Command{
			CorrelationID:      newID(),
			Action:             catalog.ActionUpdate,
			TargetDescription:  target,
			Description:        description,
			Category:           category,
			PurchaseDate:       purchaseDate,
			PurchasePriceCents: priceCents,
			Currency:           currency,
			Vendor:             vendor,
			Photos:             photos,
		}
		if locationName != "" {
			cmd.LocationName = &locationName
		}

		result, dispatchErr := app.submitCatalog(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}

		item, ok := app.store.Item(description)
		if !ok {
			writeJSON(w, http.StatusOK, outcomeResponse{CorrelationID: result.CorrelationID})
			return
		}
		writeJSON(w, http.StatusOK, newItemResponse(app.store, item))
	}
}

// archiveItem handles POST /items/{description}/archive.
func archiveItem(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.PathValue("description")

		cmd := catalog.Command{
			CorrelationID:     newID(),
			Action:            catalog.ActionArchive,
			TargetDescription: target,
		}

		result, dispatchErr := app.submitCatalog(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}
		writeJSON(w, http.StatusOK, outcomeResponse{CorrelationID: result.CorrelationID})
	}
}

// scanRequest is the request body for POST /items/scan.
type scanRequest struct {
	Barcode string `json:"barcode"`
}

// scanItem handles POST /items/scan. There is no catalog/enrich Command
// that creates the very first draft record — enrich only ever fills gaps
// on an item that already exists. In production this hand-off point is
// where a physical barcode scanner would push a bare, not-yet-described
// item into storage; this endpoint stands in for that as driving-side
// setup, not a business operation — the same role cmd/cli's "item scan"
// plays.
func scanItem(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req scanRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Barcode == "" {
			writeBadRequest(w, "barcode is required")
			return
		}

		if err := app.store.SaveDraft(jsonstore.Item{Barcode: req.Barcode}); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"barcode": req.Barcode})
	}
}

// enrichRequest is the request body for POST /items/enrich.
type enrichRequest struct {
	Reference string `json:"reference,omitempty"`
	Barcode   string `json:"barcode"`
	Target    string `json:"target,omitempty"`
}

// enrichItem handles POST /items/enrich.
func enrichItem(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req enrichRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		reference := req.Reference
		if reference == "" {
			reference = newID()
		}

		cmd := enrich.Command{
			CorrelationID:     newID(),
			Reference:         reference,
			TargetDescription: req.Target,
			Barcode:           req.Barcode,
		}

		result, dispatchErr := app.submitEnrich(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}
		writeJSON(w, http.StatusOK, outcomeResponse{CorrelationID: result.CorrelationID, Reference: result.Reference})
	}
}
