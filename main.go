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
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/marekmchl/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, r *http.Request) {
	messageHTML := `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`
	message := fmt.Sprintf(messageHTML, cfg.fileserverHits.Load())
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(message))
}

func (cfg *apiConfig) handlerResetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	if err := cfg.db.DeleteAllUsers(r.Context()); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	}
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func replaceProfanities(s string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	chirpWords := strings.Split(s, " ")
	for i, chirpWord := range chirpWords {
		for _, profaneWord := range profaneWords {
			if strings.ToLower(chirpWord) == profaneWord {
				chirpWords[i] = "****"
			}
		}
	}
	return strings.Join(chirpWords, " ")
}

func (cfg *apiConfig) handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type returnError struct {
		Error string `json:"error"`
	}

	// if !isJson {}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(400)
		resBody, err := json.Marshal(
			returnError{
				Error: "Something went wrong",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	// read Chirp
	type chirp struct {
		Body string `json:"body"`
	}
	reqData := []byte{}
	if _, err := r.Body.Read(reqData); err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(400)
		resBody, err := json.Marshal(
			returnError{
				Error: "Something went wrong",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	// parse chirp
	oneChirp := &chirp{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(oneChirp); err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(400)
		resBody, err := json.Marshal(
			returnError{
				Error: "Something went wrong",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	// validate length
	if len(oneChirp.Body) > 140 {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(400)
		resBody, err := json.Marshal(
			returnError{
				Error: "Chirp is too long",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	// is valid
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)

	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
	}
	respVals := returnVals{
		CleanedBody: replaceProfanities(oneChirp.Body),
	}
	resBody, err := json.Marshal(respVals)
	if err != nil {
		resBody = []byte{}
	}
	w.Write(resBody)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type emailStruct struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	reqData := emailStruct{}
	if err := decoder.Decode(&reqData); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawUser, err := cfg.db.CreateUser(r.Context(), reqData.Email)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}

	type userStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	user := userStruct{
		ID:        rawUser.ID,
		CreatedAt: rawUser.CreatedAt,
		UpdatedAt: rawUser.UpdatedAt,
		Email:     rawUser.Email,
	}
	userJson, err := json.Marshal(user)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(userJson)
	return
}

func main() {
	godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed - %v", err)
	}
	dbQueries := database.New(db)
	cfg := apiConfig{
		db: dbQueries,
	}

	serveMux := http.ServeMux{}
	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte{'O', 'K'})
	})
	serveMux.HandleFunc("GET /admin/metrics", cfg.handlerGetMetrics)
	serveMux.HandleFunc("POST /admin/reset", cfg.handlerResetMetrics)
	serveMux.HandleFunc("POST /api/validate_chirp", cfg.handlerValidateChirp)
	serveMux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	server := http.Server{
		Addr:    ":8080",
		Handler: &serveMux,
	}
	server.ListenAndServe()
}
