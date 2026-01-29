-- Migration: Create orders and order_items tables
-- Date: 2026-01-12
-- Description: Order management system for seller app

-- ============================================
-- ORDERS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES seller_profiles(id) ON DELETE CASCADE,
    
    -- Client info (snapshot at time of order)
    client_name VARCHAR(255) NOT NULL,
    client_phone VARCHAR(20) NOT NULL,
    client_address TEXT,
    
    -- Order details
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0,
    delivery_price NUMERIC(15, 2) DEFAULT 0,
    
    -- Status: 'new', 'confirmed', 'shipping', 'completed', 'cancelled'
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    
    -- Notes
    client_note TEXT,
    seller_note TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- ============================================
-- ORDER ITEMS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    
    -- Snapshot at time of purchase (product may change/delete later)
    product_name VARCHAR(255) NOT NULL,
    product_image VARCHAR(500),
    
    -- Quantity and pricing
    quantity INT NOT NULL DEFAULT 1,
    price NUMERIC(15, 2) NOT NULL,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- INDEXES
-- ============================================
-- Orders indexes
CREATE INDEX IF NOT EXISTS idx_orders_shop_id ON orders(shop_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_shop_status ON orders(shop_id, status);

-- Order items indexes
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

-- ============================================
-- TRIGGER: Update updated_at on orders
-- ============================================
CREATE OR REPLACE FUNCTION update_orders_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS orders_updated_at_trigger ON orders;
CREATE TRIGGER orders_updated_at_trigger
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_orders_updated_at();

-- ============================================
-- COMMENTS
-- ============================================
COMMENT ON TABLE orders IS 'Buyurtmalar jadvali';
COMMENT ON TABLE order_items IS 'Buyurtma mahsulotlari';
COMMENT ON COLUMN orders.status IS 'new=yangi, confirmed=tasdiqlangan, shipping=yetkazilmoqda, completed=yakunlangan, cancelled=bekor qilingan';

-- ============================================
-- GRANT PERMISSIONS
-- ============================================
GRANT ALL PRIVILEGES ON orders TO mebel_user;
GRANT ALL PRIVILEGES ON order_items TO mebel_user;
