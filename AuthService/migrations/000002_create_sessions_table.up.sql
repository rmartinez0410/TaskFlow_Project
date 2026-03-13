CREATE TABLE sessions (
    session_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    token_hash      BYTEA UNIQUE,
    device_name     VARCHAR(200),
    device_type     VARCHAR(50),
    remember_me     BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ DEFAULT NULL,
    ip_address      INET,
    user_agent      TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX idx_sessions_expires    ON sessions(expires_at);
CREATE INDEX idx_sessions_last_used  ON sessions(last_used_at);

CREATE OR REPLACE FUNCTION prune_user_sessions()
RETURNS TRIGGER AS $$
BEGIN
UPDATE sessions
SET revoked_at = NOW()
WHERE session_id IN (
    SELECT session_id
    FROM sessions
    WHERE user_id = NEW.user_id
      AND revoked_at IS NULL
      AND expires_at > NOW()
    ORDER BY last_used_at DESC, created_at DESC, session_id DESC
    OFFSET 5
);
RETURN NEW;
END;
$$ LANGUAGE plpgsql;;

CREATE TRIGGER trigger_prune_sessions
    AFTER INSERT ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION prune_user_sessions();