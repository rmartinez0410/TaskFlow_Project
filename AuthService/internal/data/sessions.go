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
	IPAddress  string
	UserAgent  string
}
type SessionModel struct {
	DB *sql.DB
}

func (m *SessionModel) Insert(s *Session, max int) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	const insert = `
		INSERT INTO sessions
		(token_hash, user_id, device_name, device_type, remember_me, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING session_id, created_at, last_used_at`

	err = tx.QueryRow(insert,
		s.TokenHash, s.UserID, s.DeviceName, s.DeviceType,
		s.RememberMe, s.ExpiresAt, s.IPAddress, s.UserAgent,
	).Scan(&s.SessionID, &s.CreatedAt, &s.LastUsedAt)
	if err != nil {
		return err
	}

	const prune = `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE session_id IN (
			SELECT session_id
			FROM sessions
			WHERE user_id = $1
			  AND revoked_at IS NULL
			  AND expires_at > NOW()
			ORDER BY last_used_at DESC
			OFFSET $2
			FOR UPDATE
		)`
	if _, err = tx.Exec(prune, s.UserID, max); err != nil {
		return err
	}

	return tx.Commit()
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
	stmt := `SELECT session_id, device_name, device_type, last_used_at, ip_address
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
	stmt := `UPDATE sessions SET revoked_at = NOW() WHERE session_id = $1`
	_, err := m.DB.Exec(stmt, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoRecord
	}
	return err
}
