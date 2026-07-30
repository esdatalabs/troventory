package main

import (
	"net/http"
	"time"

	"github.com/esdatalabs/troventory/internal/valuation/services/assess"
)

// today returns the current date as a "YYYY-MM-DD" string, the same
// default cmd/cli's "value price"/"value appraise"/"value current" use
// when --date is omitted.
func today() string {
	return time.Now().UTC().Format("2006-01-02")
}

// valuePriceRequest is the request body for POST /items/{description}/value/price.
type valuePriceRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency,omitempty"`
	Date        string `json:"date,omitempty"`
}

// recordPrice handles POST /items/{description}/value/price.
func recordPrice(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		description := r.PathValue("description")

		var req valuePriceRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		currency := req.Currency
		if currency == "" {
			currency = "USD"
		}
		date := req.Date
		if date == "" {
			date = today()
		}

		cmd := assess.Command{
			CorrelationID:   newID(),
			Action:          assess.ActionRecordPurchasePrice,
			ItemDescription: description,
			AmountCents:     req.AmountCents,
			Currency:        currency,
			Date:            date,
		}

		result, dispatchErr := app.submitValue(r.Context(), cmd)
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

// valueAppraisalRequest is the request body for POST
// /items/{description}/value/appraisals.
type valueAppraisalRequest struct {
	Reference   string `json:"reference,omitempty"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency,omitempty"`
	Date        string `json:"date,omitempty"`
}

// recordAppraisal handles POST /items/{description}/value/appraisals.
func recordAppraisal(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		description := r.PathValue("description")

		var req valueAppraisalRequest
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
		date := req.Date
		if date == "" {
			date = today()
		}

		cmd := assess.Command{
			CorrelationID:   newID(),
			Reference:       reference,
			Action:          assess.ActionRecordAppraisal,
			ItemDescription: description,
			AmountCents:     req.AmountCents,
			Currency:        currency,
			Date:            date,
		}

		result, dispatchErr := app.submitValue(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}
		writeJSON(w, http.StatusCreated, outcomeResponse{CorrelationID: result.CorrelationID, Reference: result.Reference})
	}
}

// depreciationRequest is the request body for PUT
// /items/{description}/value/depreciation-rate.
type depreciationRequest struct {
	RatePercent int `json:"rate_percent"`
}

// setDepreciationRate handles PUT /items/{description}/value/depreciation-rate.
func setDepreciationRate(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		description := r.PathValue("description")

		var req depreciationRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		cmd := assess.Command{
			CorrelationID:           newID(),
			Action:                  assess.ActionConfigureDepreciation,
			ItemDescription:         description,
			DepreciationRatePercent: req.RatePercent,
		}

		result, dispatchErr := app.submitValue(r.Context(), cmd)
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

// currentValueResponse is the JSON body for GET /items/{description}/value/current.
type currentValueResponse struct {
	AsOf  string   `json:"as_of"`
	Value moneyDTO `json:"value"`
}

// getCurrentValue handles GET /items/{description}/value/current.
func getCurrentValue(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		description := r.PathValue("description")

		date := r.URL.Query().Get("date")
		if date == "" {
			date = today()
		}

		cmd := assess.Command{
			CorrelationID:   newID(),
			Action:          assess.ActionComputeCurrentValue,
			ItemDescription: description,
			Date:            date,
		}

		result, dispatchErr := app.submitValue(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}
		if result.Value == nil {
			writeBadRequest(w, "no current value computed")
			return
		}
		writeJSON(w, http.StatusOK, currentValueResponse{
			AsOf:  date,
			Value: moneyDTO{AmountCents: result.Value.AmountCents, Currency: result.Value.Currency},
		})
	}
}
