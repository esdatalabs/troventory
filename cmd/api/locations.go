package main

import (
	"fmt"
	"net/http"

	locationentities "github.com/esdatalabs/troventory/internal/locations/entities"
	"github.com/esdatalabs/troventory/internal/locations/services/manage"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// locationNotFoundError wraps entities.ErrLocationNotFound with the name
// that wasn't found, so statusForError still maps it via errors.Is while
// the response body stays specific.
func locationNotFoundError(name string) error {
	return fmt.Errorf("%w: %q", locationentities.ErrLocationNotFound, name)
}

// locationResponse is the wire shape for one location.
type locationResponse struct {
	Name     string `json:"name"`
	Parent   string `json:"parent,omitempty"`
	Archived bool   `json:"archived"`
}

func newLocationResponse(loc jsonstore.Location) locationResponse {
	return locationResponse{Name: loc.Name, Parent: loc.Parent, Archived: loc.Archived}
}

// locationCreateRequest is the request body for POST /locations.
type locationCreateRequest struct {
	Reference string  `json:"reference,omitempty"`
	Name      string  `json:"name"`
	Parent    *string `json:"parent,omitempty"`
}

// createLocation handles POST /locations.
func createLocation(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req locationCreateRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		reference := req.Reference
		if reference == "" {
			reference = newID()
		}

		cmd := manage.Command{
			CorrelationID: newID(),
			Reference:     reference,
			Action:        manage.ActionCreate,
			Name:          req.Name,
			ParentName:    req.Parent,
		}

		result, dispatchErr := app.submitLocation(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}

		loc, ok := app.store.Location(req.Name)
		if !ok {
			writeJSON(w, http.StatusCreated, outcomeResponse{CorrelationID: result.CorrelationID, Reference: result.Reference})
			return
		}
		writeJSON(w, http.StatusCreated, newLocationResponse(loc))
	}
}

// listLocations handles GET /locations.
func listLocations(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		all := app.store.AllLocations()
		out := make([]locationResponse, 0, len(all))
		for _, loc := range all {
			out = append(out, newLocationResponse(loc))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// getLocation handles GET /locations/{name}.
func getLocation(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		loc, ok := app.store.Location(name)
		if !ok {
			writeError(w, locationNotFoundError(name))
			return
		}
		writeJSON(w, http.StatusOK, newLocationResponse(loc))
	}
}

// locationRenameRequest is the request body for POST /locations/{name}/rename.
type locationRenameRequest struct {
	Name string `json:"name"`
}

// renameLocation handles POST /locations/{name}/rename.
func renameLocation(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.PathValue("name")

		var req locationRenameRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		cmd := manage.Command{
			CorrelationID: newID(),
			Action:        manage.ActionRename,
			TargetName:    target,
			Name:          req.Name,
		}

		result, dispatchErr := app.submitLocation(r.Context(), cmd)
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

// locationMoveRequest is the request body for POST /locations/{name}/move.
// A nil or absent Parent moves the location to the top level, the same as
// omitting --parent on the interactive CLI's "location move".
type locationMoveRequest struct {
	Parent *string `json:"parent"`
}

// moveLocation handles POST /locations/{name}/move.
func moveLocation(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.PathValue("name")

		var req locationMoveRequest
		if !decodeJSONOptional(w, r, &req) {
			return
		}

		cmd := manage.Command{
			CorrelationID: newID(),
			Action:        manage.ActionMove,
			TargetName:    target,
			ParentName:    req.Parent,
		}

		result, dispatchErr := app.submitLocation(r.Context(), cmd)
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

// archiveLocation handles POST /locations/{name}/archive.
func archiveLocation(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.PathValue("name")

		cmd := manage.Command{
			CorrelationID: newID(),
			Action:        manage.ActionArchive,
			TargetName:    target,
		}

		result, dispatchErr := app.submitLocation(r.Context(), cmd)
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
