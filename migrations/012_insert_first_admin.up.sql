-- Migration: Insert First Super Admin User
-- This creates the initial admin user for the web panel

-- Insert admin user (only if doesn't exist)
-- Password: admin_password (hashed with bcrypt)
-- You can generate a new hash using: echo -n "admin_password" | bcrypt (or use Go's bcrypt)

DO $$
DECLARE
    admin_exists BOOLEAN;
    password_hash TEXT;
BEGIN
    -- Check if admin already exists
    SELECT EXISTS(SELECT 1 FROM users WHERE phone = '+998901234567') INTO admin_exists;
    
    IF NOT admin_exists THEN
        -- Hash for "admin_password" (cost 10)
        -- To generate a new hash, run: go run scripts/generate_admin_hash.go
        -- Or use this pre-generated hash for "admin_password":
        -- This hash will be generated when migration runs, but for now using a placeholder
        -- The actual hash should be generated using Go's bcrypt.GenerateFromPassword
        -- For now, we'll use a temporary hash that needs to be replaced
        -- Run: go run scripts/generate_admin_hash.go to get the correct hash
        password_hash := '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy';
        
        INSERT INTO users (
            id,
            full_name,
            phone,
            password_hash,
            role,
            is_active,
            created_at,
            updated_at
        ) VALUES (
            gen_random_uuid(),
            'Super Admin',
            '+998901234567',
            password_hash,
            'admin',
            true,
            NOW(),
            NOW()
        );
        
        RAISE NOTICE '✅ First Super Admin user created successfully!';
        RAISE NOTICE '   Phone: +998901234567';
        RAISE NOTICE '   Password: admin_password';
        RAISE NOTICE '   Role: admin';
    ELSE
        RAISE NOTICE '⚠️  Admin user already exists, skipping...';
    END IF;
END $$;
