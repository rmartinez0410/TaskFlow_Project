CREATE TABLE sessions (
    session_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    token_hash      VARCHAR(60) NOT NULL UNIQUE,
    device_name     VARCHAR(200),
    device_type     VARCHAR(50),
    remember_me     BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ NULL,
    ip_address      INET,
    user_agent      TEXT,
    CONSTRAINT chk_expires_future CHECK (expires_at > created_at),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX idx_sessions_expires    ON sessions(expires_at);
CREATE INDEX idx_sessions_last_used  ON sessions(last_used_at);