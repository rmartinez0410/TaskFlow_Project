package data

import (
	"database/sql"
	"errors"
	"time"
)

type Session struct {
	SessionID  string
	TokenHash  []byte
	UserID     string
	DeviceName string
	DeviceType string
	RememberMe bool
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt time.Time
	RevokedAt  *time.Time
	IPAddress  *string
	UserAgent  string
}
type SessionModel struct {
	DB *sql.DB
}

func (m *SessionModel) Insert(s *Session) error {
	const query = `
       INSERT INTO sessions
       (session_id, token_hash, user_id, device_name, device_type, 
        remember_me, expires_at, created_at, last_used_at, ip_address, user_agent)
       VALUES ($1, $2, $3, $4, $5, $6, $7, 
               CASE WHEN $8 = '0001-01-01 00:00:00+00'::timestamptz THEN NOW() ELSE $8 END, 
               CASE WHEN $9 = '0001-01-01 00:00:00+00'::timestamptz THEN NOW() ELSE $9 END, 
               $10, $11)
       RETURNING session_id, created_at, last_used_at`

	err := m.DB.QueryRow(query,
		s.SessionID,
		s.TokenHash,
		s.UserID,
		s.DeviceName,
		s.DeviceType,
		s.RememberMe,
		s.ExpiresAt,
		s.CreatedAt,
		s.LastUsedAt,
		s.IPAddress,
		s.UserAgent,
	).Scan(&s.SessionID, &s.CreatedAt, &s.LastUsedAt)

	if err != nil {
		return err
	}

	return nil
}

func (m *SessionModel) GetByID(id string) (*Session, error) {
	const stmt = `SELECT session_id, token_hash, user_id, device_name, device_type, remember_me, 
       created_at, expires_at, last_used_at, revoked_at, ip_address, user_agent FROM   sessions
		WHERE  session_id = $1 AND  revoked_at IS NULL AND  expires_at  > NOW()`

	var s Session
	err := m.DB.QueryRow(stmt, id).Scan(
		&s.SessionID,
		&s.TokenHash,
		&s.UserID,
		&s.DeviceName,
		&s.DeviceType,
		&s.RememberMe,
		&s.CreatedAt,
		&s.ExpiresAt,
		&s.LastUsedAt,
		&s.RevokedAt,
		&s.IPAddress,
		&s.UserAgent,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}
	return &s, nil
}

func (m *SessionModel) GetOtherSessions(userID string, currentSessionID string) ([]SessionResponse, error) {
	stmt := `SELECT session_id, device_name, device_type, last_used_at
	FROM   sessions
	WHERE  user_id      = $1
	  AND  session_id  != $2
	  AND  revoked_at   IS NULL
	  AND  expires_at   > NOW()
	ORDER  BY last_used_at DESC`

	rows, err := m.DB.Query(stmt, userID, currentSessionID)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var sessions []SessionResponse
	for rows.Next() {
		var s SessionResponse
		if err := rows.Scan(
			&s.SessionID,
			&s.DeviceName,
			&s.DeviceType,
			&s.LastUsedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (m *SessionModel) UpdateLastUsed(id string) error {
	_, err := m.DB.Exec(`UPDATE sessions SET last_used_at = $1 WHERE session_id = $2`, time.Now(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoRecord
	}
	return err
}

func (m *SessionModel) Revoke(id string) error {
	stmt := `UPDATE sessions SET revoked_at = NOW() WHERE session_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	r, err := m.DB.Exec(stmt, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoRecord
	}
	return nil
}

func (m *SessionModel) GetByTokenHash(hash []byte) (*Session, error) {
	const q = `
		SELECT session_id, user_id, device_name, device_type, remember_me,
		       created_at, expires_at, last_used_at, revoked_at, ip_address, user_agent
		FROM sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		LIMIT 1`

	var s Session
	err := m.DB.QueryRow(q, hash).Scan(
		&s.SessionID,
		&s.UserID,
		&s.DeviceName,
		&s.DeviceType,
		&s.RememberMe,
		&s.CreatedAt,
		&s.ExpiresAt,
		&s.LastUsedAt,
		&s.RevokedAt,
		&s.IPAddress,
		&s.UserAgent,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}
	return &s, nil
}
