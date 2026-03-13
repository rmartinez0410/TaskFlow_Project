package main

import (
	"auth/internal/data"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Test struct {
	name    string
	payload []byte
	want    int
}

var emptyJSON = Test{
	name:    "fail - empty json",
	payload: []byte(`{}`),
	want:    http.StatusUnprocessableEntity,
}

var malformedJSON = Test{
	name:    "fail - malformed json",
	payload: []byte(`not json`),
	want:    http.StatusUnprocessableEntity,
}

func runTests(t *testing.T, subj string, tests []Test) {
	t.Helper()

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			msg, err := app.nc.Request(subj, ts.payload, 2*time.Second)
			if err != nil {
				t.Fatalf("failed to get response from %s: %v", subj, err)
			}

			var r data.Response
			err = json.Unmarshal(msg.Data, &r)
			if err != nil {
				t.Fatalf("failed to unmarshal %s response: %v", subj, err)
			}

			if r.StatusCode != ts.want {
				t.Errorf("got %d want %d, %v", r.StatusCode, ts.want, r.Data)
			}
		})
	}
}

func createTestUser(t *testing.T) *data.User {
	t.Helper()

	user := &data.User{
		Email:    "test@mail.com",
		Username: "tester",
	}
	err := user.Password.Set("12345678")
	if err != nil {
		t.Fatalf("failed to set user password: %v", err)
	}
	err = app.models.UserModel.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user in db: %v", err)
	}

	return user
}

func createTestSession(t *testing.T, userID string, hash []byte, expiresAt time.Time) *data.Session {
	t.Helper()

	session := &data.Session{
		SessionID: uuid.NewString(),
		UserID:    userID,
		TokenHash: hash[:],
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: expiresAt,
	}
	err := app.models.SessionModel.Insert(session)
	if err != nil {
		t.Fatalf("failed to insert session in db:, %v", err)
	}
	return session
}

func generateExpiredTokenForTest(userID string, email string, username string, sessionID string) (string, error) {
	claims := &AccessToken{
		UserID:    userID,
		Email:     email,
		Username:  username,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Issuer:    "auth-service",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(app.jwtAccessSecret))
}

func generateOpaqueTokenForTest(t *testing.T) string {
	t.Helper()
	token, err := app.generateOpaqueToken()
	if err != nil {
		t.Fatalf("failed to generate opaque token: %v", err)
	}
	return token
}
