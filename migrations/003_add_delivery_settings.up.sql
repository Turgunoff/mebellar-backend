-- +migrate Up
-- Add delivery_settings column to products table
-- JSON structure for flexible regional delivery pricing

ALTER TABLE products 
ADD COLUMN IF NOT EXISTS delivery_settings JSONB DEFAULT '{
    "default": {
        "delivery_price": 0,
        "delivery_days": "3-5",
        "has_installation": false,
        "installation_price": 0,
        "comment": ""
    },
    "overrides": []
}'::jsonb;

-- Create index for better JSONB query performance
CREATE INDEX IF NOT EXISTS idx_products_delivery_settings 
ON products USING GIN (delivery_settings);

-- Comment explaining the structure
COMMENT ON COLUMN products.delivery_settings IS '
Regional delivery settings with default and overrides strategy.
Structure:
{
    "default": {
        "delivery_price": 200000,      -- Default price in so''m
        "delivery_days": "5-7",        -- Delivery time range
        "has_installation": false,     -- Installation service available
        "installation_price": 0,       -- Installation cost
        "comment": "Optional note"     -- Seller note
    },
    "overrides": [
        {
            "region_id": "tashkent_city",
            "region_name": "Toshkent sh.",
            "delivery_price": 50000,
            "delivery_days": "1",
            "has_installation": true,
            "installation_price": 100000
        }
    ]
}
';

-- +migrate Down
DROP INDEX IF EXISTS idx_products_delivery_settings;
ALTER TABLE products DROP COLUMN IF EXISTS delivery_settings;
