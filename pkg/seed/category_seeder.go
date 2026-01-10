package seed

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// CategoryIDs - kategoriya ID lari (products seeder uchun)
type CategoryIDs struct {
	// Yashash xonasi
	LivingRoom      string
	Sofas           string
	Armchairs       string
	CoffeeTables    string
	TVStands        string

	// Yotoqxona
	Bedroom         string
	Beds            string
	Mattresses      string
	Wardrobes       string
	Dressers        string

	// Oshxona
	Kitchen         string
	DiningSets      string
	DiningTables    string
	DiningChairs    string

	// Ofis
	Office          string
	OfficeChairs    string
	OfficeDesks     string
	Shelves         string

	// Bolalar
	Kids            string
	KidsBeds        string
	KidsDesks       string

	// Koridor
	Hallway         string
	ShoeRacks       string
	Hangers         string
}

// SeedCategories - kategoriyalarni yaratadi va ID larini qaytaradi
func SeedCategories(db *sql.DB) *CategoryIDs {
	// 1. Categories jadvalini yaratish
	createCategoriesTable(db)

	// 2. Kategoriyalar borligini tekshirish
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		log.Printf("Categories count xatosi: %v", err)
		return nil
	}

	if count > 0 {
		fmt.Printf("✅ Categories jadvalida %d ta kategoriya mavjud\n", count)
		return loadExistingCategoryIDs(db)
	}

	// 3. Kategoriyalarni qo'shish
	return seedCategoryData(db)
}

// createCategoriesTable - categories jadvalini yaratadi
func createCategoriesTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS categories (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		icon_url VARCHAR(500),
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Categories jadval yaratishda xatolik: %v", err)
	} else {
		fmt.Println("✅ Categories jadvali tayyor!")
	}
}

// loadExistingCategoryIDs - mavjud kategoriya ID larini yuklaydi
func loadExistingCategoryIDs(db *sql.DB) *CategoryIDs {
	ids := &CategoryIDs{}

	// Kategoriya nomlariga ko'ra ID larni olish
	categoryMap := map[string]*string{
		// Yashash xonasi
		"Yashash xonasi":  &ids.LivingRoom,
		"Divanlar":        &ids.Sofas,
		"Kresollar":       &ids.Armchairs,
		"Kofe stollari":   &ids.CoffeeTables,
		"TV javonlari":    &ids.TVStands,
		// Yotoqxona
		"Yotoqxona":       &ids.Bedroom,
		"Karavotlar":      &ids.Beds,
		"Matraslar":       &ids.Mattresses,
		"Shkaflar":        &ids.Wardrobes,
		"Tumbalar":        &ids.Dressers,
		// Oshxona
		"Oshxona":         &ids.Kitchen,
		"Oshxona to'plami": &ids.DiningSets,
		"Oshxona stollari": &ids.DiningTables,
		"Stullar":         &ids.DiningChairs,
		// Ofis
		"Ofis":            &ids.Office,
		"Ofis kreslosari": &ids.OfficeChairs,
		"Ofis stollari":   &ids.OfficeDesks,
		"Javonlar":        &ids.Shelves,
		// Bolalar
		"Bolalar":         &ids.Kids,
		"Bolalar karavoti": &ids.KidsBeds,
		"Bolalar stoli":   &ids.KidsDesks,
		// Koridor
		"Koridor":         &ids.Hallway,
		"Poyabzal javoni": &ids.ShoeRacks,
		"Ilgichlar":       &ids.Hangers,
	}

	for name, idPtr := range categoryMap {
		var id string
		err := db.QueryRow("SELECT id FROM categories WHERE name = $1", name).Scan(&id)
		if err == nil {
			*idPtr = id
		}
	}

	return ids
}

// seedCategoryData - kategoriyalarni seed qiladi
func seedCategoryData(db *sql.DB) *CategoryIDs {
	fmt.Println("🌱 Kategoriyalar qo'shilmoqda...")

	ids := &CategoryIDs{}

	// ============================================
	// 1. YASHASH XONASI
	// ============================================
	ids.LivingRoom = insertCategory(db, nil, "Yashash xonasi", "https://img.icons8.com/fluency/96/living-room.png")
	ids.Sofas = insertCategory(db, &ids.LivingRoom, "Divanlar", "https://img.icons8.com/fluency/96/sofa.png")
	ids.Armchairs = insertCategory(db, &ids.LivingRoom, "Kresollar", "https://img.icons8.com/fluency/96/armchair.png")
	ids.CoffeeTables = insertCategory(db, &ids.LivingRoom, "Kofe stollari", "https://img.icons8.com/fluency/96/coffee-table.png")
	ids.TVStands = insertCategory(db, &ids.LivingRoom, "TV javonlari", "https://img.icons8.com/fluency/96/tv-stand.png")

	// ============================================
	// 2. YOTOQXONA
	// ============================================
	ids.Bedroom = insertCategory(db, nil, "Yotoqxona", "https://img.icons8.com/fluency/96/bedroom.png")
	ids.Beds = insertCategory(db, &ids.Bedroom, "Karavotlar", "https://img.icons8.com/fluency/96/bed.png")
	ids.Mattresses = insertCategory(db, &ids.Bedroom, "Matraslar", "https://img.icons8.com/fluency/96/mattress.png")
	ids.Wardrobes = insertCategory(db, &ids.Bedroom, "Shkaflar", "https://img.icons8.com/fluency/96/wardrobe.png")
	ids.Dressers = insertCategory(db, &ids.Bedroom, "Tumbalar", "https://img.icons8.com/fluency/96/dresser.png")

	// ============================================
	// 3. OSHXONA
	// ============================================
	ids.Kitchen = insertCategory(db, nil, "Oshxona", "https://img.icons8.com/fluency/96/kitchen-room.png")
	ids.DiningSets = insertCategory(db, &ids.Kitchen, "Oshxona to'plami", "https://img.icons8.com/fluency/96/dining-room.png")
	ids.DiningTables = insertCategory(db, &ids.Kitchen, "Oshxona stollari", "https://img.icons8.com/fluency/96/dining-table.png")
	ids.DiningChairs = insertCategory(db, &ids.Kitchen, "Stullar", "https://img.icons8.com/fluency/96/chair.png")

	// ============================================
	// 4. OFIS
	// ============================================
	ids.Office = insertCategory(db, nil, "Ofis", "https://img.icons8.com/fluency/96/office.png")
	ids.OfficeChairs = insertCategory(db, &ids.Office, "Ofis kreslosari", "https://img.icons8.com/fluency/96/office-chair.png")
	ids.OfficeDesks = insertCategory(db, &ids.Office, "Ofis stollari", "https://img.icons8.com/fluency/96/desk.png")
	ids.Shelves = insertCategory(db, &ids.Office, "Javonlar", "https://img.icons8.com/fluency/96/bookshelf.png")

	// ============================================
	// 5. BOLALAR
	// ============================================
	ids.Kids = insertCategory(db, nil, "Bolalar", "https://img.icons8.com/fluency/96/cradle.png")
	ids.KidsBeds = insertCategory(db, &ids.Kids, "Bolalar karavoti", "https://img.icons8.com/fluency/96/kids-bed.png")
	ids.KidsDesks = insertCategory(db, &ids.Kids, "Bolalar stoli", "https://img.icons8.com/fluency/96/kids-desk.png")

	// ============================================
	// 6. KORIDOR
	// ============================================
	ids.Hallway = insertCategory(db, nil, "Koridor", "https://img.icons8.com/fluency/96/coat-rack.png")
	ids.ShoeRacks = insertCategory(db, &ids.Hallway, "Poyabzal javoni", "https://img.icons8.com/fluency/96/shoe-rack.png")
	ids.Hangers = insertCategory(db, &ids.Hallway, "Ilgichlar", "https://img.icons8.com/fluency/96/hanger.png")

	fmt.Println("✅ 6 ta asosiy va 20 ta sub-kategoriya qo'shildi!")

	return ids
}

// insertCategory - bitta kategoriya qo'shadi
func insertCategory(db *sql.DB, parentID *string, name, iconURL string) string {
	id := uuid.New().String()

	query := `INSERT INTO categories (id, parent_id, name, icon_url) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(query, id, parentID, name, iconURL)
	if err != nil {
		log.Printf("   ❌ %s qo'shishda xatolik: %v", name, err)
		return ""
	}

	if parentID == nil {
		log.Printf("   📁 %s", name)
	} else {
		log.Printf("      └── %s", name)
	}

	return id
}
