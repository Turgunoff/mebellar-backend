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

	_ "mebellar-backend/docs" // Swagger docs

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
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

// @host      api.mebellar-olami.uz
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

func main() {
	// 1. Konfiguratsiyani yuklash
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env fayli topilmadi, environment variablelardan foydalaniladi")
	} else {
		fmt.Println("✅ .env fayli yuklandi")
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "mebel_user")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "mebellar_olami")
	serverPort := getEnv("SERVER_PORT", "8081")
	jwtSecret := getEnv("JWT_SECRET", "mebellar-super-secret-key-2024")

	// Global sozlamalar
	handlers.SetJWTSecret(jwtSecret)
	websocket.SetJWTSecret(jwtSecret)

	// 2. Bazaga ulanish
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("❌ Database connection error: ", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("❌ Database ping error: ", err)
	}
	fmt.Printf("✅ Baza ulangan! (%s@%s:%s/%s)\n", dbUser, dbHost, dbPort, dbName)

	// 3. Jadvallarni tekshirish va yaratish
	createUsersTable(db)

	// 4. Xizmatlarni ishga tushirish
	initSMSService()
	websocket.InitGlobalHub()

	// 5. Marshrutlarni ro'yxatdan o'tkazish (http.HandleFunc)
	// Izoh: corsMiddleware olib tashlandi, chunki quyida rs/cors ishlatilmoqda.

	// --- Public Endpoints ---
	http.HandleFunc("/api/categories", handlers.GetCategories(db))
	http.HandleFunc("/api/categories/", handlers.GetCategoryByID(db))
	http.HandleFunc("/api/regions", handlers.GetRegions(db))
	http.HandleFunc("/api/products", handlers.GetProducts(db))
	http.HandleFunc("/api/products/new", handlers.GetNewArrivals(db))
	http.HandleFunc("/api/products/popular", handlers.GetPopularProducts(db))
	http.HandleFunc("/api/products/", handlers.GetProductByID(db))
	http.HandleFunc("/api/shops/", handlers.GetPublicShopBySlug(db))
	http.HandleFunc("/api/common/cancellation-reasons", handlers.GetCancellationReasons(db))

	// --- Auth Endpoints ---
	http.HandleFunc("/api/auth/send-otp", handlers.SendOTP(db))
	http.HandleFunc("/api/auth/verify-otp", handlers.VerifyOTP(db))
	http.HandleFunc("/api/auth/register", handlers.Register(db))
	http.HandleFunc("/api/auth/login", handlers.Login(db))
	http.HandleFunc("/api/auth/forgot-password", handlers.ForgotPassword(db))
	http.HandleFunc("/api/auth/reset-password", handlers.ResetPassword(db))

	// --- User (Protected) Endpoints ---
	http.HandleFunc("/api/user/me", handlers.JWTMiddleware(db, userMeHandler(db)))
	http.HandleFunc("/api/user/change-phone/request", handlers.JWTMiddleware(db, handlers.RequestPhoneChange(db)))
	http.HandleFunc("/api/user/change-phone/verify", handlers.JWTMiddleware(db, handlers.VerifyPhoneChange(db)))
	http.HandleFunc("/api/user/change-email/request", handlers.JWTMiddleware(db, handlers.RequestEmailChange(db)))
	http.HandleFunc("/api/user/change-email/verify", handlers.JWTMiddleware(db, handlers.VerifyEmailChange(db)))
	http.HandleFunc("/api/user/become-seller", handlers.JWTMiddleware(db, handlers.BecomeSeller(db)))
	http.HandleFunc("/api/orders", handlers.CustomerOrdersHandler(db)) // Mijoz buyurtmalari

	// --- Seller (Protected) Endpoints ---
	http.HandleFunc("/api/seller/shops", handlers.JWTMiddleware(db, handlers.ShopsHandler(db)))
	http.HandleFunc("/api/seller/shops/", handlers.JWTMiddleware(db, handlers.ShopByIDHandler(db)))
	http.HandleFunc("/api/seller/products", handlers.JWTMiddleware(db, handlers.SellerProductsHandler(db)))
	http.HandleFunc("/api/seller/products/", handlers.JWTMiddleware(db, handlers.SellerProductItemHandler(db)))
	http.HandleFunc("/api/seller/orders", handlers.JWTMiddleware(db, handlers.SellerOrdersHandler(db)))
	http.HandleFunc("/api/seller/orders/stats", handlers.JWTMiddleware(db, handlers.GetOrderStats(db)))
	http.HandleFunc("/api/seller/orders/", handlers.JWTMiddleware(db, handlers.UpdateOrderStatus(db)))
	http.HandleFunc("/api/seller/profile", handlers.JWTMiddleware(db, handlers.SellerProfileHandler(db)))
	http.HandleFunc("/api/seller/account", handlers.JWTMiddleware(db, handlers.DeleteSellerAccount(db)))
	http.HandleFunc("/api/seller/dashboard/stats", handlers.JWTMiddleware(db, handlers.GetDashboardStats(db)))
	http.HandleFunc("/api/seller/analytics/cancellations", handlers.JWTMiddleware(db, handlers.GetCancellationStats(db)))

	// --- Admin (Protected: Admin/Moderator Only) ---
	http.HandleFunc("/api/admin/dashboard-stats", handlers.RequireRole(db, "admin", "moderator")(handlers.GetAdminDashboardStats(db)))
	http.HandleFunc("/api/admin/users", handlers.RequireRole(db, "admin", "moderator")(handlers.GetUsers(db)))
	http.HandleFunc("/api/admin/categories/list", handlers.RequireRole(db, "admin", "moderator")(handlers.GetAdminCategories(db)))
	http.HandleFunc("/api/admin/categories", handlers.RequireRole(db, "admin", "moderator")(handlers.CreateCategory(db)))
	http.HandleFunc("/api/admin/categories/", handlers.RequireRole(db, "admin", "moderator")(handlers.AdminCategoryHandler(db)))
	
	// --- Admin Sellers Management ---
	http.HandleFunc("/api/admin/sellers", handlers.RequireRole(db, "admin", "moderator")(handlers.GetSellers(db)))
	http.HandleFunc("/api/admin/sellers/", handlers.RequireRole(db, "admin", "moderator")(handlers.AdminSellerHandler(db)))
	
	// --- Admin Shops Management ---
	http.HandleFunc("/api/admin/shops", handlers.RequireRole(db, "admin", "moderator")(handlers.AdminShopsHandler(db)))
	http.HandleFunc("/api/admin/shops/", handlers.RequireRole(db, "admin", "moderator")(handlers.AdminShopItemHandler(db)))

	// --- WebSocket ---
	http.HandleFunc("/ws/orders", websocket.HandleWebSocket(db))

	// --- Static Files ---
	fs := http.FileServer(http.Dir("uploads"))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// --- Swagger ---
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// --- Debug/Seed ---
	http.HandleFunc("/api/debug/seed-orders", handlers.JWTMiddleware(db, handlers.SeedOrders(db)))

	// 6. CORS Sozlamalari va Serverni ishga tushirish
	fmt.Println("🚀 Server ishga tushmoqda...")

	// rs/cors kutubxonasi barcha CORS logikasini boshqaradi
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:5173",           // Local Admin/Frontend
			"https://admin.mebellar-olami.uz", // Production Admin
			"https://mebellar-olami.uz",       // Production Landing
			"https://api.mebellar-olami.uz",   // API o'zi
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept", "X-Requested-With", "X-Shop-ID"}, // X-Shop-ID qo'shildi
		AllowCredentials: true,
		Debug:            true, // Ishlab chiqish jarayonida yoqib turish foydali
	})

	// Barcha marshrutlarni CORS handler bilan o'rash
	handler := c.Handler(http.DefaultServeMux)

	fmt.Printf("✅ Server %s-portda tayyor!\n", serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, handler))
}

// ---------------------------------------------------------
// Yordamchi Funksiyalar (DB Migration)
// ---------------------------------------------------------

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

	// Ustunlarni tekshirish va qo'shish (Migration)
	addColumnIfNotExists(db, "users", "email", "VARCHAR(255) UNIQUE")
	addColumnIfNotExists(db, "users", "avatar_url", "VARCHAR(500)")
	addColumnIfNotExists(db, "users", "is_active", "BOOLEAN DEFAULT TRUE")
	addColumnIfNotExists(db, "users", "role", "VARCHAR(50) DEFAULT 'customer'")
	addColumnIfNotExists(db, "users", "onesignal_id", "VARCHAR(255)")

	createSellerProfilesTable(db)
}

func createSellerProfilesTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS seller_profiles (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- user_id INTEGER bo'lishi kerak (serial), UUID emas
		shop_name VARCHAR(255) NOT NULL,
		slug VARCHAR(255) UNIQUE,
		description TEXT,
		logo_url VARCHAR(500),
		banner_url VARCHAR(500),
		legal_name VARCHAR(255),
		tax_id VARCHAR(50),
		bank_account VARCHAR(50),
		bank_name VARCHAR(255),
		support_phone VARCHAR(20),
		address VARCHAR(500),
		latitude FLOAT8,
		longitude FLOAT8,
		social_links JSONB DEFAULT '{}',
		working_hours JSONB DEFAULT '{}',
		is_verified BOOLEAN DEFAULT FALSE,
		rating FLOAT DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`
	// Eslatma: user_id turi 'users' jadvalidagi 'id' turi bilan mos bo'lishi kerak.
	// Agar users.id SERIAL (int) bo'lsa, bu yerda user_id INTEGER bo'lishi kerak.
	// Agar users.id UUID bo'lsa, bu yerda ham UUID bo'lishi kerak.
	// Yuqoridagi kodda users.id SERIAL deb olingan.

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

func initSMSService() {
	eskizEmail := os.Getenv("ESKIZ_EMAIL")
	eskizPassword := os.Getenv("ESKIZ_PASSWORD")

	if eskizEmail == "" || eskizPassword == "" {
		fmt.Println("⚠️  ESKIZ_EMAIL yoki ESKIZ_PASSWORD o'rnatilmagan")
		return
	}
	eskizService := sms.NewEskizService(eskizEmail, eskizPassword)
	go func() {
		if err := eskizService.Login(); err != nil {
			log.Printf("⚠️ Eskiz login xatosi: %v", err)
		}
	}()
	handlers.SetSMSService(eskizService)
	fmt.Println("✅ Eskiz SMS xizmati ulandi!")
}