package main

import (
	"auth/AuthService/internal/data"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessToken struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Type     string `json:"type"`
	jwt.RegisteredClaims
}

func (app *application) generateAccessToken(user *data.User) (string, error) {
	claims := &AccessToken{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Type:     "access_token",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
			Subject:   user.ID,
			Audience:  []string{"task-flow"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	return token.SignedString(app.jwtAccessSecret)
}

func (app *application) validateAccessToken(tokenString string) (*AccessToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessToken{}, func(token *jwt.Token) (any, error) {
		return app.jwtAccessSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AccessToken); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (app *application) generateOpaqueToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
