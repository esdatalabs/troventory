package main

import (
	"net/http"
	"strconv"

	"github.com/esdatalabs/troventory/internal/search/services/query"
)

// searchMatchDTO is one matched item on a search response.
type searchMatchDTO struct {
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	LocationName string   `json:"location_name,omitempty"`
	Archived     bool     `json:"archived"`
	CurrentValue moneyDTO `json:"current_value,omitempty"`
}

// searchResponse is the JSON body for GET /search.
type searchResponse struct {
	Matches []searchMatchDTO `json:"matches"`
}

// searchItems handles GET /search. Every filter is an optional query
// parameter; no filters at all lists everything, the same as the
// interactive CLI's bare "search".
func searchItems(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		cmd := query.Command{
			CorrelationID:       newID(),
			DescriptionContains: q.Get("desc"),
			Category:            q.Get("category"),
			LocationName:        q.Get("location"),
			Currency:            q.Get("currency"),
		}
		if currency := cmd.Currency; currency == "" {
			cmd.Currency = "USD"
		}

		if v := q.Get("min"); v != "" {
			cents, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				writeBadRequest(w, "invalid min value")
				return
			}
			cmd.MinValueCents = &cents
		}
		if v := q.Get("max"); v != "" {
			cents, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				writeBadRequest(w, "invalid max value")
				return
			}
			cmd.MaxValueCents = &cents
		}

		result, dispatchErr := app.submitSearch(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}

		matches := make([]searchMatchDTO, 0, len(result.Matches))
		for _, item := range result.Matches {
			matches = append(matches, searchMatchDTO{
				Description:  item.Description,
				Category:     item.Category,
				LocationName: item.LocationName,
				Archived:     item.Archived,
				CurrentValue: moneyDTO{AmountCents: item.CurrentValue.AmountCents, Currency: item.CurrentValue.Currency},
			})
		}
		writeJSON(w, http.StatusOK, searchResponse{Matches: matches})
	}
}
