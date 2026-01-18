-- ============================================
-- MIGRATION: Update shops table for Pro Shop Creation
-- Version: 022
-- Description: 
--   1. Ensure address is JSONB (already is, but safety check)
--   2. Make working_hours nullable (optional field)
--   3. Add simple_hours column for "HH:MM - HH:MM" format
-- ============================================

-- ============================================
-- UP MIGRATION
-- ============================================

-- 1. Safety check: Ensure shops.address is JSONB type
-- If for some reason there's VARCHAR data, convert it
DO $$
BEGIN
    -- Check if address column exists and convert if needed
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'shops' 
        AND column_name = 'address'
        AND data_type = 'character varying'
    ) THEN
        -- First add a temp column
        ALTER TABLE shops ADD COLUMN address_temp JSONB;
        
        -- Convert existing VARCHAR data to JSONB
        UPDATE shops 
        SET address_temp = jsonb_build_object('uz', COALESCE(address::text, ''))
        WHERE address IS NOT NULL;
        
        -- Drop old column and rename new one
        ALTER TABLE shops DROP COLUMN address;
        ALTER TABLE shops RENAME COLUMN address_temp TO address;
        
        -- Set default
        ALTER TABLE shops ALTER COLUMN address SET DEFAULT '{}';
        
        RAISE NOTICE 'Converted shops.address from VARCHAR to JSONB';
    ELSE
        RAISE NOTICE 'shops.address is already JSONB or does not exist';
    END IF;
END $$;

-- 2. Ensure working_hours allows NULL (for optional working hours)
ALTER TABLE shops ALTER COLUMN working_hours DROP DEFAULT;
ALTER TABLE shops ALTER COLUMN working_hours DROP NOT NULL;
ALTER TABLE shops ALTER COLUMN working_hours SET DEFAULT NULL;

-- 3. Add simple_hours column for simplified "HH:MM - HH:MM" format
-- This stores the working hours as simple text that can also be translated
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'shops' 
        AND column_name = 'simple_hours'
    ) THEN
        ALTER TABLE shops ADD COLUMN simple_hours JSONB DEFAULT NULL;
        COMMENT ON COLUMN shops.simple_hours IS 'Simplified working hours: {"uz": "09:00 - 18:00", "ru": "09:00 - 18:00", "en": "09:00 - 18:00"}';
        RAISE NOTICE 'Added simple_hours column to shops';
    END IF;
END $$;

-- 4. Update any NULL address to empty JSONB
UPDATE shops SET address = '{}' WHERE address IS NULL;

-- ============================================
-- DOWN MIGRATION (Rollback)
-- ============================================
/*
-- To rollback:
ALTER TABLE shops DROP COLUMN IF EXISTS simple_hours;
ALTER TABLE shops ALTER COLUMN working_hours SET DEFAULT '{}';
*/
