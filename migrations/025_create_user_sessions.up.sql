-- Migration: Create user_sessions table for multi-device session management
-- This table tracks all active sessions across devices for each user

-- Create user_sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(255) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    ip_address VARCHAR(50),
    last_active TIMESTAMP DEFAULT NOW(),
    is_current BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_device_id ON user_sessions(device_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_last_active ON user_sessions(last_active);

-- Add comment for documentation
COMMENT ON TABLE user_sessions IS 'Tracks active login sessions across multiple devices per user';
COMMENT ON COLUMN user_sessions.device_id IS 'Unique device identifier from mobile app';
COMMENT ON COLUMN user_sessions.is_current IS 'Marks the session making the current request';
