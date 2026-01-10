package main

import (
	"database/sql"
	"fmt"
	"log"
	"mebellar-backend/handlers"
	"net/http"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "mebel_user"
	password = "MebelStrong2024!"
	dbname   = "mebellar_olami"
)

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

	// 2. Marshrutlar (Routes) - CORS middleware bilan
	// Mahsulotlar
	http.HandleFunc("/api/products", corsMiddleware(handlers.GetProducts(db)))

	// Autentifikatsiya endpointlari
	http.HandleFunc("/api/auth/send-otp", corsMiddleware(handlers.SendOTP(db)))
	http.HandleFunc("/api/auth/verify-otp", corsMiddleware(handlers.VerifyOTP(db)))
	http.HandleFunc("/api/auth/register", corsMiddleware(handlers.Register(db)))
	http.HandleFunc("/api/auth/login", corsMiddleware(handlers.Login(db)))
	http.HandleFunc("/api/auth/forgot-password", corsMiddleware(handlers.ForgotPassword(db)))
	http.HandleFunc("/api/auth/reset-password", corsMiddleware(handlers.ResetPassword(db)))

	// 3. Serverni yoqish
	fmt.Println("🚀 Server 8081-portda ishlayapti...")
	fmt.Println("📱 Auth endpoints:")
	fmt.Println("   POST /api/auth/send-otp")
	fmt.Println("   POST /api/auth/verify-otp")
	fmt.Println("   POST /api/auth/register")
	fmt.Println("   POST /api/auth/login")
	fmt.Println("   POST /api/auth/forgot-password")
	fmt.Println("   POST /api/auth/reset-password")
	fmt.Println("")
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
}
