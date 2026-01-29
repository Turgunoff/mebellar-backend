-- Migration: Create cancellation_reasons table
-- Date: 2026-01-12
-- Description: Dynamic cancellation reasons for order rejection

-- Create table
CREATE TABLE IF NOT EXISTS cancellation_reasons (
    id SERIAL PRIMARY KEY,
    reason_text VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create unique index on reason_text to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_cancellation_reasons_text ON cancellation_reasons(reason_text);

-- Seed default data
INSERT INTO cancellation_reasons (reason_text, sort_order) VALUES
    ('❌ Mahsulot omborda qolmadi', 1),
    ('📞 Mijoz bilan bog''lanib bo''lmadi', 2),
    ('💸 Narx yoki shartlar to''g''ri kelmadi', 3),
    ('🔄 Mijoz fikridan qaytdi', 4),
    ('🚚 Yetkazib berish imkonsiz', 5),
    ('⚠️ Xato yoki dublikat buyurtma', 6)
ON CONFLICT (reason_text) DO NOTHING;

-- Grant permissions
GRANT SELECT ON cancellation_reasons TO mebel_user;
GRANT USAGE, SELECT ON SEQUENCE cancellation_reasons_id_seq TO mebel_user;

-- Comment
COMMENT ON TABLE cancellation_reasons IS 'Buyurtmani bekor qilish sabablari (dinamik)';
