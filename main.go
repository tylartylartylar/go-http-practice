package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cfg.fileserverHits.Add(1)

		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Get the current count and format it as "Hits: x"
	// Write the formatted string to the response
	currentCount := cfg.fileserverHits.Load()
	currentCountString := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", currentCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(currentCountString))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	// Reset the counter to 0
	// Optionally send a confirmation message
	cfg.fileserverHits.Store(0)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	const port = "8080"
	ServeMux := http.NewServeMux()
	apiCfg := apiConfig{}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: ServeMux,
	}

	ServeMux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	ServeMux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	ServeMux.HandleFunc("GET /api/healthz", healthHandler)

	fileHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	ServeMux.Handle("/app/", apiCfg.middlewareMetricsInc(fileHandler))

	server.ListenAndServe()
}
