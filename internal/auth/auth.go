package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashB, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hashB), nil
}

func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(expiresIn)},
			Subject:   userID.String(),
		},
	)
	signed, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("token signing failed with: %v", err)
	}
	return signed, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	type MyCustomClaims struct {
		Issuer    string           `json:"issuer"`
		IssuedAt  *jwt.NumericDate `json:"issued_at"`
		ExpiresAt *jwt.NumericDate `json:"expires_at"`
		Subject   string           `json:"subject"`
		jwt.RegisteredClaims
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&MyCustomClaims{},
		jwt.Keyfunc(
			func(token *jwt.Token) (any, error) {
				return []byte(tokenSecret), nil
			},
		),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("token parsing failed with: %v", err)
	}

	claims, ok := token.Claims.(*MyCustomClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("claims casting failed")
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return uuid.Nil, fmt.Errorf("subject get failed with: %v", err)
	}

	id, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("subject parsing failed with: %v", err)
	}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authString := headers.Get("Authorization")

	if len(authString) <= len("Bearer ") {
		return "", fmt.Errorf("Authorization not found")
	}
	tokenString := strings.TrimSpace(authString[len("Bearer "):])
	return tokenString, nil
}

func MakeRefreshToken() (string, error) {
	refreshTokenBytes := make([]byte, 256)

	_, err := rand.Read(refreshTokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to make a refresh token: %v", err)
	}

	return hex.EncodeToString(refreshTokenBytes), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authString := headers.Get("Authorization")

	authSlice := strings.Split(authString, " ")

	if strings.TrimSpace(authSlice[0]) != "ApiKey" {
		return "", fmt.Errorf("'%v' is not 'ApiKey'", strings.TrimSpace(authSlice[0]))
	}

	return strings.TrimSpace(authSlice[1]), nil
}
