package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/esdatalabs/troventory/internal/locations/services/manage"
)

func handleLocation(app *App, out *os.File, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(out, "usage: location <add|rename|move|archive|list> ...")
		return
	}

	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		locationAdd(app, out, rest)
	case "rename":
		locationRename(app, out, rest)
	case "move":
		locationMove(app, out, rest)
	case "archive":
		locationArchive(app, out, rest)
	case "list":
		locationList(app, out)
	default:
		fmt.Fprintf(out, "unknown location subcommand %q\n", sub)
	}
}

func locationAdd(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: location add <name> [--parent NAME]")
		return
	}

	cmd := manage.Command{
		CorrelationID: newID(),
		Reference:     flags["ref"],
		Action:        manage.ActionCreate,
		Name:          positional[0],
	}
	if parent, ok := flags["parent"]; ok {
		cmd.ParentName = &parent
	}
	if cmd.Reference == "" {
		cmd.Reference = newID()
	}

	result, err := app.submitLocation(cmd)
	printOutcome(out, err, result.Err, "location %q created", positional[0])
}

func locationRename(app *App, out *os.File, args []string) {
	positional, _ := parseFlags(args)
	if len(positional) < 2 {
		fmt.Fprintln(out, "usage: location rename <old-name> <new-name>")
		return
	}

	cmd := manage.Command{
		CorrelationID: newID(),
		Action:        manage.ActionRename,
		TargetName:    positional[0],
		Name:          positional[1],
	}

	result, err := app.submitLocation(cmd)
	printOutcome(out, err, result.Err, "location %q renamed to %q", positional[0], positional[1])
}

func locationMove(app *App, out *os.File, args []string) {
	positional, flags := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: location move <name> [--parent NAME]")
		return
	}

	cmd := manage.Command{
		CorrelationID: newID(),
		Action:        manage.ActionMove,
		TargetName:    positional[0],
	}
	if parent, ok := flags["parent"]; ok {
		cmd.ParentName = &parent
	}

	result, err := app.submitLocation(cmd)
	printOutcome(out, err, result.Err, "location %q moved", positional[0])
}

func locationArchive(app *App, out *os.File, args []string) {
	positional, _ := parseFlags(args)
	if len(positional) < 1 {
		fmt.Fprintln(out, "usage: location archive <name>")
		return
	}

	cmd := manage.Command{
		CorrelationID: newID(),
		Action:        manage.ActionArchive,
		TargetName:    positional[0],
	}

	result, err := app.submitLocation(cmd)
	printOutcome(out, err, result.Err, "location %q archived", positional[0])
}

func locationList(app *App, out *os.File) {
	all := app.store.AllLocations()
	if len(all) == 0 {
		fmt.Fprintln(out, "(no locations)")
		return
	}

	children := make(map[string][]string)
	for _, loc := range all {
		children[loc.Parent] = append(children[loc.Parent], loc.Name)
	}
	for parent := range children {
		sort.Strings(children[parent])
	}

	byName := make(map[string]bool)
	for _, loc := range all {
		byName[loc.Name] = loc.Archived
	}

	var walk func(name string, depth int)
	walk = func(name string, depth int) {
		archived := ""
		if byName[name] {
			archived = " (archived)"
		}
		fmt.Fprintf(out, "%s- %s%s\n", indent(depth), name, archived)
		for _, child := range children[name] {
			walk(child, depth+1)
		}
	}
	for _, top := range children[""] {
		walk(top, 0)
	}
}

func indent(depth int) string {
	out := ""
	for i := 0; i < depth; i++ {
		out += "  "
	}
	return out
}

// printOutcome prints a one-line success or failure message. dispatchErr is
// non-nil only if the Dispatcher itself has been closed; resultErr is the
// business/infrastructure outcome reported via AuditGateway.
func printOutcome(out *os.File, dispatchErr, resultErr error, successFormat string, args ...any) {
	if dispatchErr != nil {
		fmt.Fprintf(out, "error: %v\n", dispatchErr)
		return
	}
	if resultErr != nil {
		fmt.Fprintf(out, "error: %v\n", resultErr)
		return
	}
	fmt.Fprintf(out, successFormat+"\n", args...)
}
