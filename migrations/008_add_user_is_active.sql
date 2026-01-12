-- Add is_active column to users table for soft delete functionality
-- Run this migration if the column doesn't exist

-- Add is_active column (default TRUE for existing users)
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

-- Update existing users to be active
UPDATE users SET is_active = TRUE WHERE is_active IS NULL;

-- Add role column if not exists (for role-based access)
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) DEFAULT 'customer';

-- Create index for faster queries on active users
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Verify the changes
SELECT column_name, data_type, column_default 
FROM information_schema.columns 
WHERE table_name = 'users' 
AND column_name IN ('is_active', 'role');
