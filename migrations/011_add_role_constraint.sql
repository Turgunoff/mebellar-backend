-- Migration: Add role column with constraint (if not exists)
-- This ensures role can only be: 'customer', 'seller', 'moderator', 'admin'

-- Add role column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'role'
    ) THEN
        ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'customer';
    END IF;
END $$;

-- Add CHECK constraint to enforce valid roles
DO $$
BEGIN
    -- Drop existing constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'users_role_check'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_role_check;
    END IF;
    
    -- Add new constraint
    ALTER TABLE users ADD CONSTRAINT users_role_check 
        CHECK (role IN ('customer', 'seller', 'moderator', 'admin'));
END $$;

-- Update existing NULL roles to 'customer'
UPDATE users SET role = 'customer' WHERE role IS NULL;

-- Set NOT NULL constraint
ALTER TABLE users ALTER COLUMN role SET NOT NULL;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'customer';
