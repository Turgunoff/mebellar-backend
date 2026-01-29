-- Migration: Add cancellation reason columns to orders table
-- Date: 2026-01-12
-- Description: Store rejection/cancellation reasons for analytics

-- Add cancellation_reason column
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cancellation_reason VARCHAR(255);

-- Add rejection_note column (for custom "Other" reasons)
ALTER TABLE orders ADD COLUMN IF NOT EXISTS rejection_note TEXT;

-- Add confirmed_at timestamp
ALTER TABLE orders ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMP WITH TIME ZONE;

-- Comment for documentation
COMMENT ON COLUMN orders.cancellation_reason IS 'Bekor qilish sababi: no_stock, price_issue, unreachable, customer_changed_mind, other';
COMMENT ON COLUMN orders.rejection_note IS 'Boshqa sabab uchun qo''shimcha izoh';
COMMENT ON COLUMN orders.confirmed_at IS 'Buyurtma tasdiqlangan vaqt';

-- Grant permissions
GRANT ALL PRIVILEGES ON orders TO mebel_user;
