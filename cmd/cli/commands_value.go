package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/esdatalabs/troventory/internal/valuation/services/assess"
)

func handleValue(app *App, out *os.File, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(out, "usage: value <price|appraise|depreciate|current> ...")
		return
	}

	sub, rest := args[0], args[1:]
	switch sub {
	case "price":
		valuePrice(app, out, rest)
	case "appraise":
		valueAppraise(app, out, rest)
	case "depreciate":
		valueDepreciate(app, out, rest)
	case "current":
		valueCurrent(app, out, rest)
	default:
		fmt.Fprintf(out, "unknown value subcommand %q\n", sub)
	}
}

func valuePrice(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 2 {
		fmt.Fprintln(out, "usage: value price <description> <amount> [--currency USD] [--date YYYY-MM-DD]")
		return
	}

	cents, err := parseCents(positional[1])
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return
	}

	date := flags["date"]
	if date == "" {
		date = today()
	}

	cmd := assess.Command{
		CorrelationID:   newID(),
		Action:          assess.ActionRecordPurchasePrice,
		ItemDescription: positional[0],
		AmountCents:     cents,
		Currency:        currencyFlag(flags),
		Date:            date,
	}

	result, dispatchErr := app.submitValue(cmd)
	printOutcome(out, dispatchErr, result.Err, "purchase price recorded for %q", positional[0])
}

func valueAppraise(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 2 {
		fmt.Fprintln(out, "usage: value appraise <description> <amount> [--currency USD] [--date YYYY-MM-DD]")
		return
	}

	cents, err := parseCents(positional[1])
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return
	}

	date := flags["date"]
	if date == "" {
		date = today()
	}

	cmd := assess.Command{
		CorrelationID:   newID(),
		Reference:       flags["ref"],
		Action:          assess.ActionRecordAppraisal,
		ItemDescription: positional[0],
		AmountCents:     cents,
		Currency:        currencyFlag(flags),
		Date:            date,
	}
	if cmd.Reference == "" {
		cmd.Reference = newID()
	}

	result, dispatchErr := app.submitValue(cmd)
	printOutcome(out, dispatchErr, result.Err, "appraisal recorded for %q", positional[0])
}

func valueDepreciate(app *App, out *os.File, args []string) {
	positional, _ := parseFlags(args)
	if len(positional) < 2 {
		fmt.Fprintln(out, "usage: value depreciate <description> <rate-percent>")
		return
	}

	rate, err := strconv.Atoi(positional[1])
	if err != nil {
		fmt.Fprintf(out, "error: invalid rate %q\n", positional[1])
		return
	}

	cmd := assess.Command{
		CorrelationID:           newID(),
		Action:                  assess.ActionConfigureDepreciation,
		ItemDescription:         positional[0],
		DepreciationRatePercent: rate,
	}

	result, dispatchErr := app.submitValue(cmd)
	printOutcome(out, dispatchErr, result.Err, "depreciation rate for %q set to %d%%/yr", positional[0], rate)
}

func valueCurrent(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: value current <description> [--date YYYY-MM-DD]")
		return
	}

	date := flags["date"]
	if date == "" {
		date = today()
	}

	cmd := assess.Command{
		CorrelationID:   newID(),
		Action:          assess.ActionComputeCurrentValue,
		ItemDescription: positional[0],
		Date:            date,
	}

	result, dispatchErr := app.submitValue(cmd)
	if dispatchErr != nil {
		fmt.Fprintf(out, "error: %v\n", dispatchErr)
		return
	}
	if result.Err != nil {
		fmt.Fprintf(out, "error: %v\n", result.Err)
		return
	}
	if result.Value == nil {
		fmt.Fprintln(out, "error: no current value computed")
		return
	}
	fmt.Fprintf(out, "%q current value as of %s: %s\n", positional[0], date, formatCents(result.Value.AmountCents, result.Value.Currency))
}
