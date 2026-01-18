-- ============================================
-- MIGRATION: Fix region codes and add JSONB name support
-- Version: 021
-- Description: Populate region codes and convert name to JSONB
-- ============================================

-- Step 1: Update existing regions with ISO 3166-2 codes (if NULL)
UPDATE regions SET code = 'UZ-TK' WHERE name LIKE '%Toshkent%sh%' OR name LIKE '%Tashkent%City%' AND code IS NULL;
UPDATE regions SET code = 'UZ-TO' WHERE (name LIKE '%Toshkent%vil%' OR name LIKE '%Tashkent%Region%') AND code IS NULL;
UPDATE regions SET code = 'UZ-AN' WHERE name LIKE '%Andijon%' OR name LIKE '%Andijan%' AND code IS NULL;
UPDATE regions SET code = 'UZ-BU' WHERE name LIKE '%Buxoro%' OR name LIKE '%Bukhara%' AND code IS NULL;
UPDATE regions SET code = 'UZ-FA' WHERE name LIKE '%Farg%ona%' OR name LIKE '%Fergana%' AND code IS NULL;
UPDATE regions SET code = 'UZ-JI' WHERE name LIKE '%Jizzax%' OR name LIKE '%Jizzakh%' AND code IS NULL;
UPDATE regions SET code = 'UZ-XO' WHERE name LIKE '%Xorazm%' OR name LIKE '%Khorezm%' AND code IS NULL;
UPDATE regions SET code = 'UZ-NG' WHERE name LIKE '%Namangan%' AND code IS NULL;
UPDATE regions SET code = 'UZ-NW' WHERE name LIKE '%Navoiy%' OR name LIKE '%Navoi%' AND code IS NULL;
UPDATE regions SET code = 'UZ-QA' WHERE name LIKE '%Qashqadaryo%' OR name LIKE '%Kashkadarya%' AND code IS NULL;
UPDATE regions SET code = 'UZ-SA' WHERE name LIKE '%Samarqand%' OR name LIKE '%Samarkand%' AND code IS NULL;
UPDATE regions SET code = 'UZ-SI' WHERE name LIKE '%Sirdaryo%' OR name LIKE '%Syrdarya%' AND code IS NULL;
UPDATE regions SET code = 'UZ-SU' WHERE name LIKE '%Surxondaryo%' OR name LIKE '%Surkhandarya%' AND code IS NULL;
UPDATE regions SET code = 'UZ-QR' WHERE name LIKE '%Qoraqalpog%' OR name LIKE '%Karakalpakstan%' AND code IS NULL;

-- Step 2: Add name_jsonb column if not exists (for multi-language support)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'regions' AND column_name = 'name_jsonb'
    ) THEN
        ALTER TABLE regions ADD COLUMN name_jsonb JSONB DEFAULT '{}';
    END IF;
END $$;

-- Step 3: Populate name_jsonb with multi-language values
UPDATE regions SET name_jsonb = jsonb_build_object(
    'uz', name,
    'ru', CASE code
        WHEN 'UZ-TK' THEN 'г. Ташкент'
        WHEN 'UZ-TO' THEN 'Ташкентская область'
        WHEN 'UZ-AN' THEN 'Андижанская область'
        WHEN 'UZ-BU' THEN 'Бухарская область'
        WHEN 'UZ-FA' THEN 'Ферганская область'
        WHEN 'UZ-JI' THEN 'Джизакская область'
        WHEN 'UZ-XO' THEN 'Хорезмская область'
        WHEN 'UZ-NG' THEN 'Наманганская область'
        WHEN 'UZ-NW' THEN 'Навоийская область'
        WHEN 'UZ-QA' THEN 'Кашкадарьинская область'
        WHEN 'UZ-SA' THEN 'Самаркандская область'
        WHEN 'UZ-SI' THEN 'Сырдарьинская область'
        WHEN 'UZ-SU' THEN 'Сурхандарьинская область'
        WHEN 'UZ-QR' THEN 'Республика Каракалпакстан'
        ELSE name
    END,
    'en', CASE code
        WHEN 'UZ-TK' THEN 'Tashkent City'
        WHEN 'UZ-TO' THEN 'Tashkent Region'
        WHEN 'UZ-AN' THEN 'Andijan Region'
        WHEN 'UZ-BU' THEN 'Bukhara Region'
        WHEN 'UZ-FA' THEN 'Fergana Region'
        WHEN 'UZ-JI' THEN 'Jizzakh Region'
        WHEN 'UZ-XO' THEN 'Khorezm Region'
        WHEN 'UZ-NG' THEN 'Namangan Region'
        WHEN 'UZ-NW' THEN 'Navoiy Region'
        WHEN 'UZ-QA' THEN 'Kashkadarya Region'
        WHEN 'UZ-SA' THEN 'Samarkand Region'
        WHEN 'UZ-SI' THEN 'Syrdarya Region'
        WHEN 'UZ-SU' THEN 'Surkhandarya Region'
        WHEN 'UZ-QR' THEN 'Republic of Karakalpakstan'
        ELSE name
    END
) WHERE name_jsonb IS NULL OR name_jsonb = '{}';

-- Step 4: Add updated_at column if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'regions' AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE regions ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
    END IF;
END $$;

-- Create index for JSONB name
CREATE INDEX IF NOT EXISTS idx_regions_name_jsonb ON regions USING GIN (name_jsonb);
