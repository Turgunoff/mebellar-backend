-- Migration: Add has_pin column to users table
-- This column tracks whether a user has set up a PIN code for app security

-- Add has_pin column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'has_pin'
    ) THEN
        ALTER TABLE users ADD COLUMN has_pin BOOLEAN DEFAULT FALSE;
        RAISE NOTICE 'Column has_pin added to users table';
    ELSE
        RAISE NOTICE 'Column has_pin already exists in users table';
    END IF;
END $$;

-- Add comment for documentation
COMMENT ON COLUMN users.has_pin IS 'Indicates whether user has set up PIN code for mobile app security';
