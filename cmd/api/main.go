package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultDataPath := filepath.Join(home, ".troventory", "data.json")
	defaultLogPath := filepath.Join(home, ".troventory", "api.log")
	defaultProductsPath := filepath.Join(home, ".troventory", "products.json")

	dataPath := flag.String("data", defaultDataPath, "path to the JSON data file")
	logPath := flag.String("log", defaultLogPath, "path to the log file")
	productsPath := flag.String("products", defaultProductsPath, "path to a JSON barcode/product catalog to extend the built-in sample data")
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	logger, closeLog, err := openLogger(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "troventory-api: %v\n", err)
		os.Exit(1)
	}
	defer closeLog()

	app, err := NewApp(*dataPath, *productsPath, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "troventory-api: %v\n", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: newRouter(app, logger),
	}

	if err := run(server, app, logger, *addr); err != nil {
		fmt.Fprintf(os.Stderr, "troventory-api: %v\n", err)
		os.Exit(1)
	}
}

// run starts server, blocks until an interrupt/terminate signal arrives,
// then shuts down in the order ARCHITECTURE.md §4 rule 5 requires: stop
// accepting new HTTP requests, then drain app's Dispatchers, then return.
func run(server *http.Server, app *App, log *slog.Logger, addr string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		fmt.Printf("Troventory API listening on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down http server: %w", err)
	}
	app.Close()
	return <-serveErr
}

// openLogger opens a slog.Logger writing structured logs to path (created
// if needed), so request/audit logging stays out of the terminal.
func openLogger(path string) (*slog.Logger, func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logger, func() { _ = f.Close() }, nil
}
