package data

import "time"

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}
type LoginInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	RememberMe bool   `json:"remember_me"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
}

type LogoutInput struct {
	SessionID string `json:"session_id"`
}
type ValidateTokenInput struct {
	TokenString string `json:"access_token"`
}

type RefreshTokenInput struct {
	TokenString string `json:"refresh_token"`
}

type Response struct {
	StatusCode int `json:"status"`
	Data       any `json:"data"`
}

type SessionResponse struct {
	SessionID  string    `json:"session_id"`
	DeviceName string    `json:"device_name"`
	DeviceType string    `json:"device_type"`
	LastUsedAt time.Time `json:"last_used_at"`
}
type LoginResponse struct {
	RefreshToken   string            `json:"refresh_token"`
	AccessToken    string            `json:"access_token"`
	CurrentSession SessionResponse   `json:"current_session"`
	OtherSessions  []SessionResponse `json:"other_sessions"`
}
type TokenValidationResponse struct {
	UserID   string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}
type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
}
