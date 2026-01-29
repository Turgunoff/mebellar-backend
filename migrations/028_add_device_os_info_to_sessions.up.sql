-- Migration: Add device OS info columns to user_sessions table
-- This adds device_os, os_version, and app_version for detailed session tracking

-- Add device_os column (iOS, Android, etc.)
ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS device_os VARCHAR(20);

-- Add os_version column (e.g., '17.2', '14.0')
ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS os_version VARCHAR(50);

-- Add app_version column (e.g., '1.0.0', '1.0.0+12')
ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS app_version VARCHAR(20);

-- Create index for device_os for faster queries/filtering
CREATE INDEX IF NOT EXISTS idx_user_sessions_device_os ON user_sessions(device_os);

-- Create index for app_version for analytics and version tracking
CREATE INDEX IF NOT EXISTS idx_user_sessions_app_version ON user_sessions(app_version);

-- Add comments for documentation
COMMENT ON COLUMN user_sessions.device_os IS 'Operating system: iOS, Android, etc.';
COMMENT ON COLUMN user_sessions.os_version IS 'OS version: e.g., 17.2 for iOS, 14.0 for Android';
COMMENT ON COLUMN user_sessions.app_version IS 'Application version: e.g., 1.0.0 or 1.0.0+12';
