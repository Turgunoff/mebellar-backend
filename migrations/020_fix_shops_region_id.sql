-- ============================================
-- MIGRATION: Fix shops.region_id type
-- Version: 020
-- Description: Ensure region_id is INTEGER type (fix UUID mismatch if exists)
-- ============================================

-- Check and fix region_id column type if it's UUID
DO $$
BEGIN
    -- Check if region_id column exists and is UUID type
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'shops' 
        AND column_name = 'region_id' 
        AND data_type = 'uuid'
    ) THEN
        -- Drop the foreign key constraint if exists
        ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_region_id_fkey;
        
        -- Change column type from UUID to INTEGER
        ALTER TABLE shops 
        ALTER COLUMN region_id TYPE INTEGER 
        USING NULL; -- Set all values to NULL during conversion
        
        -- Re-add the foreign key constraint
        ALTER TABLE shops 
        ADD CONSTRAINT shops_region_id_fkey 
        FOREIGN KEY (region_id) REFERENCES regions(id) ON DELETE SET NULL;
        
        RAISE NOTICE 'region_id column converted from UUID to INTEGER';
    ELSE
        RAISE NOTICE 'region_id column is already INTEGER or does not exist';
    END IF;
END $$;

-- Ensure index exists
CREATE INDEX IF NOT EXISTS idx_shops_region_id ON shops(region_id);
