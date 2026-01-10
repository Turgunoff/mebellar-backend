package seed

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// SeedAll - barcha seederlarni ishga tushiradi
func SeedAll(db *sql.DB) {
	fmt.Println("\n🌱 ======= SEEDING STARTED =======")

	// 1. Kategoriyalarni seed qilish (birinchi!)
	categoryIDs := SeedCategories(db)

	// 2. Mahsulotlarni seed qilish (kategoriya ID lari bilan)
	SeedProducts(db, categoryIDs)

	fmt.Println("🌱 ======= SEEDING COMPLETED =======\n")
}

// SeedProducts - Products jadvalini yaratadi va namuna mahsulotlar bilan to'ldiradi
func SeedProducts(db *sql.DB, catIDs *CategoryIDs) {
	// 1. Products jadvalini yaratish (yangi schema)
	createProductsTable(db)

	// 2. Mahsulotlar borligini tekshirish
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		log.Printf("Products count xatosi: %v", err)
		return
	}

	if count > 0 {
		fmt.Printf("✅ Products jadvalida %d ta mahsulot mavjud\n", count)
		return
	}

	// 3. Namuna mahsulotlarni qo'shish
	seedSampleProducts(db, catIDs)
}

// createProductsTable - products jadvalini yaratadi (MVP uchun moslashuvchan)
func createProductsTable(db *sql.DB) {
	// FK constraint qo'shish uchun categories jadvalidan keyin yaratiladi
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		price DECIMAL(15, 2) NOT NULL,
		discount_price DECIMAL(15, 2),
		images TEXT[] DEFAULT '{}',
		specs JSONB DEFAULT '{}',
		variants JSONB DEFAULT '[]',
		rating DECIMAL(2, 1) DEFAULT 4.5,
		is_new BOOLEAN DEFAULT true,
		is_popular BOOLEAN DEFAULT false,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id);
	CREATE INDEX IF NOT EXISTS idx_products_is_new ON products(is_new);
	CREATE INDEX IF NOT EXISTS idx_products_is_popular ON products(is_popular);
	CREATE INDEX IF NOT EXISTS idx_products_is_active ON products(is_active);
	CREATE INDEX IF NOT EXISTS idx_products_specs ON products USING GIN(specs);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Products jadval yaratishda xatolik: %v", err)
	} else {
		fmt.Println("✅ Products jadvali tayyor (UUID + JSONB + FK)!")
	}
}

// seedSampleProducts - yuqori sifatli namuna mahsulotlar
func seedSampleProducts(db *sql.DB, catIDs *CategoryIDs) {
	fmt.Println("🌱 Yuqori sifatli namuna mahsulotlar qo'shilmoqda...")

	type ProductSeed struct {
		Name          string
		Description   string
		Price         float64
		DiscountPrice *float64
		CategoryID    string
		Images        string // PostgreSQL array format
		Specs         string // JSONB format
		Variants      string // JSONB array format
		Rating        float64
		IsNew         bool
		IsPopular     bool
	}

	// Chegirmali narxlar uchun helper
	discount := func(price float64) *float64 { return &price }

	// Kategoriya ID larini olish (nil bo'lsa bo'sh string)
	getCatID := func(id string) string {
		if id == "" {
			return ""
		}
		return id
	}

	products := []ProductSeed{
		// ============================================
		// DIVANLAR
		// ============================================
		{
			Name:          "Premium L-shaklidagi Divan \"Milano\"",
			Description:   "Zamonaviy italyan dizaynidagi hashamatli burchak divani. Yumshoq velvet qoplama, mustahkam yog'och ramka. 5 kishilik sig'im. Yotish funksiyasi mavjud.",
			Price:         8500000,
			DiscountPrice: discount(6800000),
			CategoryID:    getCatID(catIDs.Sofas),
			Images:        `{"https://images.unsplash.com/photo-1555041469-a586c61ea9bc?w=800","https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?w=800"}`,
			Specs:         `{"Material": "Velvet", "Ramka": "Eman yog'ochi", "O'lcham": "280x180x85 sm", "Ishlab chiqaruvchi": "Italiya dizayni", "Kafolat": "2 yil"}`,
			Variants:      `[{"color": "Kulrang", "colorCode": "6B6B6B", "stock": 5}, {"color": "Yashil", "colorCode": "2D5A3D", "stock": 3}]`,
			Rating:        4.9,
			IsNew:         true,
			IsPopular:     true,
		},
		{
			Name:          "Ikki kishilik Divan \"Nordic\"",
			Description:   "Skandinav uslubidagi zamonaviy divan. Yengil va chiroyli dizayn. Premium mato qoplamasi.",
			Price:         4200000,
			DiscountPrice: nil,
			CategoryID:    getCatID(catIDs.Sofas),
			Images:        `{"https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?w=800"}`,
			Specs:         `{"Material": "Premium mato", "O'lcham": "180x90x85 sm", "Uslub": "Skandinav"}`,
			Variants:      `[{"color": "Oq", "colorCode": "FFFAF0", "stock": 8}]`,
			Rating:        4.6,
			IsNew:         true,
			IsPopular:     false,
		},
		// ============================================
		// KARAVOTLAR
		// ============================================
		{
			Name:          "Klassik Karavot \"Royal\" 180x200",
			Description:   "Hashamatli klassik uslubdagi karavot. Qo'lda ishlangan o'ymakorlik elementlari. Premium sifatli eman yog'ochidan yasalgan. Matras alohida sotiladi.",
			Price:         6200000,
			DiscountPrice: nil,
			CategoryID:    getCatID(catIDs.Beds),
			Images:        `{"https://images.unsplash.com/photo-1617325247661-675ab4b64ae2?w=800","https://images.unsplash.com/photo-1505693416388-ac5ce068fe85?w=800"}`,
			Specs:         `{"Material": "Eman yog'ochi", "O'lcham": "180x200 sm", "Balandlik": "120 sm", "Ishlab chiqaruvchi": "O'zbekiston", "Uslub": "Klassik"}`,
			Variants:      `[{"color": "Jigarrang", "colorCode": "633E33", "stock": 8}, {"color": "Oq", "colorCode": "F5F5DC", "stock": 4}]`,
			Rating:        4.8,
			IsNew:         true,
			IsPopular:     false,
		},
		{
			Name:          "Zamonaviy Karavot \"Comfort\" 160x200",
			Description:   "Minimalist dizayndagi zamonaviy karavot. Yumshoq bosh qismi va mustahkam ramka.",
			Price:         4500000,
			DiscountPrice: discount(3600000),
			CategoryID:    getCatID(catIDs.Beds),
			Images:        `{"https://images.unsplash.com/photo-1505693416388-ac5ce068fe85?w=800"}`,
			Specs:         `{"Material": "MDF + Teri", "O'lcham": "160x200 sm", "Uslub": "Zamonaviy"}`,
			Variants:      `[{"color": "Qora", "colorCode": "1E1E20", "stock": 6}]`,
			Rating:        4.7,
			IsNew:         false,
			IsPopular:     true,
		},
		// ============================================
		// KOFE STOLLARI
		// ============================================
		{
			Name:          "Marmar Kofe Stoli \"Elegance\"",
			Description:   "Tabiiy marmar ustki qismi va oltin rangli metall oyoqlari. Zamonaviy minimalist dizayn. Yashash xonangizga nafislik qo'shadi.",
			Price:         2400000,
			DiscountPrice: discount(1920000),
			CategoryID:    getCatID(catIDs.CoffeeTables),
			Images:        `{"https://images.unsplash.com/photo-1533090481720-856c6e3c1fdc?w=800","https://images.unsplash.com/photo-1611967164521-abae8fba4668?w=800"}`,
			Specs:         `{"Material": "Tabiiy marmar + Metall", "Diametr": "80 sm", "Balandlik": "45 sm", "Og'irlik": "25 kg", "Ishlab chiqaruvchi": "Turkiya"}`,
			Variants:      `[{"color": "Oq marmar", "colorCode": "FFFFFF", "stock": 12}, {"color": "Qora marmar", "colorCode": "1E1E20", "stock": 6}]`,
			Rating:        4.7,
			IsNew:         false,
			IsPopular:     true,
		},
		// ============================================
		// OFIS KRESLOSARI
		// ============================================
		{
			Name:          "Ergonomik Ofis Kreslosi \"ProSit\"",
			Description:   "To'liq sozlanishi mumkin professional ofis kreslosi. Bel qismi va bo'yin qismi alohida qo'llab-quvvatlaydi. 8+ soatlik ishlash uchun ideal.",
			Price:         3800000,
			DiscountPrice: nil,
			CategoryID:    getCatID(catIDs.OfficeChairs),
			Images:        `{"https://images.unsplash.com/photo-1580480055273-228ff5388ef8?w=800","https://images.unsplash.com/photo-1589384267710-7a170981ca78?w=800"}`,
			Specs:         `{"Material": "Premium mesh + Plastik", "Sozlamalar": "Bel, qo'ltiq, balandlik", "Yuk sig'imi": "150 kg", "G'ildirak": "360° aylanadi", "Kafolat": "5 yil"}`,
			Variants:      `[{"color": "Qora", "colorCode": "1E1E20", "stock": 20}, {"color": "Kulrang", "colorCode": "6B6B6B", "stock": 15}]`,
			Rating:        4.9,
			IsNew:         true,
			IsPopular:     true,
		},
		// ============================================
		// OSHXONA TO'PLAMI
		// ============================================
		{
			Name:          "Oshxona To'plami \"Family\" (Stol + 6 Stul)",
			Description:   "Oila uchun ideal oshxona to'plami. Kengaytirilishi mumkin stol (160-200 sm). 6 ta qulay stul. Chidamli materialdan yasalgan.",
			Price:         4200000,
			DiscountPrice: discount(3570000),
			CategoryID:    getCatID(catIDs.DiningSets),
			Images:        `{"https://images.unsplash.com/photo-1617806118233-18e1de247200?w=800","https://images.unsplash.com/photo-1615066390971-03e4e1c36ddf?w=800"}`,
			Specs:         `{"Material": "MDF + Yog'och oyoqlar", "Stol o'lchami": "160-200x90 sm", "Stullar soni": "6 dona", "Rang": "Oq + Yog'och", "Ishlab chiqaruvchi": "O'zbekiston"}`,
			Variants:      `[{"color": "Oq", "colorCode": "FFFFFF", "stock": 8}, {"color": "Kulrang", "colorCode": "E5E5E5", "stock": 5}]`,
			Rating:        4.6,
			IsNew:         false,
			IsPopular:     true,
		},
		// ============================================
		// SHKAFLAR
		// ============================================
		{
			Name:          "Ko'zguyli Shkaf \"Elegance\" 4 eshikli",
			Description:   "Zamonaviy 4 eshikli shkaf. 2 ta katta ko'zgu. Ko'p xonali ichki tuzilishi. Kiyim, ko'rpalar va aksessuarlar uchun ideal.",
			Price:         5500000,
			DiscountPrice: nil,
			CategoryID:    getCatID(catIDs.Wardrobes),
			Images:        `{"https://images.unsplash.com/photo-1558997519-83ea9252edf8?w=800","https://images.unsplash.com/photo-1595428774223-ef52624120d2?w=800"}`,
			Specs:         `{"Material": "Laminat DSP", "O'lcham": "200x240x60 sm", "Eshiklar": "4 ta", "Ko'zgu": "2 ta katta", "Ichki bo'limlar": "12+"}`,
			Variants:      `[{"color": "Oq", "colorCode": "FFFFFF", "stock": 6}, {"color": "Jigarrang", "colorCode": "633E33", "stock": 4}]`,
			Rating:        4.5,
			IsNew:         true,
			IsPopular:     false,
		},
		// ============================================
		// KRESOLLAR
		// ============================================
		{
			Name:          "Kreslo \"Vintage\"",
			Description:   "Retro uslubidagi qulay kreslo. Yumshoq to'ldirma va mustahkam yog'och ramkasi. Dam olish uchun ideal.",
			Price:         2400000,
			DiscountPrice: nil,
			CategoryID:    getCatID(catIDs.Armchairs),
			Images:        `{"https://images.unsplash.com/photo-1586023492125-27b2c045efd7?w=800"}`,
			Specs:         `{"Material": "Velvet + Yog'och", "Uslub": "Retro/Vintage", "Yuk sig'imi": "120 kg"}`,
			Variants:      `[{"color": "Yashil", "colorCode": "2D5A3D", "stock": 4}, {"color": "Sariq", "colorCode": "F4A460", "stock": 3}]`,
			Rating:        4.7,
			IsNew:         true,
			IsPopular:     false,
		},
		// ============================================
		// OFIS STOLLARI
		// ============================================
		{
			Name:          "Ofis stoli \"Executive\"",
			Description:   "Professional ofis stoli. Ko'p tortmali va kabel boshqaruvi. Keng ish maydoni.",
			Price:         2800000,
			DiscountPrice: nil,
			CategoryID:    getCatID(catIDs.OfficeDesks),
			Images:        `{"https://images.unsplash.com/photo-1518455027359-f3f8164ba6bd?w=800"}`,
			Specs:         `{"Material": "MDF + Metall", "O'lcham": "160x80x75 sm", "Tortmalar": "3 ta"}`,
			Variants:      `[{"color": "Jigarrang", "colorCode": "633E33", "stock": 10}]`,
			Rating:        4.6,
			IsNew:         false,
			IsPopular:     false,
		},
	}

	// Insert query
	query := `
		INSERT INTO products (id, category_id, name, description, price, discount_price, images, specs, variants, rating, is_new, is_popular)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7::text[], $8::jsonb, $9::jsonb, $10, $11, $12)
	`

	successCount := 0
	for _, p := range products {
		id := uuid.New().String()
		_, err := db.Exec(query, id, p.CategoryID, p.Name, p.Description, p.Price, p.DiscountPrice, p.Images, p.Specs, p.Variants, p.Rating, p.IsNew, p.IsPopular)
		if err != nil {
			log.Printf("❌ Mahsulot qo'shishda xatolik (%s): %v", p.Name, err)
		} else {
			successCount++
			log.Printf("   ✓ %s", p.Name)
		}
	}

	fmt.Printf("✅ %d ta yuqori sifatli mahsulot qo'shildi!\n", successCount)
}
