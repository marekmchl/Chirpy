package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeValidateJWT(t *testing.T) {
	cases := []struct {
		UserID      uuid.UUID
		TokenSecret string
		ExpiresIn   time.Duration
	}{
		{
			UserID:      uuid.New(),
			TokenSecret: "superSecret",
			ExpiresIn:   time.Duration(60 * time.Second),
		},
		{
			UserID:      uuid.New(),
			TokenSecret: "evenMoreSecret",
			ExpiresIn:   time.Duration(60 * time.Second),
		},
		{
			UserID:      uuid.New(),
			TokenSecret: "superDuperSecret",
			ExpiresIn:   time.Duration(60 * time.Second),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case: %v", i), func(t *testing.T) {
			jwt, err := MakeJWT(c.UserID, c.TokenSecret, c.ExpiresIn)
			if err != nil {
				t.Errorf("MakeJWT failed with: %v", err)
				return
			}

			id, err := ValidateJWT(jwt, c.TokenSecret)
			if err != nil {
				t.Errorf("ValidateJWT failed with: %v", err)
				return
			}
			if id != c.UserID {
				t.Errorf("IDs don't match: %v != %v", id, c.UserID)
				return
			}
		})
	}
}

func TestMakeValidateJWTExpired(t *testing.T) {
	const sleepTime = time.Duration(5 * time.Second)
	cases := []struct {
		UserID      uuid.UUID
		TokenSecret string
		ExpiresIn   time.Duration
	}{
		{
			UserID:      uuid.New(),
			TokenSecret: "secret",
			ExpiresIn:   time.Duration(1 * time.Second),
		},
		{
			UserID:      uuid.New(),
			TokenSecret: "secreter",
			ExpiresIn:   time.Duration(2 * time.Second),
		},
		{
			UserID:      uuid.New(),
			TokenSecret: "secreterer",
			ExpiresIn:   time.Duration(3 * time.Second),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case: %v", i), func(t *testing.T) {
			jwt, err := MakeJWT(c.UserID, c.TokenSecret, c.ExpiresIn)
			if err != nil {
				t.Errorf("MakeJWT failed with: %v", err)
				return
			}

			time.Sleep(sleepTime)

			_, err = ValidateJWT(jwt, c.TokenSecret)
			if err == nil {
				t.Errorf("ValidateJWT succeeded after it should have been expired")
				return
			}
		})
	}
}
