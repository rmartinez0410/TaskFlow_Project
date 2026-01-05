package main

import (
	"auth/internal/data"
	"auth/internal/testutils"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRegisterHandler(t *testing.T) {
	testutils.ResetTestDB(t, dsn)

	tests := []Test{
		{
			name:    "success - valid register",
			payload: []byte(`{ "email":"test@mail.com", "password":"password123", "username":"tester"}`),
			want:    http.StatusCreated,
		},
		{
			name:    "fail - email already in use",
			payload: []byte(`{ "email":"test@mail.com", "password":"password123", "username":"tester"}`),
			want:    http.StatusConflict,
		},
		{
			name:    "fail - missing email",
			payload: []byte(`{ "email":"", "password":"password123", "username":"tester"}`),
			want:    http.StatusUnprocessableEntity,
		},
		{
			name:    "fail - invalid email format",
			payload: []byte(`{ "email":"not-an-email", "password":"password123", "username":"tester"}`),
			want:    http.StatusUnprocessableEntity,
		},
		{
			name:    "fail - password too short",
			payload: []byte(`{ "email":"valid@mail.com", "password":"123", "username":"tester"}`),
			want:    http.StatusUnprocessableEntity,
		},
		{
			name:    "fail - missing username",
			payload: []byte(`{ "email":"valid2@mail.com", "password":"password123", "username":""}`),
			want:    http.StatusUnprocessableEntity,
		},
		malformedJSON,
		emptyJSON,
	}

	runTests(t, "auth.register", tests)
}

func TestLoginHandler(t *testing.T) {
	testutils.ResetTestDB(t, dsn)

	_ = createTestUser(t)

	tests := []Test{
		{
			name: "success - valid login",
			payload: []byte(`{
                "email":"test@mail.com",
                "password":"12345678",
                "device_name":"laptop",
                "device_type":"desktop",
                "remember_me":false,
                "ip_address":"127.0.0.1",
                "user_agent":"go-test-client"
            }`),
			want: http.StatusOK,
		},
		{
			name: "fail - incorrect password",
			payload: []byte(`{
                "email":"test@mail.com",
                "password":"wrong password",
                "device_name":"laptop",
                "user_agent":"go-test-client"
            }`),
			want: http.StatusUnauthorized,
		},
		{
			name: "fail - non-existent user",
			payload: []byte(`{
                "email":"nonexistent@mail.com",
                "password":"12345678",
                "device_name":"laptop",
                "user_agent":"go-test-client"
            }`),
			want: http.StatusUnauthorized,
		},
		{
			name: "fail - invalid email format",
			payload: []byte(`{
                "email":"not-an-email",
                "password":"12345678",
                "device_name":"laptop"
            }`),
			want: http.StatusUnprocessableEntity,
		},
		malformedJSON,
		emptyJSON,
	}

	runTests(t, "auth.login", tests)

}

func TestSessionPruning(t *testing.T) {
	testutils.ResetTestDB(t, dsn)

	var oldestID string
	user := createTestUser(t)
	for i := range 5 {
		hash := sha256.Sum256([]byte(generateOpaqueTokenForTest(t)))
		s := &data.Session{
			SessionID:  uuid.NewString(),
			UserID:     user.ID,
			TokenHash:  hash[:],
			CreatedAt:  time.Now().Add(time.Duration(-i) * time.Hour),
			LastUsedAt: time.Now().Add(time.Duration(-i) * time.Hour),
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		err := app.models.SessionModel.Insert(s)
		if err != nil {
			t.Fatalf("failed to insert setup session %d: %v", i, err)
		}
		if i == 4 {
			oldestID = s.SessionID
		}
	}

	newHash := sha256.Sum256([]byte(generateOpaqueTokenForTest(t)))
	sixthSession := &data.Session{
		SessionID: uuid.NewString(),
		UserID:    user.ID,
		TokenHash: newHash[:],
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := app.models.SessionModel.Insert(sixthSession)
	if err != nil {
		t.Fatalf("failed to insert 6th session: %v", err)
	}

	var revokedAt *time.Time
	query := "SELECT revoked_at FROM sessions WHERE session_id = $1"
	err = app.models.SessionModel.DB.QueryRow(query, oldestID).Scan(&revokedAt)
	if err != nil {
		t.Fatalf("failed to query database for revoked status: %v", err)
	}

	if revokedAt == nil {
		t.Errorf("expected oldest session %s to be revoked, but it is still active", oldestID)
	}
}

func TestLogoutHandler(t *testing.T) {
	testutils.ResetTestDB(t, dsn)

	user := createTestUser(t)
	hash := sha256.Sum256([]byte(generateOpaqueTokenForTest(t)))
	session := createTestSession(t, user.ID, hash[:], time.Now().Add(24*time.Hour))
	hash = sha256.Sum256([]byte(generateOpaqueTokenForTest(t)))
	expiredSession := createTestSession(t, user.ID, hash[:], time.Now().Add(-1*time.Hour))

	tests := []Test{
		{
			name:    "success - valid session logout",
			payload: []byte(fmt.Sprintf(`{"session_id": "%s"}`, session.SessionID)),
			want:    http.StatusOK,
		},
		{
			name:    "fail - expired session",
			payload: []byte(fmt.Sprintf(`{"session_id": "%s"}`, expiredSession.SessionID)),
			want:    http.StatusNotFound,
		},
		malformedJSON,
		emptyJSON,
	}

	runTests(t, "auth.logout", tests)
}

func TestValidateTokenHandler(t *testing.T) {
	testutils.ResetTestDB(t, dsn)

	user := createTestUser(t)
	hash := sha256.Sum256([]byte(generateOpaqueTokenForTest(t)))
	validSession := createTestSession(t, user.ID, hash[:], time.Now().Add(24*time.Hour))
	hash = sha256.Sum256([]byte(generateOpaqueTokenForTest(t)))
	expiredSession := createTestSession(t, user.ID, hash[:], time.Now().Add(-1*time.Hour))

	validToken, err := app.generateAccessToken(user.ID, user.Email, user.Username, validSession.SessionID)
	if err != nil {
		t.Fatalf("failed to generate valid access token: %v", err)
	}

	expiredToken, err := generateExpiredTokenForTest(user.ID, user.Email, user.Username, validSession.SessionID)
	if err != nil {
		t.Fatalf("failed to generate expired access token: %v", err)
	}

	sessionExpiredToken, err := app.generateAccessToken(user.ID, user.Email, user.Username, expiredSession.SessionID)
	if err != nil {
		t.Fatalf("failed to generate session expired access token: %v", err)
	}

	tests := []Test{
		{
			name:    "success - valid token",
			payload: []byte(fmt.Sprintf(`{"access_token": "%s"}`, validToken)),
			want:    http.StatusOK,
		},
		{
			name:    "fail - expired token",
			payload: []byte(fmt.Sprintf(`{"access_token": "%s"}`, expiredToken)),
			want:    http.StatusUnauthorized,
		},
		{
			name:    "fail - invalid token",
			payload: []byte(`{"access_token": "not a valid token"}`),
			want:    http.StatusUnauthorized,
		},
		{
			name:    "fail - expired session",
			payload: []byte(fmt.Sprintf(`{"access_token": "%s"}`, sessionExpiredToken)),
			want:    http.StatusUnauthorized,
		},
		malformedJSON,
		emptyJSON,
	}

	runTests(t, "auth.validate", tests)
}

func TestRefreshTokenHandler(t *testing.T) {
	testutils.ResetTestDB(t, dsn)

	user := createTestUser(t)
	token := generateOpaqueTokenForTest(t)
	hash := sha256.Sum256([]byte(token))
	_ = createTestSession(t, user.ID, hash[:], time.Now().Add(24*time.Hour))

	sessionExpiredToken := generateOpaqueTokenForTest(t)
	hash = sha256.Sum256([]byte(sessionExpiredToken))
	_ = createTestSession(t, user.ID, hash[:], time.Now().Add(-1*time.Hour))

	tests := []Test{
		{
			name:    "success - valid refresh token",
			payload: []byte(fmt.Sprintf(`{"refresh_token": "%s"}`, token)),
			want:    http.StatusOK,
		},
		{
			name:    "fail - session expired",
			payload: []byte(fmt.Sprintf(`{"refresh_token": "%s"}`, sessionExpiredToken)),
			want:    http.StatusUnauthorized,
		},
		{
			name:    "fail - invalid token",
			payload: []byte(fmt.Sprintf(`{"refresh_token": "not a valid token"}`)),
			want:    http.StatusUnauthorized,
		},
		{
			name:    "fail - invalid json",
			payload: []byte(fmt.Sprintf(`{}`)),
			want:    http.StatusUnprocessableEntity,
		},
		malformedJSON,
		emptyJSON,
	}

	runTests(t, "auth.refresh", tests)
}
