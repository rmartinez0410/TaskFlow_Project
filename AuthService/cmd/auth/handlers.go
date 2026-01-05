package main

import (
	"auth/internal/data"
	"auth/internal/validator"
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func (app *application) start() error {
	_, err := app.nc.QueueSubscribe("auth.*", "auth_workers", func(msg *nats.Msg) {
		subject := strings.Split(msg.Subject, ".")[1]

		switch subject {
		case "healthcheck":
			app.healthcheck(msg)
		case "register":
			app.registerHandler(msg)
		case "login":
			app.loginHandler(msg)
		case "validate":
			app.accessTokenHandler(msg)
		case "refresh":
			app.refreshTokenHandler(msg)
		case "logout":
			app.logOutHandler(msg)
		default:
			app.sendErrorResponse(msg, http.StatusUnprocessableEntity, "invalid subject")
		}
	})
	return err
}

func (app *application) healthcheck(msg *nats.Msg) {
	app.sendSuccessResponse(msg, http.StatusOK, "auth up and running")
}

func (app *application) registerHandler(msg *nats.Msg) {
	var input data.RegisterInput
	if !app.readJSON(msg, &input, func(v *validator.Validator) {
		data.ValidateRegisterInput(v, input)
	}) {
		return
	}

	user := &data.User{Email: input.Email, Username: input.Username}
	if err := user.Password.Set(input.Password); err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	if err := app.models.UserModel.Insert(user); err != nil {
		if errors.Is(err, data.ErrDuplicateEmail) {
			app.sendErrorResponse(msg, http.StatusConflict, "email is already in use")
			return
		}
		app.sendInternalServerErrorResponse(msg)
		return
	}
	app.sendSuccessResponse(msg, http.StatusCreated, "user successfully created")
}

func (app *application) loginHandler(msg *nats.Msg) {
	var input data.LoginInput
	if !app.readJSON(msg, &input, func(v *validator.Validator) {
		data.ValidateLoginInput(v, input)
	}) {
		return
	}

	user, err := app.models.UserModel.GetByEmail(input.Email)
	if err != nil && !errors.Is(err, data.ErrNoRecord) {
		app.sendInternalServerErrorResponse(msg)
		return
	}

	ok, err := func() (bool, error) {
		dummyHash := data.User{}
		_ = dummyHash.Password.Set(input.Password)
		if user != nil {
			return user.Password.Matches(input.Password)
		}
		return dummyHash.Password.Matches(input.Password)
	}()

	if err != nil || !ok || user == nil {
		app.sendErrorResponse(msg, http.StatusUnauthorized, "invalid credentials")
		return
	}

	sessionID := uuid.NewString()
	accessToken, err := app.generateAccessToken(user.ID, user.Email, user.Username, sessionID)
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	opaqueToken, err := app.generateOpaqueToken()
	if err != nil {
		app.logger.Error("error generating opaque token")
		app.sendInternalServerErrorResponse(msg)
		return
	}
	hash := sha256.Sum256([]byte(opaqueToken))

	session := &data.Session{
		SessionID:  sessionID,
		TokenHash:  hash[:],
		UserID:     user.ID,
		DeviceName: input.DeviceName,
		DeviceType: input.DeviceType,
		RememberMe: input.RememberMe,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		IPAddress:  nil,
		UserAgent:  input.UserAgent,
	}

	if input.IPAddress != "" {
		session.IPAddress = &input.IPAddress
	}
	if session.RememberMe {
		session.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	}

	if err := app.models.SessionModel.Insert(session); err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}

	otherSessions, err := app.models.GetOtherSessions(user.ID, session.SessionID)
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	if len(otherSessions) > 4 {
		err = app.models.SessionModel.Revoke(otherSessions[4].SessionID)
		if err != nil {
			app.sendInternalServerErrorResponse(msg)
			return
		}
	}
	app.sendSuccessResponse(msg, http.StatusOK, data.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: opaqueToken,
		CurrentSession: data.SessionResponse{
			SessionID:  session.SessionID,
			DeviceName: session.DeviceName,
			DeviceType: session.DeviceType,
			LastUsedAt: session.LastUsedAt,
		},
		OtherSessions: otherSessions,
	})
}

func (app *application) logOutHandler(msg *nats.Msg) {
	var input data.LogoutInput
	if !app.readJSON(msg, &input, func(v *validator.Validator) {
		data.ValidateLogoutInput(v, input)
	}) {
		return
	}

	err := app.models.SessionModel.Revoke(input.SessionID)
	if err != nil {
		if errors.Is(err, data.ErrNoRecord) {
			app.sendErrorResponse(msg, http.StatusNotFound, "session not found")
			return
		}
		app.sendInternalServerErrorResponse(msg)
		return
	}

	app.sendSuccessResponse(msg, http.StatusOK, "user successfully logged out")
}

func (app *application) accessTokenHandler(msg *nats.Msg) {
	var input data.AccessTokenInput
	if !app.readJSON(msg, &input, func(v *validator.Validator) {
		data.ValidateAccessTokenInput(v, input)
	}) {
		return
	}

	claims, err := app.validateAccessToken(input.TokenString)
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed), errors.Is(err, jwt.ErrTokenNotValidYet):
			app.sendErrorResponse(msg, http.StatusUnauthorized, "invalid token")
		case errors.Is(err, jwt.ErrTokenExpired):
			app.sendErrorResponse(msg, http.StatusUnauthorized, "token expired")
		default:
			app.sendInternalServerErrorResponse(msg)
		}
		return
	}

	session, err := app.models.SessionModel.GetByID(claims.SessionID)
	if err != nil {
		if errors.Is(err, data.ErrNoRecord) {
			app.sendErrorResponse(msg, http.StatusUnauthorized, "no session found")
		}
		app.sendInternalServerErrorResponse(msg)
		return
	}

	if session.RevokedAt != nil {
		app.sendErrorResponse(msg, http.StatusUnauthorized, "token expired")
		return
	}

	app.sendSuccessResponse(msg, http.StatusOK, data.TokenValidationResponse{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Username: claims.Username,
	})
}

func (app *application) refreshTokenHandler(msg *nats.Msg) {
	var input data.RefreshTokenInput
	if !app.readJSON(msg, &input, func(v *validator.Validator) {
		data.ValidateRefreshTokenInput(v, input)
	}) {
		return
	}

	hash := sha256.Sum256([]byte(input.TokenString))
	session, err := app.models.SessionModel.GetByTokenHash(hash[:])
	if err != nil {
		if errors.Is(err, data.ErrNoRecord) {
			app.sendErrorResponse(msg, http.StatusUnauthorized, "invalid token")
			return
		}
		app.sendInternalServerErrorResponse(msg)
		return
	}
	switch {
	case session.RevokedAt != nil:
		app.sendErrorResponse(msg, http.StatusUnauthorized, true)
		return
	case time.Now().After(session.ExpiresAt):
		app.sendErrorResponse(msg, http.StatusUnauthorized, false)
		return
	}
	if err := app.models.SessionModel.UpdateLastUsed(session.SessionID); err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	user, err := app.models.UserModel.GetByID(session.UserID)
	if err != nil {
		if errors.Is(err, data.ErrNoRecord) {
			app.sendErrorResponse(msg, http.StatusUnauthorized, "invalid token")
			return
		}
		app.sendInternalServerErrorResponse(msg)
		return
	}
	accessToken, err := app.generateAccessToken(user.ID, user.Email, user.Username, session.SessionID)
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	app.sendSuccessResponse(msg, http.StatusOK, data.TokenRefreshResponse{
		AccessToken: accessToken,
	})
}
