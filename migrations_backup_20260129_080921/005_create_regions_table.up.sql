-- +migrate Up
-- Create regions table for Uzbekistan administrative divisions

CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(10) UNIQUE,
    is_active BOOLEAN DEFAULT true,
    ordering INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_regions_is_active ON regions(is_active);
CREATE INDEX IF NOT EXISTS idx_regions_ordering ON regions(ordering);

-- Seed data: 14 regions of Uzbekistan
INSERT INTO regions (name, code, is_active, ordering) VALUES
    ('Toshkent sh.', 'UZ-TK', true, 1),
    ('Toshkent vil.', 'UZ-TO', true, 2),
    ('Andijon', 'UZ-AN', true, 10),
    ('Buxoro', 'UZ-BU', true, 11),
    ('Farg''ona', 'UZ-FA', true, 12),
    ('Jizzax', 'UZ-JI', true, 13),
    ('Xorazm', 'UZ-XO', true, 14),
    ('Namangan', 'UZ-NG', true, 15),
    ('Navoiy', 'UZ-NW', true, 16),
    ('Qashqadaryo', 'UZ-QA', true, 17),
    ('Samarqand', 'UZ-SA', true, 18),
    ('Sirdaryo', 'UZ-SI', true, 19),
    ('Surxondaryo', 'UZ-SU', true, 20),
    ('Qoraqalpog''iston', 'UZ-QR', true, 21)
ON CONFLICT (code) DO NOTHING;

-- +migrate Down
DROP INDEX IF EXISTS idx_regions_ordering;
DROP INDEX IF EXISTS idx_regions_is_active;
DROP TABLE IF EXISTS regions;
