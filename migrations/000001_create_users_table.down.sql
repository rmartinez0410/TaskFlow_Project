DROP TRIGGER IF EXISTS users_updated_at_trigger ON users;
DROP FUNCTION IF EXISTS update_updated_at;
DROP INDEX IF EXISTS users_email_lower_idx;
DROP INDEX IF EXISTS users_username_idx;
DROP INDEX IF EXISTS users_activated_idx;
DROP INDEX IF EXISTS users_email_idx;
DROP TABLE IF EXISTS users;