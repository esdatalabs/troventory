package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/esdatalabs/troventory/internal/export/entities"
	report "github.com/esdatalabs/troventory/internal/export/services/insurance-report"
)

func handleExport(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: export <CSV|PDF>")
		return
	}

	format := strings.ToUpper(positional[0])

	cmd := report.Command{
		CorrelationID: newID(),
		Reference:     flags["ref"],
		Format:        format,
	}
	if cmd.Reference == "" {
		cmd.Reference = newID()
	}

	result, dispatchErr := app.submitExport(cmd)
	if dispatchErr != nil {
		fmt.Fprintf(out, "error: %v\n", dispatchErr)
		return
	}
	if result.Err != nil {
		fmt.Fprintf(out, "error: %v\n", result.Err)
		return
	}
	if result.Report == nil {
		fmt.Fprintln(out, "error: no report generated")
		return
	}

	path, err := writeReportFile(*result.Report)
	if err != nil {
		fmt.Fprintf(out, "report generated (%d lines) but failed to write to disk: %v\n", len(result.Report.Lines), err)
		return
	}

	fmt.Fprintf(out, "%s report generated: %d item(s) — written to %s\n", format, len(result.Report.Lines), path)
	if format == "PDF" {
		fmt.Fprintln(out, "note: PDF rendering isn't wired up yet — this is a plain-text stand-in with the same content.")
	}
}

// writeReportFile renders rpt as CSV text (the only rendering this CLI
// implements) and writes it under ./exports, returning the path written.
// A "PDF"-format report still gets this same textual rendering, saved with
// a .txt extension rather than a corrupt-looking .pdf, since no PDF
// renderer is wired up.
func writeReportFile(rpt entities.Report) (string, error) {
	if err := os.MkdirAll("exports", 0o755); err != nil {
		return "", fmt.Errorf("create exports directory: %w", err)
	}

	ext := ".csv"
	if rpt.Format == "PDF" {
		ext = ".txt"
	}
	path := filepath.Join("exports", fmt.Sprintf("insurance-report-%s%s", time.Now().UTC().Format("20060102-150405"), ext))

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

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write report file: %w", err)
	}
	return path, nil
}

func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
