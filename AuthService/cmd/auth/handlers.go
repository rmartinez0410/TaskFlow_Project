package main

import (
	"auth/internal/data"
	"auth/internal/validator"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nats-io/nats.go"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) start() error {
	_, err := app.nc.QueueSubscribe("auth.*", "auth_workers", func(msg *nats.Msg) {
		subject := strings.Split(msg.Subject, ".")[1]

		switch subject {
		case "register":
			app.registerHandler(msg)
		case "login":
			app.loginHandler(msg)
		case "validate":
			app.validateTokenHandler(msg)
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

func (app *application) registerHandler(msg *nats.Msg) {
	var input data.RegisterInput
	if err := json.Unmarshal(msg.Data, &input); err != nil {
		app.sendUnprocessableEntityResponse(msg)
		return
	}
	user := &data.User{Email: input.Email, Username: input.Username}
	if err := user.Password.Set(&input.Password); err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	v := validator.New()
	if data.ValidateUser(user, v); !v.Valid() {
		app.sendErrorResponse(msg, http.StatusUnprocessableEntity, v.Errors)
		return
	}
	if err := app.models.UserModel.Insert(user); err != nil {
		if errors.Is(err, data.ErrDuplicateEmail) {
			v.AddError("email", "is already in use")
			app.sendErrorResponse(msg, http.StatusConflict, v.Errors)
			return
		}
		app.sendInternalServerErrorResponse(msg)
		return
	}
	app.sendSuccessResponse(msg, http.StatusCreated, "user successfully created")
}

func (app *application) loginHandler(msg *nats.Msg) {
	var input data.LoginInput
	if err := json.Unmarshal(msg.Data, &input); err != nil {
		app.sendUnprocessableEntityResponse(msg)
		return
	}

	v := validator.New()
	data.ValidateEmail(input.Email, v)
	data.ValidatePasswordPlainText(input.Password, v)
	if !v.Valid() {
		app.sendErrorResponse(msg, http.StatusBadRequest, v.Errors)
		return
	}

	user, err := app.models.UserModel.GetByEmail(input.Email)
	if err != nil && !errors.Is(err, data.ErrNoRecord) {
		app.sendInternalServerErrorResponse(msg)
		return
	}

	ok, err := func() (bool, error) {
		dummyHash := data.User{}
		_ = dummyHash.Password.Set(&input.Password)
		if user != nil {
			return user.Password.Matches(&input.Password)
		}
		return dummyHash.Password.Matches(&input.Password)
	}()

	if err != nil || !ok || user == nil {
		app.sendErrorResponse(msg, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := app.generateAccessToken(user)
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	opaqueToken, err := app.generateOpaqueToken()
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(opaqueToken), 12)
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}

	session := &data.Session{
		TokenHash:  hash,
		UserID:     user.ID,
		DeviceName: input.DeviceName,
		DeviceType: input.DeviceType,
		RememberMe: input.RememberMe,
		IPAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
	}
	if err := app.models.SessionModel.Insert(session, 5); err != nil {
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
	err := json.Unmarshal(msg.Data, &input)
	if err != nil {
		app.sendUnprocessableEntityResponse(msg)
		return
	}

	err = app.models.SessionModel.Revoke(input.SessionID)
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

func (app *application) validateTokenHandler(msg *nats.Msg) {
	var input data.ValidateTokenInput
	if err := json.Unmarshal(msg.Data, &input); err != nil {
		app.sendUnprocessableEntityResponse(msg)
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
	app.sendSuccessResponse(msg, http.StatusOK, data.TokenValidationResponse{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Username: claims.Username,
	})
}

func (app *application) refreshTokenHandler(msg *nats.Msg) {
	var input data.RefreshTokenInput
	if err := json.Unmarshal(msg.Data, &input); err != nil {
		app.sendUnprocessableEntityResponse(msg)
		return
	}

	session, err := app.models.SessionModel.GetByID(input.TokenString)
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
	accessToken, err := app.generateAccessToken(user)
	if err != nil {
		app.sendInternalServerErrorResponse(msg)
		return
	}
	app.sendSuccessResponse(msg, http.StatusOK, data.TokenRefreshResponse{
		AccessToken: accessToken,
	})
}
