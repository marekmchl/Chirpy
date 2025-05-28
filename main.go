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

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(message))
}

func (cfg *apiConfig) handlerResetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	cfg := apiConfig{}

	serveMux := http.ServeMux{}
	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte{'O', 'K'})
	})
	serveMux.HandleFunc("/metrics", cfg.handlerGetMetrics)
	serveMux.HandleFunc("/reset", cfg.handlerResetMetrics)
	server := http.Server{
		Addr:    ":8080",
		Handler: &serveMux,
	}
	server.ListenAndServe()
}
