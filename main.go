package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func (cfg *apiConfig) validate_chirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	if len(params.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)

		type errorResponse struct {
			Error string `json:"error"`
		}
		resp := errorResponse{
			Error: "Chirp is too long",
		}
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}

		w.Write(jsonResp)
		return
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)

		type successResponse struct {
			Valid bool `json:"valid"`
		}
		resp := successResponse{
			Valid: true,
		}
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Write(jsonResp)
	}
}

func main() {
	const port = "8080"
	ServeMux := http.NewServeMux()
	apiCfg := apiConfig{}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: ServeMux,
	}
	ServeMux.HandleFunc("POST /api/validate_chirp", apiCfg.validate_chirp)
	ServeMux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	ServeMux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	ServeMux.HandleFunc("GET /api/healthz", healthHandler)

	fileHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	ServeMux.Handle("/app/", apiCfg.middlewareMetricsInc(fileHandler))

	server.ListenAndServe()
}
