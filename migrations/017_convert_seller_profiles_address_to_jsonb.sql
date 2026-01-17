-- ============================================
-- MIGRATION: Convert seller_profiles.address to JSONB
-- Version: 017
-- Description: seller_profiles jadvalidagi address maydonini JSONB ga o'zgartirish
-- ============================================

-- ============================================
-- UP MIGRATION
-- ============================================

-- 1. Yangi JSONB ustun qo'shish
ALTER TABLE seller_profiles 
ADD COLUMN IF NOT EXISTS address_jsonb JSONB DEFAULT '{}';

-- 2. Mavjud address ma'lumotlarini JSONB ga ko'chirish
-- Agar address bo'sh bo'lmasa, uni {"uz": "address_value"} formatida saqlash
UPDATE seller_profiles
SET address_jsonb = CASE 
    WHEN address IS NOT NULL AND address != '' THEN
        jsonb_build_object('uz', address)
    ELSE
        '{}'::jsonb
END
WHERE address_jsonb = '{}'::jsonb OR address_jsonb IS NULL;

-- 3. Eski address ustunini o'chirish
ALTER TABLE seller_profiles DROP COLUMN IF EXISTS address;

-- 4. Yangi ustunni address deb qayta nomlash
ALTER TABLE seller_profiles RENAME COLUMN address_jsonb TO address;

-- 5. GIN indeks qo'shish (JSONB uchun)
CREATE INDEX IF NOT EXISTS idx_seller_profiles_address_gin 
ON seller_profiles USING GIN (address);

-- Izoh
COMMENT ON COLUMN seller_profiles.address IS 'Do''kon manzili (multi-language JSONB: {"uz": "...", "ru": "...", "en": "..."})';

-- ============================================
-- DOWN MIGRATION (Rollback)
-- ============================================
-- Agar tiklash kerak bo'lsa quyidagini ishlatish:
/*
DROP INDEX IF EXISTS idx_seller_profiles_address_gin;
ALTER TABLE seller_profiles ADD COLUMN address_old VARCHAR(500);
UPDATE seller_profiles SET address_old = COALESCE(address->>'uz', '');
ALTER TABLE seller_profiles DROP COLUMN address;
ALTER TABLE seller_profiles RENAME COLUMN address_old TO address;
*/
