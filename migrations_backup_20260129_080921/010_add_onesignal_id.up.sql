-- Migration: Add onesignal_id column to users table
-- This column stores the OneSignal Player ID for push notifications

-- Add onesignal_id column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'onesignal_id'
    ) THEN
        ALTER TABLE users ADD COLUMN onesignal_id VARCHAR(255);
        CREATE INDEX IF NOT EXISTS idx_users_onesignal_id ON users(onesignal_id);
        RAISE NOTICE 'Column onesignal_id added to users table';
    ELSE
        RAISE NOTICE 'Column onesignal_id already exists in users table';
    END IF;
END $$;
