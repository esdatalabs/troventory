package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/esdatalabs/troventory/internal/export/entities"
	report "github.com/esdatalabs/troventory/internal/export/services/insurance-report"
)

// exportReport handles GET /export?format=CSV|PDF, streaming the generated
// report back as a file download rather than writing it to disk — an API
// server has no natural place on its host to keep it for the client the
// way the interactive CLI's local ./exports directory does.
func exportReport(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		format := strings.ToUpper(r.URL.Query().Get("format"))
		if format == "" {
			format = "CSV"
		}

		reference := r.URL.Query().Get("reference")
		if reference == "" {
			reference = newID()
		}

		cmd := report.Command{
			CorrelationID: newID(),
			Reference:     reference,
			Format:        format,
		}

		result, dispatchErr := app.submitExport(r.Context(), cmd)
		if dispatchErr != nil {
			writeError(w, dispatchErr)
			return
		}
		if result.Err != nil {
			writeError(w, result.Err)
			return
		}
		if result.Report == nil {
			writeBadRequest(w, "no report generated")
			return
		}

		body := renderReportCSV(*result.Report)

		ext := ".csv"
		contentType := "text/csv; charset=utf-8"
		if result.Report.Format == "PDF" {
			// PDF rendering isn't wired up yet — this is a plain-text
			// stand-in with the same content, the same caveat cmd/cli's
			// "export PDF" carries.
			ext = ".txt"
			contentType = "text/plain; charset=utf-8"
		}
		filename := fmt.Sprintf("insurance-report-%s%s", time.Now().UTC().Format("20060102-150405"), ext)

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}
}

// renderReportCSV renders rpt as CSV text — the same rendering cmd/cli's
// "export" writes to disk, reused here to stream directly in the HTTP
// response.
func renderReportCSV(rpt entities.Report) string {
	var b strings.Builder
	b.WriteString("description,category,location,purchase_price,purchase_date,current_value,current_value_as_of\n")
	for _, line := range rpt.Lines {
		currentValue := ""
		if line.CurrentValue != nil {
			currentValue = formatCents(line.CurrentValue.AmountCents, line.CurrentValue.Currency)
		}
		b.WriteString(csvField(line.ItemDescription))
		b.WriteString(",")
		b.WriteString(csvField(line.Category))
		b.WriteString(",")
		b.WriteString(csvField(line.LocationName))
		b.WriteString(",")
		b.WriteString(csvField(formatMoneyOrDash(line.PurchasePrice.AmountCents, line.PurchasePrice.Currency)))
		b.WriteString(",")
		b.WriteString(csvField(line.PurchaseDate))
		b.WriteString(",")
		b.WriteString(csvField(currentValue))
		b.WriteString(",")
		b.WriteString(csvField(line.CurrentValueAsOf))
		b.WriteString("\n")
	}
	return b.String()
}

func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// formatCents renders amountCents/currency as a human-readable "12.34 USD"
// string.
func formatCents(amountCents int64, currency string) string {
	neg := ""
	if amountCents < 0 {
		neg = "-"
		amountCents = -amountCents
	}
	return fmt.Sprintf("%s%d.%02d %s", neg, amountCents/100, amountCents%100, currency)
}

// formatMoneyOrDash is formatCents, except an empty currency (no value
// recorded/computed) renders as "—" rather than a misleading "0.00".
func formatMoneyOrDash(amountCents int64, currency string) string {
	if currency == "" {
		return "—"
	}
	return formatCents(amountCents, currency)
}
