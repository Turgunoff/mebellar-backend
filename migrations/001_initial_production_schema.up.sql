-- ============================================
-- PRODUCTION INITIAL SCHEMA
-- Generated from existing database structure
-- Date: 2026-01-29
-- ============================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================
-- TABLES
-- ============================================

-- 1. USERS
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(100),
    phone VARCHAR(20) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'buyer' CHECK (role IN ('buyer', 'seller', 'admin')),
    email VARCHAR(255),
    avatar_url TEXT,
    is_active BOOLEAN DEFAULT true,
    onesignal_id VARCHAR(255),
    has_pin BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. SELLER PROFILES
CREATE TABLE IF NOT EXISTS seller_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    legal_name VARCHAR(255),
    tax_id VARCHAR(50),
    bank_account VARCHAR(50),
    bank_name VARCHAR(255),
    passport_seria_number VARCHAR(20),
    legal_address TEXT,
    social_links JSONB DEFAULT '{}',
    is_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. REGIONS
CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name JSONB NOT NULL,
    code VARCHAR(10) UNIQUE,
    is_active BOOLEAN DEFAULT true,
    ordering INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. SHOPS
CREATE TABLE IF NOT EXISTS shops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES seller_profiles(id) ON DELETE CASCADE,
    name JSONB NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description JSONB,
    logo_url VARCHAR(500),
    banner_url VARCHAR(500),
    phone VARCHAR(20),
    address JSONB,
    latitude FLOAT8,
    longitude FLOAT8,
    region_id INTEGER REFERENCES regions(id),
    working_hours JSONB,
    simple_hours JSONB,
    is_main BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    rating FLOAT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. CATEGORIES
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    name JSONB NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    icon_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. PRODUCTS
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    name JSONB NOT NULL,
    description JSONB,
    price NUMERIC(15, 2) NOT NULL,
    discount_price NUMERIC(15, 2),
    images TEXT[],
    specs JSONB,
    variants JSONB,
    rating NUMERIC(3, 2) DEFAULT 0,
    is_new BOOLEAN DEFAULT false,
    is_popular BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    sold_count INTEGER DEFAULT 0,
    view_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. FAVORITES
CREATE TABLE IF NOT EXISTS favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, product_id)
);

-- 8. ORDERS
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    client_name VARCHAR(255) NOT NULL,
    client_phone VARCHAR(20) NOT NULL,
    client_address TEXT,
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0,
    delivery_price NUMERIC(15, 2) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    client_note TEXT,
    seller_note TEXT,
    cancellation_reason VARCHAR(255),
    rejection_note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    confirmed_at TIMESTAMP WITH TIME ZONE
);

-- 9. ORDER ITEMS
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    product_name VARCHAR(255) NOT NULL,
    product_image VARCHAR(500),
    quantity INTEGER NOT NULL DEFAULT 1,
    price NUMERIC(15, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 10. CANCELLATION REASONS
CREATE TABLE IF NOT EXISTS cancellation_reasons (
    id SERIAL PRIMARY KEY,
    reason_text VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 11. CATEGORY ATTRIBUTES
CREATE TABLE IF NOT EXISTS category_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    label JSONB NOT NULL,
    options JSONB,
    is_required BOOLEAN DEFAULT false,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 12. USER SESSIONS
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(255) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    device_os VARCHAR(20),
    os_version VARCHAR(50),
    app_type VARCHAR(20),
    app_version VARCHAR(20),
    ip_address VARCHAR(50),
    refresh_token_hash VARCHAR(255),
    is_current BOOLEAN DEFAULT true,
    is_trusted BOOLEAN DEFAULT false,
    last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 13. BANNERS
CREATE TABLE IF NOT EXISTS banners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title JSONB NOT NULL,
    subtitle JSONB,
    image_url TEXT NOT NULL,
    target_type VARCHAR(50) DEFAULT 'none',
    target_id VARCHAR(255),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- INDEXES
-- ============================================

-- Users
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Seller Profiles
CREATE INDEX IF NOT EXISTS idx_seller_profiles_user_id ON seller_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_seller_profiles_is_verified ON seller_profiles(is_verified);

-- Regions
CREATE INDEX IF NOT EXISTS idx_regions_is_active ON regions(is_active);
CREATE INDEX IF NOT EXISTS idx_regions_ordering ON regions(ordering);
CREATE INDEX IF NOT EXISTS idx_regions_name_jsonb ON regions USING GIN (name);

-- Shops
CREATE INDEX IF NOT EXISTS idx_shops_seller_id ON shops(seller_id);
CREATE INDEX IF NOT EXISTS idx_shops_slug ON shops(slug);
CREATE INDEX IF NOT EXISTS idx_shops_is_active ON shops(is_active);
CREATE INDEX IF NOT EXISTS idx_shops_region_id ON shops(region_id);

-- Categories
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_is_active ON categories(is_active);
CREATE INDEX IF NOT EXISTS idx_categories_name_jsonb ON categories USING GIN (name);

-- Products
CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_is_active ON products(is_active);
CREATE INDEX IF NOT EXISTS idx_products_is_new ON products(is_new);
CREATE INDEX IF NOT EXISTS idx_products_is_popular ON products(is_popular);
CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);

-- Orders
CREATE INDEX IF NOT EXISTS idx_orders_shop_id ON orders(shop_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);

-- Order Items
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

-- Category Attributes
CREATE INDEX IF NOT EXISTS idx_category_attributes_category_id ON category_attributes(category_id);

-- User Sessions
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_device_id ON user_sessions(device_id);

-- Banners
CREATE INDEX IF NOT EXISTS idx_banners_is_active ON banners(is_active);
CREATE INDEX IF NOT EXISTS idx_banners_sort_order ON banners(sort_order);

-- ============================================
-- TRIGGERS
-- ============================================

-- Update updated_at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_seller_profiles_updated_at ON seller_profiles;
CREATE TRIGGER update_seller_profiles_updated_at BEFORE UPDATE ON seller_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_shops_updated_at ON shops;
CREATE TRIGGER update_shops_updated_at BEFORE UPDATE ON shops
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_products_updated_at ON products;
CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- GRANTS
-- ============================================
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO mebel_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO mebel_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO mebel_user;

-- ============================================
-- COMMENTS
-- ============================================
COMMENT ON TABLE users IS 'Foydalanuvchilar jadvali';
COMMENT ON TABLE seller_profiles IS 'Sotuvchi profillari';
COMMENT ON TABLE shops IS 'Do''konlar (Multi-shop support)';
COMMENT ON TABLE regions IS 'O''zbekiston hududlari';
COMMENT ON TABLE categories IS 'Mahsulot kategoriyalari';
COMMENT ON TABLE products IS 'Mahsulotlar';
COMMENT ON TABLE orders IS 'Buyurtmalar';
COMMENT ON TABLE banners IS 'Banner reklamalar';
