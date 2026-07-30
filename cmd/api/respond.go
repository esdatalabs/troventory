package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	exportentities "github.com/esdatalabs/troventory/internal/export/entities"
	itementities "github.com/esdatalabs/troventory/internal/items/entities"
	locationentities "github.com/esdatalabs/troventory/internal/locations/entities"
	searchentities "github.com/esdatalabs/troventory/internal/search/entities"
	valentities "github.com/esdatalabs/troventory/internal/valuation/entities"
)

// sentinelStatus pairs every business sentinel error this API's features
// can report with the HTTP status that best represents it, so handlers
// never have to know a feature's error catalog themselves — see
// statusForError.
var sentinelStatus = []struct {
	err    error
	status int
}{
	{itementities.ErrItemNotFound, http.StatusNotFound},
	{itementities.ErrItemArchived, http.StatusConflict},
	{itementities.ErrItemDescriptionRequired, http.StatusBadRequest},
	{itementities.ErrItemCategoryRequired, http.StatusBadRequest},
	{itementities.ErrAssignedLocationNotFound, http.StatusBadRequest},
	{itementities.ErrAssignedLocationArchived, http.StatusBadRequest},
	{itementities.ErrBarcodeInvalid, http.StatusBadRequest},
	{itementities.ErrProductNotFound, http.StatusNotFound},
	{itementities.ErrProductLookupUnavailable, http.StatusBadGateway},

	{locationentities.ErrLocationNotFound, http.StatusNotFound},
	{locationentities.ErrLocationArchived, http.StatusConflict},
	{locationentities.ErrDuplicateLocationName, http.StatusConflict},
	{locationentities.ErrCyclicMove, http.StatusConflict},
	{locationentities.ErrLocationHasActiveChildren, http.StatusConflict},

	{valentities.ErrItemNotFound, http.StatusNotFound},
	{valentities.ErrPurchasePriceNotPositive, http.StatusBadRequest},
	{valentities.ErrAppraisalNotPositive, http.StatusBadRequest},
	{valentities.ErrAppraisalOutOfOrder, http.StatusConflict},
	{valentities.ErrNoValuationRecorded, http.StatusNotFound},

	{searchentities.ErrInvalidValueRange, http.StatusBadRequest},
	{searchentities.ErrLocationNotFound, http.StatusBadRequest},

	{exportentities.ErrUnsupportedFormat, http.StatusBadRequest},
	{exportentities.ErrNoItemsToExport, http.StatusConflict},
}

// statusForError maps a business/infrastructure error to the HTTP status
// that best represents it, falling back to 500 for anything not in
// sentinelStatus (dispatcher-closed errors, context deadlines, etc.).
func statusForError(err error) int {
	for _, s := range sentinelStatus {
		if errors.Is(err, s.err) {
			return s.status
		}
	}
	return http.StatusInternalServerError
}

// writeJSON writes v as a JSON response body with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// errorResponse is the JSON body shape for every failed request.
type errorResponse struct {
	Error string `json:"error"`
}

// writeError writes err as a JSON error body, choosing the status via
// statusForError.
func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
}

// writeBadRequest writes msg as a 400 JSON error body, for request-shape
// problems (bad JSON, missing required field) that never reach a Command.
func writeBadRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, errorResponse{Error: msg})
}

// decodeJSON decodes the request body into v, reporting a 400 on failure.
// It returns false if decoding failed and the caller should stop handling
// the request.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		writeBadRequest(w, "request body is required")
		return false
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return false
	}
	return true
}

// decodeJSONOptional is decodeJSON, except a genuinely empty body is not an
// error — v is left at its zero value. Used by endpoints where every field
// is optional (e.g. "move to top level" needs no body at all).
func decodeJSONOptional(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		return true
	}
	br := bufio.NewReader(r.Body)
	if _, err := br.Peek(1); err != nil {
		if err == io.EOF {
			return true
		}
		writeBadRequest(w, "invalid request body: "+err.Error())
		return false
	}

	dec := json.NewDecoder(br)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return false
	}
	return true
}
