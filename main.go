package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"mebellar-backend/handlers"
	"mebellar-backend/pkg/sms"
	"mebellar-backend/pkg/websocket"

	_ "mebellar-backend/docs" // Swagger docs - swag init dan keyin paydo bo'ladi

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"
)

// getEnv - environment variable olish (default bilan)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// @title           Mebellar Olami API
// @version         1.0
// @description     Bu Flutter ilovasi uchun Backend API serveri. Mebel sotish platformasi.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@mebellar.uz

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      45.93.201.167:8081
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token kiritish: "Bearer {token}"

// userMeHandler - /api/user/me uchun method router
func userMeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetProfile(db)(w, r)
		case http.MethodPut:
			handlers.UpdateProfile(db)(w, r)
		case http.MethodDelete:
			handlers.DeleteAccount(db)(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"success":false,"message":"Bu metod qo'llab-quvvatlanmaydi"}`))
		}
	}
}

// CORS middleware - barcha so'rovlarga CORS headerlarini qo'shadi
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// OPTIONS so'rovlarini darhol qaytarish (preflight)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log incoming request
		log.Printf("📥 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next(w, r)
	}
}

func main() {
	// .env faylini yuklash
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env fayli topilmadi, environment variablelardan foydalaniladi")
	} else {
		fmt.Println("✅ .env fayli yuklandi")
	}

	// Environment variablelarni o'qish
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "mebel_user")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "mebellar_olami")
	serverPort := getEnv("SERVER_PORT", "8081")
	jwtSecret := getEnv("JWT_SECRET", "mebellar-super-secret-key-2024")

	// JWT secretni handlers ga uzatish
	handlers.SetJWTSecret(jwtSecret)

	// WebSocket JWT secretni uzatish
	websocket.SetJWTSecret(jwtSecret)

	// 1. Bazaga ulanish
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("❌ Database connection error: ", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("❌ Database ping error: ", err)
	}
	fmt.Printf("✅ Baza ulangan! (%s@%s:%s/%s)\n", dbUser, dbHost, dbPort, dbName)

	// Users jadvalini yaratish (agar mavjud bo'lmasa)
	createUsersTable(db)

	// SMS Service (Eskiz.uz) sozlash
	initSMSService()

	// WebSocket Hub ishga tushirish
	websocket.InitGlobalHub()

	// 2. Marshrutlar (Routes) - CORS middleware bilan
	// Kategoriyalar
	http.HandleFunc("/api/categories", corsMiddleware(handlers.GetCategories(db)))
	http.HandleFunc("/api/categories/", corsMiddleware(handlers.GetCategoryByID(db))) // /api/categories/{id}

	// Hududlar (Viloyatlar)
	http.HandleFunc("/api/regions", corsMiddleware(handlers.GetRegions(db)))

	// Mahsulotlar
	http.HandleFunc("/api/products", corsMiddleware(handlers.GetProducts(db)))
	http.HandleFunc("/api/products/new", corsMiddleware(handlers.GetNewArrivals(db)))
	http.HandleFunc("/api/products/popular", corsMiddleware(handlers.GetPopularProducts(db)))
	http.HandleFunc("/api/products/", corsMiddleware(handlers.GetProductByID(db))) // /api/products/{id}

	// Autentifikatsiya endpointlari
	http.HandleFunc("/api/auth/send-otp", corsMiddleware(handlers.SendOTP(db)))
	http.HandleFunc("/api/auth/verify-otp", corsMiddleware(handlers.VerifyOTP(db)))
	http.HandleFunc("/api/auth/register", corsMiddleware(handlers.Register(db)))
	http.HandleFunc("/api/auth/login", corsMiddleware(handlers.Login(db)))
	http.HandleFunc("/api/auth/forgot-password", corsMiddleware(handlers.ForgotPassword(db)))
	http.HandleFunc("/api/auth/reset-password", corsMiddleware(handlers.ResetPassword(db)))

	// User profile endpointlari (JWT himoyalangan)
	http.HandleFunc("/api/user/me", corsMiddleware(handlers.JWTMiddleware(db, userMeHandler(db))))

	// Telefon o'zgartirish (JWT himoyalangan)
	http.HandleFunc("/api/user/change-phone/request", corsMiddleware(handlers.JWTMiddleware(db, handlers.RequestPhoneChange(db))))
	http.HandleFunc("/api/user/change-phone/verify", corsMiddleware(handlers.JWTMiddleware(db, handlers.VerifyPhoneChange(db))))

	// Email o'zgartirish (JWT himoyalangan)
	http.HandleFunc("/api/user/change-email/request", corsMiddleware(handlers.JWTMiddleware(db, handlers.RequestEmailChange(db))))
	http.HandleFunc("/api/user/change-email/verify", corsMiddleware(handlers.JWTMiddleware(db, handlers.VerifyEmailChange(db))))

	// Sotuvchi bo'lish (JWT himoyalangan)
	http.HandleFunc("/api/user/become-seller", corsMiddleware(handlers.JWTMiddleware(db, handlers.BecomeSeller(db))))

	// ============================================
	// SELLER SHOP ENDPOINTS (Multi-Shop Architecture)
	// ============================================
	// Do'konlar ro'yxati va yaratish
	http.HandleFunc("/api/seller/shops", corsMiddleware(handlers.JWTMiddleware(db, handlers.ShopsHandler(db))))
	// Do'kon bo'yicha operatsiyalar (GET, PUT, DELETE)
	http.HandleFunc("/api/seller/shops/", corsMiddleware(handlers.JWTMiddleware(db, handlers.ShopByIDHandler(db))))

	// Seller mahsulotlari (GET: ro'yxat, POST: yaratish)
	http.HandleFunc("/api/seller/products", corsMiddleware(handlers.JWTMiddleware(db, handlers.SellerProductsHandler(db))))

	// Seller mahsulot item (PUT: yangilash, DELETE: o'chirish)
	http.HandleFunc("/api/seller/products/", corsMiddleware(handlers.JWTMiddleware(db, handlers.SellerProductItemHandler(db))))

	// ============================================
	// CUSTOMER ORDERS ENDPOINTS
	// ============================================
	// GET /api/orders - Mijoz buyurtmalari ro'yxati (JWT talab qiladi)
	// POST /api/orders - Yangi buyurtma yaratish (Public)
	http.HandleFunc("/api/orders", corsMiddleware(handlers.CustomerOrdersHandler(db)))

	// ============================================
	// SELLER ORDERS ENDPOINTS
	// ============================================
	// Buyurtmalar ro'yxati
	http.HandleFunc("/api/seller/orders", corsMiddleware(handlers.JWTMiddleware(db, handlers.SellerOrdersHandler(db))))
	// Buyurtmalar statistikasi
	http.HandleFunc("/api/seller/orders/stats", corsMiddleware(handlers.JWTMiddleware(db, handlers.GetOrderStats(db))))
	// Buyurtma statusini o'zgartirish
	http.HandleFunc("/api/seller/orders/", corsMiddleware(handlers.JWTMiddleware(db, handlers.UpdateOrderStatus(db))))

	// ============================================
	// SELLER PROFILE ENDPOINT
	// ============================================
	// Aggregated seller profile (GET + PUT)
	http.HandleFunc("/api/seller/profile", corsMiddleware(handlers.JWTMiddleware(db, handlers.SellerProfileHandler(db))))
	// Delete account (soft delete)
	http.HandleFunc("/api/seller/account", corsMiddleware(handlers.JWTMiddleware(db, handlers.DeleteSellerAccount(db))))

	// ============================================
	// SELLER DASHBOARD ENDPOINTS
	// ============================================
	// Dashboard statistikasi (Aggregation)
	http.HandleFunc("/api/seller/dashboard/stats", corsMiddleware(handlers.JWTMiddleware(db, handlers.GetDashboardStats(db))))

	// ============================================
	// SELLER ANALYTICS ENDPOINTS
	// ============================================
	// Bekor qilish statistikasi (Cancellation Analytics)
	http.HandleFunc("/api/seller/analytics/cancellations", corsMiddleware(handlers.JWTMiddleware(db, handlers.GetCancellationStats(db))))

	// ============================================
	// COMMON ENDPOINTS (Umumiy)
	// ============================================
	// Bekor qilish sabablari (dinamik)
	http.HandleFunc("/api/common/cancellation-reasons", corsMiddleware(handlers.GetCancellationReasons(db)))

	// ============================================
	// DEBUG ENDPOINTS
	// ============================================
	// Test buyurtmalarini yaratish
	http.HandleFunc("/api/debug/seed-orders", corsMiddleware(handlers.JWTMiddleware(db, handlers.SeedOrders(db))))

	// Ommaviy do'kon sahifasi (slug bo'yicha)
	http.HandleFunc("/api/shops/", corsMiddleware(handlers.GetPublicShopBySlug(db)))

	// ============================================
	// WEBSOCKET ENDPOINT (Real-time)
	// ============================================
	// WebSocket ulanish (JWT + shop_id orqali)
	http.HandleFunc("/ws/orders", websocket.HandleWebSocket(db))

	// 3. Static files - uploads papkasini serve qilish
	fs := http.FileServer(http.Dir("uploads"))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// 4. Swagger UI
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// 5. Serverni yoqish
	fmt.Println("🚀 Server 8081-portda ishlayapti...")
	fmt.Println("")
	fmt.Println("📂 Categories endpoints:")
	fmt.Println("   GET /api/categories       - Barcha kategoriyalar (daraxt)")
	fmt.Println("   GET /api/categories?flat=true - Tekis ro'yxat")
	fmt.Println("   GET /api/categories/{id}  - Bitta kategoriya")
	fmt.Println("")
	fmt.Println("🛋️ Products endpoints:")
	fmt.Println("   GET /api/products         - Barcha mahsulotlar (?category_id=...)")
	fmt.Println("   GET /api/products/new     - Yangi mahsulotlar")
	fmt.Println("   GET /api/products/popular - Mashhur mahsulotlar")
	fmt.Println("   GET /api/products/{id}    - Bitta mahsulot")
	fmt.Println("")
	fmt.Println("📱 Auth endpoints:")
	fmt.Println("   POST /api/auth/send-otp")
	fmt.Println("   POST /api/auth/verify-otp")
	fmt.Println("   POST /api/auth/register")
	fmt.Println("   POST /api/auth/login")
	fmt.Println("   POST /api/auth/forgot-password")
	fmt.Println("   POST /api/auth/reset-password")
	fmt.Println("")
	fmt.Println("👤 User endpoints (JWT himoyalangan):")
	fmt.Println("   GET    /api/user/me - Profilni olish")
	fmt.Println("   PUT    /api/user/me - Profilni yangilash (multipart)")
	fmt.Println("   DELETE /api/user/me - Hisobni o'chirish")
	fmt.Println("")
	fmt.Println("📞 Telefon/Email o'zgartirish (JWT himoyalangan):")
	fmt.Println("   POST /api/user/change-phone/request")
	fmt.Println("   POST /api/user/change-phone/verify")
	fmt.Println("   POST /api/user/change-email/request")
	fmt.Println("   POST /api/user/change-email/verify")
	fmt.Println("")
	fmt.Println("🏪 Sotuvchi bo'lish (JWT himoyalangan):")
	fmt.Println("   POST /api/user/become-seller")
	fmt.Println("")
	fmt.Println("🏬 Seller Shops (Multi-Shop, JWT himoyalangan):")
	fmt.Println("   GET    /api/seller/shops      - Mening do'konlarim")
	fmt.Println("   POST   /api/seller/shops      - Yangi do'kon yaratish")
	fmt.Println("   GET    /api/seller/shops/{id} - Do'kon ma'lumotlari")
	fmt.Println("   PUT    /api/seller/shops/{id} - Do'konni yangilash")
	fmt.Println("   DELETE /api/seller/shops/{id} - Do'konni o'chirish")
	fmt.Println("")
	fmt.Println("🛒 Orders (Customer App):")
	fmt.Println("   GET  /api/orders - Mijoz buyurtmalari ro'yxati (JWT talab qiladi)")
	fmt.Println("   POST /api/orders - Yangi buyurtma yaratish (Ommaviy, WebSocket broadcast)")
	fmt.Println("")
	fmt.Println("📦 Seller Orders (JWT himoyalangan):")
	fmt.Println("   GET  /api/seller/orders        - Buyurtmalar ro'yxati (?status=new)")
	fmt.Println("   GET  /api/seller/orders/stats  - Buyurtmalar statistikasi")
	fmt.Println("   PUT  /api/seller/orders/{id}/status?status=confirmed - Status o'zgartirish")
	fmt.Println("")
	fmt.Println("👤 Seller Profile (JWT himoyalangan):")
	fmt.Println("   GET    /api/seller/profile - Aggregated profile (user + shop stats)")
	fmt.Println("   PUT    /api/seller/profile - Update profile (name, password)")
	fmt.Println("   DELETE /api/seller/account - Soft delete account")
	fmt.Println("")
	fmt.Println("🏠 Seller Dashboard (JWT himoyalangan):")
	fmt.Println("   GET /api/seller/dashboard/stats - Dashboard statistikasi")
	fmt.Println("")
	fmt.Println("📊 Seller Analytics (JWT himoyalangan):")
	fmt.Println("   GET /api/seller/analytics/cancellations - Bekor qilish tahlili")
	fmt.Println("")
	fmt.Println("📋 Common endpoints (Ommaviy):")
	fmt.Println("   GET /api/common/cancellation-reasons - Bekor qilish sabablari")
	fmt.Println("")
	fmt.Println("🔧 Debug endpoints (JWT himoyalangan):")
	fmt.Println("   POST /api/debug/seed-orders?count=10 - Test buyurtmalar yaratish")
	fmt.Println("")
	fmt.Println("🌐 Public Shop (Ommaviy):")
	fmt.Println("   GET /api/shops/{slug} - Do'kon sahifasi")
	fmt.Println("")
	fmt.Println("🔌 WebSocket (Real-time):")
	fmt.Println("   WS /ws/orders?token=JWT&shop_id=UUID - Real-time buyurtmalar")
	fmt.Println("")
	fmt.Println("📁 Static files: /uploads/*")
	fmt.Printf("📚 Swagger UI: http://localhost:%s/swagger/index.html\n", serverPort)
	fmt.Println("🔧 CORS enabled for all origins")
	log.Fatal(http.ListenAndServe(":"+serverPort, nil))
}

// createUsersTable - users jadvalini yaratadi (agar mavjud bo'lmasa)
func createUsersTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		full_name VARCHAR(255) NOT NULL,
		phone VARCHAR(20) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE,
		avatar_url VARCHAR(500),
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Users jadvalini yaratishda xatolik: %v", err)
	} else {
		fmt.Println("✅ Users jadvali tayyor!")
	}

	// Mavjud jadvalga yangi ustunlarni qo'shish (agar yo'q bo'lsa)
	addColumnIfNotExists(db, "users", "email", "VARCHAR(255) UNIQUE")
	addColumnIfNotExists(db, "users", "avatar_url", "VARCHAR(500)")
	addColumnIfNotExists(db, "users", "is_active", "BOOLEAN DEFAULT TRUE")
	addColumnIfNotExists(db, "users", "role", "VARCHAR(50) DEFAULT 'customer'")
	addColumnIfNotExists(db, "users", "onesignal_id", "VARCHAR(255)")

	// Seller Profiles jadvalini yaratish
	createSellerProfilesTable(db)
}

// createSellerProfilesTable - seller_profiles jadvalini yaratadi (Multi-Shop)
func createSellerProfilesTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS seller_profiles (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- One User -> Many Shops
		
		-- Biznes ma'lumotlari
		shop_name VARCHAR(255) NOT NULL,
		slug VARCHAR(255) UNIQUE,
		description TEXT,
		logo_url VARCHAR(500),
		banner_url VARCHAR(500),
		
		-- Yuridik va moliyaviy ma'lumotlar
		legal_name VARCHAR(255),
		tax_id VARCHAR(50),
		bank_account VARCHAR(50),
		bank_name VARCHAR(255),
		
		-- Aloqa va joylashuv
		support_phone VARCHAR(20),
		address VARCHAR(500),
		latitude FLOAT8,
		longitude FLOAT8,
		
		-- JSONB maydonlari
		social_links JSONB DEFAULT '{}',
		working_hours JSONB DEFAULT '{}',
		
		-- Status va reyting
		is_verified BOOLEAN DEFAULT FALSE,
		rating FLOAT DEFAULT 0,
		
		-- Vaqt belgilari
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Seller profiles jadvalini yaratishda xatolik: %v", err)
	} else {
		fmt.Println("✅ Seller Profiles jadvali tayyor!")
	}

	// Indekslar
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_seller_profiles_user_id ON seller_profiles(user_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_seller_profiles_shop_name ON seller_profiles(shop_name)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_seller_profiles_slug ON seller_profiles(slug)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_seller_profiles_is_verified ON seller_profiles(is_verified)`)
}

// addColumnIfNotExists - mavjud jadvalga ustun qo'shadi (agar yo'q bo'lsa)
func addColumnIfNotExists(db *sql.DB, table, column, dataType string) {
	query := fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = '%s' AND column_name = '%s'
			) THEN
				ALTER TABLE %s ADD COLUMN %s %s;
			END IF;
		END $$;
	`, table, column, table, column, dataType)

	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Ustun qo'shishda xatolik (%s.%s): %v", table, column, err)
	}
}

// initSMSService - Eskiz.uz SMS xizmatini sozlash
func initSMSService() {
	eskizEmail := os.Getenv("ESKIZ_EMAIL")
	eskizPassword := os.Getenv("ESKIZ_PASSWORD")

	if eskizEmail == "" || eskizPassword == "" {
		fmt.Println("⚠️  ESKIZ_EMAIL yoki ESKIZ_PASSWORD o'rnatilmagan")
		fmt.Println("   SMS xizmati MOCK rejimida ishlaydi (konsolga chiqadi)")
		fmt.Println("   Real SMS yuborish uchun environment variable o'rnating:")
		fmt.Println("   export ESKIZ_EMAIL=your@email.com")
		fmt.Println("   export ESKIZ_PASSWORD=yourpassword")
		return
	}

	// Eskiz servisini yaratish
	eskizService := sms.NewEskizService(eskizEmail, eskizPassword)

	// Dastlabki login (background)
	go func() {
		if err := eskizService.Login(); err != nil {
			log.Printf("⚠️ Eskiz login xatosi: %v", err)
			log.Println("   SMS xizmati keyingi so'rovda qayta urinadi")
		}
	}()

	// Handlers ga SMS servisini o'rnatish
	handlers.SetSMSService(eskizService)

	fmt.Println("✅ Eskiz SMS xizmati ulandi!")
}
