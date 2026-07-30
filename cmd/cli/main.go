package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultDataPath := filepath.Join(home, ".troventory", "data.json")
	defaultLogPath := filepath.Join(home, ".troventory", "cli.log")
	defaultProductsPath := filepath.Join(home, ".troventory", "products.json")

	dataPath := flag.String("data", defaultDataPath, "path to the JSON data file")
	logPath := flag.String("log", defaultLogPath, "path to the log file")
	productsPath := flag.String("products", defaultProductsPath, "path to a JSON barcode/product catalog to extend the built-in sample data")
	flag.Parse()

	logger, closeLog, err := openLogger(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "troventory: %v\n", err)
		os.Exit(1)
	}
	defer closeLog()

	app, err := NewApp(*dataPath, *productsPath, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "troventory: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	fmt.Printf("Troventory — data file: %s\n", *dataPath)
	fmt.Println(`Type "help" for a list of commands, "exit" to quit.`)

	runREPL(app, os.Stdin, os.Stdout)
}

// openLogger opens a slog.Logger writing structured logs to path (created
// if needed), so the CLI's own terminal stays uncluttered by per-command
// debug logging.
func openLogger(path string) (*slog.Logger, func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logger, func() { f.Close() }, nil
}

// runREPL reads one command per line from in, dispatching each to the
// matching handler, until "exit"/"quit" or EOF.
func runREPL(app *App, in *os.File, out *os.File) {
	scanner := bufio.NewScanner(in)
	fmt.Fprint(out, "> ")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Fprint(out, "> ")
			continue
		}

		tokens, err := tokenize(line)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			fmt.Fprint(out, "> ")
			continue
		}

		cmd, args := tokens[0], tokens[1:]
		if cmd == "exit" || cmd == "quit" {
			break
		}

		dispatchCommand(app, out, cmd, args)
		fmt.Fprint(out, "> ")
	}
	fmt.Fprintln(out)
}

// dispatchCommand routes cmd/args to the right feature's command handler.
func dispatchCommand(app *App, out *os.File, cmd string, args []string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(out, "error: %v\n", r)
		}
	}()

	switch cmd {
	case "help", "?":
		printHelp(out)
	case "item":
		handleItem(app, out, args)
	case "location":
		handleLocation(app, out, args)
	case "value":
		handleValue(app, out, args)
	case "search":
		handleSearch(app, out, args)
	case "export":
		handleExport(app, out, args)
	default:
		fmt.Fprintf(out, "unknown command %q — type \"help\" for a list of commands\n", cmd)
	}
}
