-- Migration: Create category_attributes table
-- Date: 2026-01-18
-- Description: Server-driven UI for product specifications - Attributes are defined per category

-- ============================================
-- CATEGORY ATTRIBUTES TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS category_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    
    -- Attribute key (used as JSON key in product specs)
    key VARCHAR(100) NOT NULL,
    
    -- Input type: text, number, dropdown, switch
    type VARCHAR(20) NOT NULL CHECK (type IN ('text', 'number', 'dropdown', 'switch')),
    
    -- Translatable label: {"uz": "Material", "ru": "Материал", "en": "Material"}
    label JSONB NOT NULL,
    
    -- Options for dropdown type: [{"value": "mdf", "label": {"uz": "MDF", "ru": "МДФ", "en": "MDF"}}]
    options JSONB,
    
    -- Whether this attribute is required when creating a product
    is_required BOOLEAN DEFAULT FALSE,
    
    -- Display order (lower numbers appear first)
    sort_order INT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique keys per category
    CONSTRAINT unique_category_attribute_key UNIQUE (category_id, key)
);

-- ============================================
-- INDEXES
-- ============================================
CREATE INDEX IF NOT EXISTS idx_category_attributes_category_id ON category_attributes(category_id);
CREATE INDEX IF NOT EXISTS idx_category_attributes_sort_order ON category_attributes(sort_order);

-- ============================================
-- TRIGGER: Update updated_at on category_attributes
-- ============================================
CREATE OR REPLACE FUNCTION update_category_attributes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS category_attributes_updated_at_trigger ON category_attributes;
CREATE TRIGGER category_attributes_updated_at_trigger
    BEFORE UPDATE ON category_attributes
    FOR EACH ROW
    EXECUTE FUNCTION update_category_attributes_updated_at();

-- ============================================
-- COMMENTS
-- ============================================
COMMENT ON TABLE category_attributes IS 'Kategoriya atributlari - Server-driven UI uchun dinamik form maydonlari';
COMMENT ON COLUMN category_attributes.key IS 'JSON kalit nomi (masalan: "mechanism", "material")';
COMMENT ON COLUMN category_attributes.type IS 'Input turi: text, number, dropdown, switch';
COMMENT ON COLUMN category_attributes.label IS 'Tarjima qilingan yorliq: {"uz": "...", "ru": "...", "en": "..."}';
COMMENT ON COLUMN category_attributes.options IS 'Dropdown uchun variantlar: [{"value": "x", "label": {"uz": "...", ...}}]';
COMMENT ON COLUMN category_attributes.is_required IS 'Majburiy maydon yoki yo''q';
COMMENT ON COLUMN category_attributes.sort_order IS 'Ko''rsatish tartibi (kichik raqamlar birinchi)';

-- ============================================
-- GRANT PERMISSIONS
-- ============================================
GRANT ALL PRIVILEGES ON category_attributes TO mebel_user;
