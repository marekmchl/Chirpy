package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marekmchl/Chirpy/internal/auth"
	"github.com/marekmchl/Chirpy/internal/database"
)

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
	if cfg.platform != "dev" {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		w.Write([]byte("Forbidden"))
		return
	}
	cfg.fileserverHits.Store(0)
	if err := cfg.db.DeleteAllUsers(r.Context()); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
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

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
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

	// parse chirp
	type chirp struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

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

	// user authorization
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		resBody, err := json.Marshal(
			returnError{
				Error: "Internal server error",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	tokenID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(401)
		resBody, err := json.Marshal(
			returnError{
				Error: "Unauthorized",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	// is valid -> create chirp
	dbChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   replaceProfanities(oneChirp.Body),
		UserID: tokenID,
	})
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(500)
		resBody, err := json.Marshal(
			returnError{
				Error: "Internal server error",
			},
		)
		if err != nil {
			resBody = []byte{}
		}
		w.Write(resBody)
		return
	}

	type resChirpStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}
	respVals := resChirpStruct{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	resBody, err := json.Marshal(respVals)
	if err != nil {
		resBody = []byte{}
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(resBody)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type emailStruct struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	reqData := emailStruct{}
	if err := decoder.Decode(&reqData); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	hashedPassword, err := auth.HashPassword(reqData.Password)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}
	rawUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		HashedPassword: hashedPassword,
		Email:          reqData.Email,
	})
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

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	type chirpStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	dbChirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}

	chirpStructs := []chirpStruct{}
	for _, dbChirp := range dbChirps {
		chirpStructs = append(chirpStructs, chirpStruct{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.ID,
		})
	}

	chirpsJson, err := json.Marshal(chirpStructs)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(chirpsJson)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	type chirpStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	reqID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}
	dbChirp, err := cfg.db.GetChirpByID(r.Context(), reqID)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte("Chirp not found"))
		return
	}

	respChirp := chirpStruct{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.ID,
	}

	chirpJson, err := json.Marshal(respChirp)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte("Chirp not Found"))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(chirpJson)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type loginStruct struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	reqData := loginStruct{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&reqData); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	userDB, err := cfg.db.GetUserByEmail(r.Context(), reqData.Email)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}

	if err := auth.CheckPasswordHash(userDB.HashedPassword, reqData.Password); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}

	token, err := auth.MakeJWT(userDB.ID, cfg.secret, time.Duration(10*time.Minute))
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}

	refreshTokenString, err := auth.MakeRefreshToken()
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal server error - failed to make refresh token"))
		return
	}
	cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshTokenString,
		UserID:    userDB.ID,
		ExpiresAt: time.Now().Add(time.Duration(1 * time.Hour)),
	})

	type userStruct struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}
	user := userStruct{
		ID:           userDB.ID,
		CreatedAt:    userDB.CreatedAt,
		UpdatedAt:    userDB.UpdatedAt,
		Email:        userDB.Email,
		Token:        token,
		RefreshToken: refreshTokenString,
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(userJson)
	return
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}

	refreshToken, err := cfg.db.GetRefreshTokenByToken(r.Context(), refreshTokenString)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized Request"))
		return
	}
	if refreshToken.RevokedAt.Valid {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized Request"))
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken.Token)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.secret, time.Duration(1*time.Hour))
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}

	type returnTokenType struct {
		Token string `json:"token"`
	}
	returnToken := returnTokenType{Token: accessToken}

	tokenJson, err := json.Marshal(returnToken)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(tokenJson)
	return
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte{}, "Internal Server Error - %v", err))
		return
	}

	if err := cfg.db.RevokeRefreshToken(r.Context(), refreshTokenString); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte{}, "Internal Server Error - %v", err))
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	reqJWT, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte{}, "Failed reading token from header: %v", err))
		return
	}
	reqUserID, err := auth.ValidateJWT(reqJWT, cfg.secret)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte{}, "Failed reading token from header: %v", err))
		return
	}

	type passAndEmail struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	reqData := passAndEmail{}
	if err := decoder.Decode(&reqData); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte{}, "Failed decoding data: %v", err))
		return
	}

	hashedPassword, err := auth.HashPassword(reqData.Password)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte{}, "Failed hashing the password: %v", err))
		return
	}

	dbUser, err := cfg.db.UpdateUserWithID(r.Context(), database.UpdateUserWithIDParams{
		ID:             reqUserID,
		HashedPassword: hashedPassword,
		Email:          reqData.Email,
	})
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte{}, "Failed updating the user credentials: %v", err))
		return
	}

	type userInfo struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	user := userInfo{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte{}, "Failed marshalling the response body: %v", err))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(userJson)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	reqJWT, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte{}, "Failed getting token: %v", err))
		return
	}

	reqUserID, err := auth.ValidateJWT(reqJWT, cfg.secret)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		w.Write(fmt.Appendf([]byte{}, "Token invalid"))
		return
	}

	reqID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		w.Write(fmt.Appendf([]byte{}, "Failed parsing the ID: %v", err))
		return
	}

	chirpDB, err := cfg.db.GetChirpByID(r.Context(), reqID)
	if err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		w.Write(fmt.Appendf([]byte{}, "Chirp with ID %v not found", reqID))
		return
	}

	if chirpDB.UserID != reqUserID {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		w.Write(fmt.Appendf([]byte{}, "Unauthorized request"))
		return
	}

	if err := cfg.db.DeleteChirpByID(r.Context(), reqID); err != nil {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		w.Write(fmt.Appendf([]byte{}, "Chirp with ID %v not found", reqID))
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(204)
	w.Write(fmt.Appendf([]byte{}, "Chirp deleted"))
}
