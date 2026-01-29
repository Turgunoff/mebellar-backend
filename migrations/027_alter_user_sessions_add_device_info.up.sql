-- Migration: Add device info columns to user_sessions table
-- This adds app_type, is_trusted, and expires_at for cross-app security

-- Add app_type column (client or seller)
ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS app_type VARCHAR(50) NOT NULL DEFAULT 'client';

-- Add is_trusted column (for 2FA trusted devices)
ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS is_trusted BOOLEAN DEFAULT FALSE;

-- Add expires_at column (session expiration time)
ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP;

-- Update existing sessions with default expires_at (30 days from now)
UPDATE user_sessions 
SET expires_at = NOW() + INTERVAL '30 days'
WHERE expires_at IS NULL;

-- Create index for app_type for faster queries
CREATE INDEX IF NOT EXISTS idx_user_sessions_app_type ON user_sessions(app_type);

-- Create index for expires_at for session cleanup
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Add constraint for app_type values
ALTER TABLE user_sessions 
DROP CONSTRAINT IF EXISTS check_app_type;

ALTER TABLE user_sessions 
ADD CONSTRAINT check_app_type CHECK (app_type IN ('client', 'seller', 'admin'));

-- Add comments for documentation
COMMENT ON COLUMN user_sessions.app_type IS 'App type: client (customer app), seller (seller app), admin (admin panel)';
COMMENT ON COLUMN user_sessions.is_trusted IS 'Whether this device is trusted (for 2FA bypass)';
COMMENT ON COLUMN user_sessions.expires_at IS 'Session expiration timestamp';
