package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/esdatalabs/troventory/internal/items/services/catalog"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
)

func handleItem(app *App, out *os.File, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(out, "usage: item <add|update|archive|show|scan|enrich> ...")
		return
	}

	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		itemAdd(app, out, rest)
	case "update":
		itemUpdate(app, out, rest)
	case "archive":
		itemArchive(app, out, rest)
	case "show":
		itemShow(app, out, rest)
	case "scan":
		itemScan(app, out, rest)
	case "enrich":
		itemEnrich(app, out, rest)
	default:
		fmt.Fprintf(out, "unknown item subcommand %q\n", sub)
	}
}

func itemAdd(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, `usage: item add <description> --category CAT [--date YYYY-MM-DD] [--price DOLLARS] [--currency USD] [--vendor NAME] [--location NAME] [--photo FILE]`)
		return
	}
	if flags["category"] == "" {
		fmt.Fprintln(out, "error: --category is required")
		return
	}

	priceCents, err := priceFlag(flags)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return
	}

	cmd := catalog.Command{
		CorrelationID:      newID(),
		Reference:          flags["ref"],
		Action:             catalog.ActionCreate,
		Description:        positional[0],
		Category:           flags["category"],
		PurchaseDate:       flags["date"],
		PurchasePriceCents: priceCents,
		Currency:           currencyFlag(flags),
		Vendor:             flags["vendor"],
		Photos:             photosFlag(flags),
	}
	if loc, ok := flags["location"]; ok {
		cmd.LocationName = &loc
	}
	if cmd.Reference == "" {
		cmd.Reference = newID()
	}

	result, dispatchErr := app.submitCatalog(cmd)
	printOutcome(out, dispatchErr, result.Err, "item %q created", positional[0])
}

func itemUpdate(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, `usage: item update <description> [--description NEW] [--category CAT] [--date YYYY-MM-DD] [--price DOLLARS] [--currency USD] [--vendor NAME] [--location NAME] [--photo FILE]`)
		return
	}

	current, ok := app.store.Item(positional[0])
	if !ok {
		fmt.Fprintf(out, "error: item %q not found\n", positional[0])
		return
	}

	newDescription := current.Description
	if v, ok := flags["description"]; ok {
		newDescription = v
	}
	category := current.Category
	if v, ok := flags["category"]; ok {
		category = v
	}
	date := current.PurchaseDate
	if v, ok := flags["date"]; ok {
		date = v
	}
	priceCents := current.PurchasePriceCents
	if v, ok := flags["price"]; ok {
		cents, err := parseCents(v)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return
		}
		priceCents = cents
	}
	currency := current.Currency
	if v, ok := flags["currency"]; ok {
		currency = v
	}
	vendor := current.Vendor
	if v, ok := flags["vendor"]; ok {
		vendor = v
	}
	locationName := current.LocationName
	if v, ok := flags["location"]; ok {
		locationName = v
	}
	photos := current.Photos
	if _, ok := flags["photo"]; ok {
		photos = photosFlag(flags)
	}

	cmd := catalog.Command{
		CorrelationID:      newID(),
		Action:             catalog.ActionUpdate,
		TargetDescription:  positional[0],
		Description:        newDescription,
		Category:           category,
		PurchaseDate:       date,
		PurchasePriceCents: priceCents,
		Currency:           currency,
		Vendor:             vendor,
		Photos:             photos,
	}
	if locationName != "" {
		cmd.LocationName = &locationName
	}

	result, dispatchErr := app.submitCatalog(cmd)
	printOutcome(out, dispatchErr, result.Err, "item %q updated", positional[0])
}

func itemArchive(app *App, out *os.File, args []string) {
	positional, _ := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: item archive <description>")
		return
	}

	cmd := catalog.Command{
		CorrelationID:     newID(),
		Action:            catalog.ActionArchive,
		TargetDescription: positional[0],
	}

	result, dispatchErr := app.submitCatalog(cmd)
	printOutcome(out, dispatchErr, result.Err, "item %q archived", positional[0])
}

func itemShow(app *App, out *os.File, args []string) {
	positional, _ := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: item show <description>")
		return
	}

	item, ok := app.store.Item(positional[0])
	if !ok {
		fmt.Fprintf(out, "error: item %q not found\n", positional[0])
		return
	}

	fmt.Fprintf(out, "Description:   %s\n", item.Description)
	fmt.Fprintf(out, "Category:      %s\n", item.Category)
	fmt.Fprintf(out, "Purchase date: %s\n", item.PurchaseDate)
	fmt.Fprintf(out, "Purchase price:%s\n", formatCents(item.PurchasePriceCents, item.Currency))
	fmt.Fprintf(out, "Vendor:        %s\n", item.Vendor)
	fmt.Fprintf(out, "Location:      %s\n", item.LocationName)
	fmt.Fprintf(out, "Photos:        %s\n", strings.Join(item.Photos, ", "))
	fmt.Fprintf(out, "Archived:      %v\n", item.Archived)
	fmt.Fprintf(out, "Barcode:       %s\n", item.Barcode)

	if val, ok := app.store.Valuation(item.Description); ok {
		fmt.Fprintf(out, "Recorded purchase price: %s\n", formatCents(val.PurchasePriceCents, val.PurchaseCurrency))
		fmt.Fprintf(out, "Depreciation rate: %d%%/yr\n", val.DepreciationRatePercent)
		for _, a := range val.Appraisals {
			fmt.Fprintf(out, "Appraisal (%s): %s\n", a.AsOf, formatCents(a.AmountCents, a.Currency))
		}
	}
}

func itemScan(app *App, out *os.File, args []string) {
	positional, _ := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: item scan <barcode>")
		return
	}

	// There is no catalog/enrich Command that creates the very first draft
	// record — enrich only ever fills gaps on an item that already exists
	// (see enrich.Command's TargetDescription doc). In production this
	// hand-off point is where a physical barcode scanner would push a
	// bare, not-yet-described item into storage; the CLI stands in for
	// that here as driving-side setup, not a business operation.
	if err := app.store.SaveDraft(itemDraft(positional[0])); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return
	}
	fmt.Fprintf(out, "scanned draft item for barcode %q — run \"item enrich %s\" to fill it in\n", positional[0], positional[0])
}

func itemEnrich(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: item enrich <barcode> [--target DESCRIPTION]")
		return
	}

	cmd := enrich.Command{
		CorrelationID:     newID(),
		Reference:         flags["ref"],
		TargetDescription: flags["target"],
		Barcode:           positional[0],
	}
	if cmd.Reference == "" {
		cmd.Reference = newID()
	}

	result, dispatchErr := app.submitEnrich(cmd)
	printOutcome(out, dispatchErr, result.Err, "item enriched from barcode %q", positional[0])
}
