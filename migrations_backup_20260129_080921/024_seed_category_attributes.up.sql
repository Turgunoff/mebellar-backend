-- Migration: Seed category attributes for all furniture categories
-- Date: 2026-01-18
-- Description: Server-Driven UI - Initial attributes for all categories

-- ============================================
-- COMMON ATTRIBUTES FOR ALL CATEGORIES
-- ============================================

-- 1. Office Furniture (Ofis mebellari) - 19287ffa-4534-4f61-9d72-72b5f66bb247
INSERT INTO category_attributes (category_id, key, type, label, options, is_required, sort_order) VALUES
-- Material
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'material', 'dropdown', 
 '{"uz": "Material", "ru": "Материал", "en": "Material"}',
 '[{"value": "mdf", "label": {"uz": "MDF", "ru": "МДФ", "en": "MDF"}},
   {"value": "dsp", "label": {"uz": "DSP (Siqilgan yog''och)", "ru": "ДСП", "en": "Particleboard"}},
   {"value": "natural_wood", "label": {"uz": "Tabiiy yog''och", "ru": "Натуральное дерево", "en": "Natural Wood"}},
   {"value": "metal", "label": {"uz": "Metall", "ru": "Металл", "en": "Metal"}},
   {"value": "plastic", "label": {"uz": "Plastik", "ru": "Пластик", "en": "Plastic"}},
   {"value": "glass", "label": {"uz": "Shisha", "ru": "Стекло", "en": "Glass"}}]',
 true, 1),

-- Color
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'color', 'dropdown',
 '{"uz": "Rang", "ru": "Цвет", "en": "Color"}',
 '[{"value": "white", "label": {"uz": "Oq", "ru": "Белый", "en": "White"}},
   {"value": "black", "label": {"uz": "Qora", "ru": "Чёрный", "en": "Black"}},
   {"value": "brown", "label": {"uz": "Jigarrang", "ru": "Коричневый", "en": "Brown"}},
   {"value": "beige", "label": {"uz": "Bej", "ru": "Бежевый", "en": "Beige"}},
   {"value": "gray", "label": {"uz": "Kulrang", "ru": "Серый", "en": "Gray"}},
   {"value": "oak", "label": {"uz": "Eman", "ru": "Дуб", "en": "Oak"}},
   {"value": "wenge", "label": {"uz": "Venge", "ru": "Венге", "en": "Wenge"}}]',
 true, 2),

-- Dimensions
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'dimensions', 'text',
 '{"uz": "O''lchamlari (UxBxK)", "ru": "Размеры (ШxГxВ)", "en": "Dimensions (WxDxH)"}',
 NULL, false, 3),

-- Height adjustable (for office chairs/desks)
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'height_adjustable', 'switch',
 '{"uz": "Balandligi sozlanadi", "ru": "Регулируемая высота", "en": "Height Adjustable"}',
 NULL, false, 4),

-- Mechanism (for office chairs)
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'chair_mechanism', 'dropdown',
 '{"uz": "Kreslo mexanizmi", "ru": "Механизм кресла", "en": "Chair Mechanism"}',
 '[{"value": "tilt", "label": {"uz": "Tilt", "ru": "Тилт", "en": "Tilt"}},
   {"value": "synchro", "label": {"uz": "Sinxron", "ru": "Синхронный", "en": "Synchro"}},
   {"value": "multiblock", "label": {"uz": "Multiblok", "ru": "Мультиблок", "en": "Multiblock"}},
   {"value": "none", "label": {"uz": "Yo''q", "ru": "Нет", "en": "None"}}]',
 false, 5),

-- Armrests
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'armrests', 'switch',
 '{"uz": "Qo''l tayanchli", "ru": "С подлокотниками", "en": "With Armrests"}',
 NULL, false, 6),

-- Wheels
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'wheels', 'switch',
 '{"uz": "G''ildirakli", "ru": "На колёсиках", "en": "With Wheels"}',
 NULL, false, 7),

-- Warranty
('19287ffa-4534-4f61-9d72-72b5f66bb247', 'warranty', 'dropdown',
 '{"uz": "Kafolat", "ru": "Гарантия", "en": "Warranty"}',
 '[{"value": "6_months", "label": {"uz": "6 oy", "ru": "6 месяцев", "en": "6 months"}},
   {"value": "1_year", "label": {"uz": "1 yil", "ru": "1 год", "en": "1 year"}},
   {"value": "2_years", "label": {"uz": "2 yil", "ru": "2 года", "en": "2 years"}},
   {"value": "3_years", "label": {"uz": "3 yil", "ru": "3 года", "en": "3 years"}}]',
 false, 8);


-- 2. Kitchen & Dining (Oshxona mebeli) - 324e69bd-617a-4212-83ea-a7ae9aab1e9a
INSERT INTO category_attributes (category_id, key, type, label, options, is_required, sort_order) VALUES
-- Material
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'material', 'dropdown', 
 '{"uz": "Material", "ru": "Материал", "en": "Material"}',
 '[{"value": "mdf", "label": {"uz": "MDF", "ru": "МДФ", "en": "MDF"}},
   {"value": "dsp", "label": {"uz": "DSP (Siqilgan yog''och)", "ru": "ДСП", "en": "Particleboard"}},
   {"value": "natural_wood", "label": {"uz": "Tabiiy yog''och", "ru": "Натуральное дерево", "en": "Natural Wood"}},
   {"value": "metal", "label": {"uz": "Metall", "ru": "Металл", "en": "Metal"}},
   {"value": "glass", "label": {"uz": "Shisha", "ru": "Стекло", "en": "Glass"}}]',
 true, 1),

-- Color
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'color', 'dropdown',
 '{"uz": "Rang", "ru": "Цвет", "en": "Color"}',
 '[{"value": "white", "label": {"uz": "Oq", "ru": "Белый", "en": "White"}},
   {"value": "black", "label": {"uz": "Qora", "ru": "Чёрный", "en": "Black"}},
   {"value": "brown", "label": {"uz": "Jigarrang", "ru": "Коричневый", "en": "Brown"}},
   {"value": "beige", "label": {"uz": "Bej", "ru": "Бежевый", "en": "Beige"}},
   {"value": "gray", "label": {"uz": "Kulrang", "ru": "Серый", "en": "Gray"}},
   {"value": "oak", "label": {"uz": "Eman", "ru": "Дуб", "en": "Oak"}},
   {"value": "wenge", "label": {"uz": "Venge", "ru": "Венге", "en": "Wenge"}}]',
 true, 2),

-- Dimensions
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'dimensions', 'text',
 '{"uz": "O''lchamlari (UxBxK)", "ru": "Размеры (ШxГxВ)", "en": "Dimensions (WxDxH)"}',
 NULL, false, 3),

-- Number of seats (for dining tables)
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'seats', 'dropdown',
 '{"uz": "O''rindiqlar soni", "ru": "Количество мест", "en": "Number of Seats"}',
 '[{"value": "2", "label": {"uz": "2 kishilik", "ru": "На 2 персоны", "en": "2 seats"}},
   {"value": "4", "label": {"uz": "4 kishilik", "ru": "На 4 персоны", "en": "4 seats"}},
   {"value": "6", "label": {"uz": "6 kishilik", "ru": "На 6 персон", "en": "6 seats"}},
   {"value": "8", "label": {"uz": "8 kishilik", "ru": "На 8 персон", "en": "8 seats"}},
   {"value": "10+", "label": {"uz": "10+ kishilik", "ru": "На 10+ персон", "en": "10+ seats"}}]',
 false, 4),

-- Extendable table
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'extendable', 'switch',
 '{"uz": "Kengaytiriladigan", "ru": "Раздвижной", "en": "Extendable"}',
 NULL, false, 5),

-- Water resistant
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'water_resistant', 'switch',
 '{"uz": "Suv o''tkazmaydigan", "ru": "Влагостойкий", "en": "Water Resistant"}',
 NULL, false, 6),

-- Surface type
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'surface_type', 'dropdown',
 '{"uz": "Sirt turi", "ru": "Тип поверхности", "en": "Surface Type"}',
 '[{"value": "glossy", "label": {"uz": "Yaltiroq", "ru": "Глянец", "en": "Glossy"}},
   {"value": "matte", "label": {"uz": "Mat", "ru": "Матовый", "en": "Matte"}},
   {"value": "textured", "label": {"uz": "Teksturali", "ru": "Текстурированный", "en": "Textured"}}]',
 false, 7),

-- Warranty
('324e69bd-617a-4212-83ea-a7ae9aab1e9a', 'warranty', 'dropdown',
 '{"uz": "Kafolat", "ru": "Гарантия", "en": "Warranty"}',
 '[{"value": "6_months", "label": {"uz": "6 oy", "ru": "6 месяцев", "en": "6 months"}},
   {"value": "1_year", "label": {"uz": "1 yil", "ru": "1 год", "en": "1 year"}},
   {"value": "2_years", "label": {"uz": "2 yil", "ru": "2 года", "en": "2 years"}}]',
 false, 8);


-- 3. Sofas & Armchairs (Yumshoq mebel) - 41ba213b-84dc-436e-b2cc-210064983c05
INSERT INTO category_attributes (category_id, key, type, label, options, is_required, sort_order) VALUES
-- Mechanism (CRITICAL for sofas!)
('41ba213b-84dc-436e-b2cc-210064983c05', 'mechanism', 'dropdown',
 '{"uz": "Mexanizm", "ru": "Механизм", "en": "Mechanism"}',
 '[{"value": "delfin", "label": {"uz": "Delfin", "ru": "Дельфин", "en": "Dolphin"}},
   {"value": "pantograph", "label": {"uz": "Pantograf", "ru": "Пантограф", "en": "Pantograph"}},
   {"value": "accordion", "label": {"uz": "Akkordeon", "ru": "Аккордеон", "en": "Accordion"}},
   {"value": "eurobook", "label": {"uz": "Evrokitob", "ru": "Еврокнижка", "en": "Eurobook"}},
   {"value": "click_clack", "label": {"uz": "Klik-klyak", "ru": "Клик-кляк", "en": "Click-Clack"}},
   {"value": "roll_out", "label": {"uz": "Surg''ich", "ru": "Выкатной", "en": "Roll-out"}},
   {"value": "none", "label": {"uz": "Mexanizmsiz", "ru": "Без механизма", "en": "No mechanism"}}]',
 true, 1),

-- Upholstery/Fabric
('41ba213b-84dc-436e-b2cc-210064983c05', 'upholstery', 'dropdown',
 '{"uz": "Qoplama", "ru": "Обивка", "en": "Upholstery"}',
 '[{"value": "velvet", "label": {"uz": "Baxmal", "ru": "Велюр", "en": "Velvet"}},
   {"value": "leather", "label": {"uz": "Teri", "ru": "Кожа", "en": "Leather"}},
   {"value": "eco_leather", "label": {"uz": "Eko-teri", "ru": "Эко-кожа", "en": "Eco-leather"}},
   {"value": "textile", "label": {"uz": "Mato", "ru": "Ткань", "en": "Textile"}},
   {"value": "microfiber", "label": {"uz": "Mikrofibra", "ru": "Микрофибра", "en": "Microfiber"}},
   {"value": "chenille", "label": {"uz": "Shenil", "ru": "Шенилл", "en": "Chenille"}}]',
 true, 2),

-- Color
('41ba213b-84dc-436e-b2cc-210064983c05', 'color', 'dropdown',
 '{"uz": "Rang", "ru": "Цвет", "en": "Color"}',
 '[{"value": "white", "label": {"uz": "Oq", "ru": "Белый", "en": "White"}},
   {"value": "black", "label": {"uz": "Qora", "ru": "Чёрный", "en": "Black"}},
   {"value": "brown", "label": {"uz": "Jigarrang", "ru": "Коричневый", "en": "Brown"}},
   {"value": "beige", "label": {"uz": "Bej", "ru": "Бежевый", "en": "Beige"}},
   {"value": "gray", "label": {"uz": "Kulrang", "ru": "Серый", "en": "Gray"}},
   {"value": "blue", "label": {"uz": "Ko''k", "ru": "Синий", "en": "Blue"}},
   {"value": "green", "label": {"uz": "Yashil", "ru": "Зелёный", "en": "Green"}},
   {"value": "red", "label": {"uz": "Qizil", "ru": "Красный", "en": "Red"}}]',
 true, 3),

-- Number of seats
('41ba213b-84dc-436e-b2cc-210064983c05', 'seats', 'dropdown',
 '{"uz": "O''rindiqlar soni", "ru": "Количество мест", "en": "Number of Seats"}',
 '[{"value": "1", "label": {"uz": "1 kishilik (Kreslo)", "ru": "1 место (Кресло)", "en": "1 seat (Armchair)"}},
   {"value": "2", "label": {"uz": "2 kishilik", "ru": "2 места", "en": "2 seats"}},
   {"value": "3", "label": {"uz": "3 kishilik", "ru": "3 места", "en": "3 seats"}},
   {"value": "4", "label": {"uz": "4 kishilik", "ru": "4 места", "en": "4 seats"}},
   {"value": "corner", "label": {"uz": "Burchakli", "ru": "Угловой", "en": "Corner sofa"}}]',
 false, 4),

-- Sleeping place dimensions
('41ba213b-84dc-436e-b2cc-210064983c05', 'sleeping_size', 'dropdown',
 '{"uz": "Uxlash joyi o''lchami", "ru": "Размер спального места", "en": "Sleeping Place Size"}',
 '[{"value": "120x190", "label": {"uz": "120x190 sm", "ru": "120x190 см", "en": "120x190 cm"}},
   {"value": "140x190", "label": {"uz": "140x190 sm", "ru": "140x190 см", "en": "140x190 cm"}},
   {"value": "140x200", "label": {"uz": "140x200 sm", "ru": "140x200 см", "en": "140x200 cm"}},
   {"value": "160x200", "label": {"uz": "160x200 sm", "ru": "160x200 см", "en": "160x200 cm"}},
   {"value": "180x200", "label": {"uz": "180x200 sm", "ru": "180x200 см", "en": "180x200 cm"}},
   {"value": "none", "label": {"uz": "Yo''q", "ru": "Нет", "en": "None"}}]',
 false, 5),

-- Has storage
('41ba213b-84dc-436e-b2cc-210064983c05', 'storage', 'switch',
 '{"uz": "Ichki bo''lma (saqlash joyi)", "ru": "С ящиком для белья", "en": "With Storage"}',
 NULL, false, 6),

-- Frame material
('41ba213b-84dc-436e-b2cc-210064983c05', 'frame_material', 'dropdown',
 '{"uz": "Karkass materiali", "ru": "Материал каркаса", "en": "Frame Material"}',
 '[{"value": "wood", "label": {"uz": "Yog''och", "ru": "Дерево", "en": "Wood"}},
   {"value": "metal", "label": {"uz": "Metall", "ru": "Металл", "en": "Metal"}},
   {"value": "plywood", "label": {"uz": "Fanera", "ru": "Фанера", "en": "Plywood"}}]',
 false, 7),

-- Warranty
('41ba213b-84dc-436e-b2cc-210064983c05', 'warranty', 'dropdown',
 '{"uz": "Kafolat", "ru": "Гарантия", "en": "Warranty"}',
 '[{"value": "1_year", "label": {"uz": "1 yil", "ru": "1 год", "en": "1 year"}},
   {"value": "2_years", "label": {"uz": "2 yil", "ru": "2 года", "en": "2 years"}},
   {"value": "3_years", "label": {"uz": "3 yil", "ru": "3 года", "en": "3 years"}},
   {"value": "5_years", "label": {"uz": "5 yil", "ru": "5 лет", "en": "5 years"}}]',
 false, 8);


-- 4. Wardrobes (Shkaflar) - 42c20ed3-db22-4268-95f2-1f9c36455f1f
INSERT INTO category_attributes (category_id, key, type, label, options, is_required, sort_order) VALUES
-- Material
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'material', 'dropdown', 
 '{"uz": "Material", "ru": "Материал", "en": "Material"}',
 '[{"value": "mdf", "label": {"uz": "MDF", "ru": "МДФ", "en": "MDF"}},
   {"value": "dsp", "label": {"uz": "DSP (ЛДСП)", "ru": "ЛДСП", "en": "Laminated Particleboard"}},
   {"value": "natural_wood", "label": {"uz": "Tabiiy yog''och", "ru": "Натуральное дерево", "en": "Natural Wood"}}]',
 true, 1),

-- Color
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'color', 'dropdown',
 '{"uz": "Rang", "ru": "Цвет", "en": "Color"}',
 '[{"value": "white", "label": {"uz": "Oq", "ru": "Белый", "en": "White"}},
   {"value": "black", "label": {"uz": "Qora", "ru": "Чёрный", "en": "Black"}},
   {"value": "brown", "label": {"uz": "Jigarrang", "ru": "Коричневый", "en": "Brown"}},
   {"value": "beige", "label": {"uz": "Bej", "ru": "Бежевый", "en": "Beige"}},
   {"value": "gray", "label": {"uz": "Kulrang", "ru": "Серый", "en": "Gray"}},
   {"value": "oak", "label": {"uz": "Eman", "ru": "Дуб", "en": "Oak"}},
   {"value": "wenge", "label": {"uz": "Venge", "ru": "Венге", "en": "Wenge"}},
   {"value": "sonoma", "label": {"uz": "Sonoma", "ru": "Сонома", "en": "Sonoma"}}]',
 true, 2),

-- Dimensions
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'dimensions', 'text',
 '{"uz": "O''lchamlari (UxBxK)", "ru": "Размеры (ШxГxВ)", "en": "Dimensions (WxDxH)"}',
 NULL, true, 3),

-- Number of doors
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'doors_count', 'dropdown',
 '{"uz": "Eshiklar soni", "ru": "Количество дверей", "en": "Number of Doors"}',
 '[{"value": "2", "label": {"uz": "2 ta", "ru": "2 шт", "en": "2 doors"}},
   {"value": "3", "label": {"uz": "3 ta", "ru": "3 шт", "en": "3 doors"}},
   {"value": "4", "label": {"uz": "4 ta", "ru": "4 шт", "en": "4 doors"}},
   {"value": "5", "label": {"uz": "5 ta", "ru": "5 шт", "en": "5 doors"}},
   {"value": "6", "label": {"uz": "6 ta", "ru": "6 шт", "en": "6 doors"}}]',
 true, 4),

-- Door type
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'door_type', 'dropdown',
 '{"uz": "Eshik turi", "ru": "Тип дверей", "en": "Door Type"}',
 '[{"value": "swing", "label": {"uz": "Ochiluvchi", "ru": "Распашные", "en": "Swing doors"}},
   {"value": "sliding", "label": {"uz": "Siljuvchi (kupe)", "ru": "Раздвижные (купе)", "en": "Sliding doors"}}]',
 true, 5),

-- Has mirror
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'mirror', 'switch',
 '{"uz": "Oynali", "ru": "С зеркалом", "en": "With Mirror"}',
 NULL, false, 6),

-- Internal organization
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'shelves_count', 'dropdown',
 '{"uz": "Javonlar soni", "ru": "Количество полок", "en": "Number of Shelves"}',
 '[{"value": "3-5", "label": {"uz": "3-5 ta", "ru": "3-5 шт", "en": "3-5"}},
   {"value": "6-8", "label": {"uz": "6-8 ta", "ru": "6-8 шт", "en": "6-8"}},
   {"value": "9+", "label": {"uz": "9+ ta", "ru": "9+ шт", "en": "9+"}}]',
 false, 7),

-- Has hanging rail
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'hanging_rail', 'switch',
 '{"uz": "Kiyim osish joyi", "ru": "Штанга для одежды", "en": "Hanging Rail"}',
 NULL, false, 8),

-- Warranty
('42c20ed3-db22-4268-95f2-1f9c36455f1f', 'warranty', 'dropdown',
 '{"uz": "Kafolat", "ru": "Гарантия", "en": "Warranty"}',
 '[{"value": "1_year", "label": {"uz": "1 yil", "ru": "1 год", "en": "1 year"}},
   {"value": "2_years", "label": {"uz": "2 yil", "ru": "2 года", "en": "2 years"}},
   {"value": "3_years", "label": {"uz": "3 yil", "ru": "3 года", "en": "3 years"}}]',
 false, 9);


-- 5. Bedroom (Yotoqxona) - 4e3dd395-da48-4a85-9498-8806b69cc036
INSERT INTO category_attributes (category_id, key, type, label, options, is_required, sort_order) VALUES
-- Material
('4e3dd395-da48-4a85-9498-8806b69cc036', 'material', 'dropdown', 
 '{"uz": "Material", "ru": "Материал", "en": "Material"}',
 '[{"value": "mdf", "label": {"uz": "MDF", "ru": "МДФ", "en": "MDF"}},
   {"value": "dsp", "label": {"uz": "DSP (ЛДСП)", "ru": "ЛДСП", "en": "Laminated Particleboard"}},
   {"value": "natural_wood", "label": {"uz": "Tabiiy yog''och", "ru": "Натуральное дерево", "en": "Natural Wood"}},
   {"value": "metal", "label": {"uz": "Metall", "ru": "Металл", "en": "Metal"}}]',
 true, 1),

-- Color
('4e3dd395-da48-4a85-9498-8806b69cc036', 'color', 'dropdown',
 '{"uz": "Rang", "ru": "Цвет", "en": "Color"}',
 '[{"value": "white", "label": {"uz": "Oq", "ru": "Белый", "en": "White"}},
   {"value": "black", "label": {"uz": "Qora", "ru": "Чёрный", "en": "Black"}},
   {"value": "brown", "label": {"uz": "Jigarrang", "ru": "Коричневый", "en": "Brown"}},
   {"value": "beige", "label": {"uz": "Bej", "ru": "Бежевый", "en": "Beige"}},
   {"value": "gray", "label": {"uz": "Kulrang", "ru": "Серый", "en": "Gray"}},
   {"value": "oak", "label": {"uz": "Eman", "ru": "Дуб", "en": "Oak"}},
   {"value": "wenge", "label": {"uz": "Venge", "ru": "Венге", "en": "Wenge"}}]',
 true, 2),

-- Bed size
('4e3dd395-da48-4a85-9498-8806b69cc036', 'bed_size', 'dropdown',
 '{"uz": "Karavot o''lchami", "ru": "Размер кровати", "en": "Bed Size"}',
 '[{"value": "90x200", "label": {"uz": "90x200 (1 kishilik)", "ru": "90x200 (односпальная)", "en": "90x200 (Single)"}},
   {"value": "120x200", "label": {"uz": "120x200 (1.5 kishilik)", "ru": "120x200 (полуторная)", "en": "120x200 (Small Double)"}},
   {"value": "140x200", "label": {"uz": "140x200 (2 kishilik)", "ru": "140x200 (двуспальная)", "en": "140x200 (Double)"}},
   {"value": "160x200", "label": {"uz": "160x200 (2 kishilik)", "ru": "160x200 (двуспальная)", "en": "160x200 (Queen)"}},
   {"value": "180x200", "label": {"uz": "180x200 (King)", "ru": "180x200 (King)", "en": "180x200 (King)"}},
   {"value": "200x200", "label": {"uz": "200x200 (Super King)", "ru": "200x200 (Super King)", "en": "200x200 (Super King)"}}]',
 true, 3),

-- Has headboard
('4e3dd395-da48-4a85-9498-8806b69cc036', 'headboard', 'switch',
 '{"uz": "Bosh qismi (izgolov)", "ru": "С изголовьем", "en": "With Headboard"}',
 NULL, false, 4),

-- Headboard type
('4e3dd395-da48-4a85-9498-8806b69cc036', 'headboard_type', 'dropdown',
 '{"uz": "Bosh qismi turi", "ru": "Тип изголовья", "en": "Headboard Type"}',
 '[{"value": "soft", "label": {"uz": "Yumshoq", "ru": "Мягкое", "en": "Soft/Upholstered"}},
   {"value": "hard", "label": {"uz": "Qattiq", "ru": "Жёсткое", "en": "Hard/Wood"}},
   {"value": "none", "label": {"uz": "Yo''q", "ru": "Нет", "en": "None"}}]',
 false, 5),

-- Has storage
('4e3dd395-da48-4a85-9498-8806b69cc036', 'storage', 'switch',
 '{"uz": "Ichki bo''lma (saqlash joyi)", "ru": "С ящиками/подъёмным механизмом", "en": "With Storage"}',
 NULL, false, 6),

-- Lift mechanism
('4e3dd395-da48-4a85-9498-8806b69cc036', 'lift_mechanism', 'switch',
 '{"uz": "Ko''tarma mexanizm", "ru": "Подъёмный механизм", "en": "Lift Mechanism"}',
 NULL, false, 7),

-- Mattress included
('4e3dd395-da48-4a85-9498-8806b69cc036', 'mattress_included', 'switch',
 '{"uz": "Matras bilan", "ru": "С матрасом", "en": "Mattress Included"}',
 NULL, false, 8),

-- Warranty
('4e3dd395-da48-4a85-9498-8806b69cc036', 'warranty', 'dropdown',
 '{"uz": "Kafolat", "ru": "Гарантия", "en": "Warranty"}',
 '[{"value": "1_year", "label": {"uz": "1 yil", "ru": "1 год", "en": "1 year"}},
   {"value": "2_years", "label": {"uz": "2 yil", "ru": "2 года", "en": "2 years"}},
   {"value": "3_years", "label": {"uz": "3 yil", "ru": "3 года", "en": "3 years"}}]',
 false, 9);


-- 6. Kids Furniture (Bolalar mebeli) - bc81b070-80bd-4092-ae55-0d7c397edd05
INSERT INTO category_attributes (category_id, key, type, label, options, is_required, sort_order) VALUES
-- Material
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'material', 'dropdown', 
 '{"uz": "Material", "ru": "Материал", "en": "Material"}',
 '[{"value": "mdf", "label": {"uz": "MDF", "ru": "МДФ", "en": "MDF"}},
   {"value": "dsp", "label": {"uz": "DSP (ЛДСП)", "ru": "ЛДСП", "en": "Laminated Particleboard"}},
   {"value": "natural_wood", "label": {"uz": "Tabiiy yog''och", "ru": "Натуральное дерево", "en": "Natural Wood"}},
   {"value": "plastic", "label": {"uz": "Plastik", "ru": "Пластик", "en": "Plastic"}}]',
 true, 1),

-- Color
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'color', 'dropdown',
 '{"uz": "Rang", "ru": "Цвет", "en": "Color"}',
 '[{"value": "white", "label": {"uz": "Oq", "ru": "Белый", "en": "White"}},
   {"value": "pink", "label": {"uz": "Pushti", "ru": "Розовый", "en": "Pink"}},
   {"value": "blue", "label": {"uz": "Ko''k", "ru": "Голубой", "en": "Blue"}},
   {"value": "green", "label": {"uz": "Yashil", "ru": "Зелёный", "en": "Green"}},
   {"value": "yellow", "label": {"uz": "Sariq", "ru": "Жёлтый", "en": "Yellow"}},
   {"value": "beige", "label": {"uz": "Bej", "ru": "Бежевый", "en": "Beige"}},
   {"value": "oak", "label": {"uz": "Eman", "ru": "Дуб", "en": "Oak"}}]',
 true, 2),

-- Age range
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'age_range', 'dropdown',
 '{"uz": "Yosh chegarasi", "ru": "Возрастная группа", "en": "Age Range"}',
 '[{"value": "0-3", "label": {"uz": "0-3 yosh", "ru": "0-3 года", "en": "0-3 years"}},
   {"value": "3-6", "label": {"uz": "3-6 yosh", "ru": "3-6 лет", "en": "3-6 years"}},
   {"value": "6-12", "label": {"uz": "6-12 yosh", "ru": "6-12 лет", "en": "6-12 years"}},
   {"value": "12+", "label": {"uz": "12+ yosh", "ru": "12+ лет", "en": "12+ years"}}]',
 true, 3),

-- Bed size (for kids beds)
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'bed_size', 'dropdown',
 '{"uz": "Karavot o''lchami", "ru": "Размер кровати", "en": "Bed Size"}',
 '[{"value": "70x140", "label": {"uz": "70x140 sm", "ru": "70x140 см", "en": "70x140 cm"}},
   {"value": "80x160", "label": {"uz": "80x160 sm", "ru": "80x160 см", "en": "80x160 cm"}},
   {"value": "80x180", "label": {"uz": "80x180 sm", "ru": "80x180 см", "en": "80x180 cm"}},
   {"value": "90x190", "label": {"uz": "90x190 sm", "ru": "90x190 см", "en": "90x190 cm"}},
   {"value": "90x200", "label": {"uz": "90x200 sm", "ru": "90x200 см", "en": "90x200 cm"}}]',
 false, 4),

-- Bunk bed
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'bunk_bed', 'switch',
 '{"uz": "Ikki qavatli", "ru": "Двухъярусная", "en": "Bunk Bed"}',
 NULL, false, 5),

-- Height adjustable
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'height_adjustable', 'switch',
 '{"uz": "Balandligi sozlanadi", "ru": "Регулируемая высота", "en": "Height Adjustable"}',
 NULL, false, 6),

-- Safety rails
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'safety_rails', 'switch',
 '{"uz": "Himoya to''sig''i", "ru": "Защитные бортики", "en": "Safety Rails"}',
 NULL, false, 7),

-- Safety certified
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'safety_certified', 'switch',
 '{"uz": "Xavfsizlik sertifikati", "ru": "Сертификат безопасности", "en": "Safety Certified"}',
 NULL, false, 8),

-- Eco-friendly
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'eco_friendly', 'switch',
 '{"uz": "Ekologik toza", "ru": "Экологичный", "en": "Eco-friendly"}',
 NULL, false, 9),

-- Warranty
('bc81b070-80bd-4092-ae55-0d7c397edd05', 'warranty', 'dropdown',
 '{"uz": "Kafolat", "ru": "Гарантия", "en": "Warranty"}',
 '[{"value": "1_year", "label": {"uz": "1 yil", "ru": "1 год", "en": "1 year"}},
   {"value": "2_years", "label": {"uz": "2 yil", "ru": "2 года", "en": "2 years"}},
   {"value": "3_years", "label": {"uz": "3 yil", "ru": "3 года", "en": "3 years"}}]',
 false, 10);


-- ============================================
-- VERIFICATION QUERY
-- ============================================
-- Run this to verify:
-- SELECT c.name->>'uz' as category, COUNT(ca.id) as attributes_count
-- FROM categories c
-- LEFT JOIN category_attributes ca ON c.id = ca.category_id
-- GROUP BY c.id, c.name
-- ORDER BY c.name->>'uz';
