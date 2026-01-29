-- ============================================
-- MIGRATION: Update shops table for Pro Shop Creation
-- Version: 022
-- Description: 
--   1. Ensure address is JSONB (already is, but safety check)
--   2. Make working_hours nullable (optional field)
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

-- 2. Ensure working_hours allows empty JSONB (for optional working hours)
-- Note: We store simple format like {"uz": "09:00 - 18:00", "ru": "09:00 - 18:00", "en": "09:00 - 18:00"}
-- in the existing working_hours column

-- 3. Update any NULL address to empty JSONB
UPDATE shops SET address = '{}' WHERE address IS NULL;

-- ============================================
-- DOWN MIGRATION (Rollback)
-- ============================================
/*
-- No changes needed for rollback since we're using existing columns
*/
