package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"goServer/internal/database"
)

type apiConfig struct {
    fileserverHits atomic.Int32
    DB *database.Queries
}

var profaneWords = []string{
	"kerfuffle",
	"sharbert",
	"fornax",
}

// filterProfanity replaces profane words with **** while preserving case
func filterProfanity(text string) string {
	words := strings.Split(text, " ")
	for i, word := range words {
		wordLower := strings.ToLower(word)
		for _, profane := range profaneWords {
			if wordLower == strings.ToLower(profane) {
				words[i] = "****"
				break
			}
		}
	}
	return strings.Join(words, " ")
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	currentCount := cfg.fileserverHits.Load()
	currentCountString := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", currentCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(currentCountString))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	resp := struct {
		Error string `json:"error"`
	}{
		Error: msg,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling error response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonResp)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	jsonResp, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonResp)
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
		respondWithError(w, http.StatusInternalServerError, "Error decoding request body")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	// Filter profanity
	cleaned := filterProfanity(params.Body)

	// Return the cleaned body
	resp := struct {
		CleanedBody string `json:"cleaned_body"`
	}{
		CleanedBody: cleaned,
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %s", err)
	}

    dbURL := os.Getenv("DB_URL")
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal(err)
    }

    defer db.Close()

	err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    dbQueries := database.New(db)

	const port = "8080"
	ServeMux := http.NewServeMux()
	apiCfg := apiConfig{
		DB: dbQueries,
	}
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
