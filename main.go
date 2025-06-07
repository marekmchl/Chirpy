package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/marekmchl/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func getConfig() *apiConfig {
	godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	pltfrm := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed - %v", err)
	}
	dbQueries := database.New(db)
	cfg := &apiConfig{
		db:       dbQueries,
		platform: pltfrm,
	}
	cfg.fileserverHits.Store(0)
	return cfg
}

func main() {
	cfg := getConfig()

	serveMux := http.ServeMux{}
	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte{'O', 'K'})
	})
	serveMux.HandleFunc("GET /admin/metrics", cfg.handlerGetMetrics)
	serveMux.HandleFunc("POST /admin/reset", cfg.handlerResetMetrics)
	serveMux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	serveMux.HandleFunc("POST /api/chirps", cfg.handlerCreateChirp)
	serveMux.HandleFunc("GET /api/chirps", cfg.handlerGetAllChirps)

	server := http.Server{
		Addr:    ":8080",
		Handler: &serveMux,
	}
	server.ListenAndServe()
}
