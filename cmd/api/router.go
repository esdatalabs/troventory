package main

import (
	"log/slog"
	"net/http"
	"time"
)

// newRouter registers every route this API exposes against app, wrapped
// with logging and CORS middleware so a locally-run Vue dashboard (a
// different origin/port during development) can call it directly.
func newRouter(app *App, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)

	mux.HandleFunc("POST /items", createItem(app))
	mux.HandleFunc("GET /items", listItems(app))
	mux.HandleFunc("GET /items/{description}", getItem(app))
	mux.HandleFunc("PATCH /items/{description}", updateItem(app))
	mux.HandleFunc("POST /items/{description}/archive", archiveItem(app))
	mux.HandleFunc("POST /items/scan", scanItem(app))
	mux.HandleFunc("POST /items/enrich", enrichItem(app))
	mux.HandleFunc("POST /items/{description}/value/price", recordPrice(app))
	mux.HandleFunc("POST /items/{description}/value/appraisals", recordAppraisal(app))
	mux.HandleFunc("PUT /items/{description}/value/depreciation-rate", setDepreciationRate(app))
	mux.HandleFunc("GET /items/{description}/value/current", getCurrentValue(app))

	mux.HandleFunc("POST /locations", createLocation(app))
	mux.HandleFunc("GET /locations", listLocations(app))
	mux.HandleFunc("GET /locations/{name}", getLocation(app))
	mux.HandleFunc("POST /locations/{name}/rename", renameLocation(app))
	mux.HandleFunc("POST /locations/{name}/move", moveLocation(app))
	mux.HandleFunc("POST /locations/{name}/archive", archiveLocation(app))

	mux.HandleFunc("GET /search", searchItems(app))
	mux.HandleFunc("GET /export", exportReport(app))

	return withLogging(log, withCORS(mux))
}

func healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// withCORS allows any origin to call this API. This is a locally-run demo
// server with no auth/cookies to protect, so a permissive policy trades
// away nothing — it just lets the Vue dashboard's dev server (a different
// origin/port) call it without a proxy.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withLogging logs one structured line per request: method, path, status,
// and duration.
func withLogging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start),
		)
	})
}

// statusWriter captures the status code written so withLogging can report
// it after the handler returns.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
