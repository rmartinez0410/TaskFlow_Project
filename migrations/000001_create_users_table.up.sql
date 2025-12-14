CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    activated BOOLEAN NOT NULL DEFAULT FALSE,
    username VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX users_email_idx ON users(email);
CREATE INDEX users_activated_idx ON users(activated);
CREATE INDEX users_username_idx ON users(username);
CREATE UNIQUE INDEX users_email_lower_idx ON users(LOWER(email));

CREATE OR REPLACE FUNCTION update_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at();