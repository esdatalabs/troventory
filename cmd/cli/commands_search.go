package main

import (
	"fmt"
	"os"

	"github.com/esdatalabs/troventory/internal/search/services/query"
)

func handleSearch(app *App, out *os.File, args []string) {
	_, flags := parseFlags(args)

	cmd := query.Command{
		CorrelationID:       newID(),
		DescriptionContains: flags["desc"],
		Category:            flags["category"],
		LocationName:        flags["location"],
		Currency:            currencyFlag(flags),
	}

	if v, ok := flags["min"]; ok {
		cents, err := parseCents(v)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return
		}
		cmd.MinValueCents = &cents
	}
	if v, ok := flags["max"]; ok {
		cents, err := parseCents(v)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return
		}
		cmd.MaxValueCents = &cents
	}

	result, dispatchErr := app.submitSearch(cmd)
	if dispatchErr != nil {
		fmt.Fprintf(out, "error: %v\n", dispatchErr)
		return
	}
	if result.Err != nil {
		fmt.Fprintf(out, "error: %v\n", result.Err)
		return
	}

	if len(result.Matches) == 0 {
		fmt.Fprintln(out, "(no matches)")
		return
	}

	fmt.Fprintf(out, "%-30s %-20s %-20s %s\n", "DESCRIPTION", "CATEGORY", "LOCATION", "CURRENT VALUE")
	for _, item := range result.Matches {
		fmt.Fprintf(out, "%-30s %-20s %-20s %s\n",
			truncate(item.Description, 30),
			truncate(item.Category, 20),
			truncate(item.LocationName, 20),
			formatMoneyOrDash(item.CurrentValue.AmountCents, item.CurrentValue.Currency),
		)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
