package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"mebellar-backend/handlers"
	"mebellar-backend/pkg/seed"

	_ "mebellar-backend/docs" // Swagger docs - swag init dan keyin paydo bo'ladi

	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "mebel_user"
	password = "MebelStrong2024!"
	dbname   = "mebellar_olami"
)

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
	// 1. Bazaga ulanish
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Baza ulangan!")

	// Users jadvalini yaratish (agar mavjud bo'lmasa)
	createUsersTable(db)

	// Kategoriyalar va mahsulotlarni seed qilish
	seed.SeedAll(db)

	// 2. Marshrutlar (Routes) - CORS middleware bilan
	// Kategoriyalar
	http.HandleFunc("/api/categories", corsMiddleware(handlers.GetCategories(db)))
	http.HandleFunc("/api/categories/", corsMiddleware(handlers.GetCategoryByID(db))) // /api/categories/{id}

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
	fmt.Println("📁 Static files: /uploads/*")
	fmt.Println("📚 Swagger UI: http://45.93.201.167:8081/swagger/index.html")
	fmt.Println("🔧 CORS enabled for all origins")
	log.Fatal(http.ListenAndServe(":8081", nil))
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
