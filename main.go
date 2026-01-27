package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/internal/grpc/server"
	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/sms"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// getEnv - environment variable olish (default bilan)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
	staticPort := getEnv("STATIC_PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "50051")
	jwtSecret := getEnv("JWT_SECRET", "mebellar-super-secret-key-2024")

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

	// 4. SMS xizmatini ishga tushirish
	initSMSService()

	// 5. gRPC Server setup
	fmt.Println("🔧 gRPC server sozlanmoqda...")

	// Methods that don't require authentication
	skipAuthMethods := map[string]bool{
		// Auth service - public
		"/auth.AuthService/SendOTP":      true,
		"/auth.AuthService/Login":        true,
		"/auth.AuthService/Register":     true,
		"/auth.AuthService/VerifyOTP":    true,
		"/auth.AuthService/RefreshToken": true,

		// Product service - public read endpoints
		"/product.ProductService/GetProduct":                       true,
		"/product.ProductService/ListProducts":                     true,
		"/product.ProductService/ListNewArrivals":                  true,
		"/product.ProductService/ListPopularProducts":              true,
		"/product.ProductService/ListProductsGroupedBySubcategory": true,

		// Category service - public read endpoints
		"/category.CategoryService/ListCategories":         true,
		"/category.CategoryService/ListFlatCategories":     true,
		"/category.CategoryService/GetCategory":            true,
		"/category.CategoryService/ListCategoryAttributes": true,
		"/category.CategoryService/GetCategoryAttribute":   true,

		// Shop service - public endpoints
		"/shop.ShopService/GetShopBySlug":          true,
		"/shop.ShopService/GetPublicSellerProfile": true,

		// Common service - public endpoints
		"/common.CommonService/ListRegions":             true,
		"/common.CommonService/GetRegion":               true,
		"/common.CommonService/ListBanners":             true,
		"/common.CommonService/GetBanner":               true,
		"/common.CommonService/ListCancellationReasons": true,

		// Order service - create order is public (guest checkout)
		"/order.OrderService/CreateOrder": true,
	}

	unaryAuthInterceptor, streamAuthInterceptor := middleware.NewAuthInterceptors(
		[]byte(jwtSecret),
		db,
		skipAuthMethods,
	)

	// Chain interceptors: Logger first, then Auth
	unaryInterceptor := grpc.ChainUnaryInterceptor(
		middleware.UnaryLogger,
		unaryAuthInterceptor,
	)
	streamInterceptor := grpc.ChainStreamInterceptor(
		middleware.StreamLogger,
		streamAuthInterceptor,
	)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	// Register all gRPC services
	authService := server.NewAuthServiceServer(db, []byte(jwtSecret))
	pb.RegisterAuthServiceServer(grpcServer, authService)

	userService := server.NewUserServiceServer(db)
	pb.RegisterUserServiceServer(grpcServer, userService)

	orderService := server.NewOrderServiceServer(db)
	pb.RegisterOrderServiceServer(grpcServer, orderService)

	productService := server.NewProductServiceServer(db)
	pb.RegisterProductServiceServer(grpcServer, productService)

	categoryService := server.NewCategoryServiceServer(db)
	pb.RegisterCategoryServiceServer(grpcServer, categoryService)

	shopService := server.NewShopServiceServer(db)
	pb.RegisterShopServiceServer(grpcServer, shopService)

	commonService := server.NewCommonServiceServer(db)
	pb.RegisterCommonServiceServer(grpcServer, commonService)

	// Enable reflection for gRPC CLI tools (grpcurl, grpcui, etc.)
	reflection.Register(grpcServer)

	// 6. Start servers
	fmt.Println("🚀 Serverlar ishga tushmoqda...")

	var wg sync.WaitGroup
	wg.Add(2)

	// Start Static File Server (for serving uploaded images)
	go func() {
		defer wg.Done()

		mux := http.NewServeMux()

		// Static files for uploads
		fs := http.FileServer(http.Dir("uploads"))
		mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

		// Health check endpoint
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","service":"mebellar-backend"}`))
		})

		fmt.Printf("✅ Static File Server %s-portda tayyor! (uploads servisi)\n", staticPort)
		if err := http.ListenAndServe(":"+staticPort, mux); err != nil {
			log.Fatalf("❌ Static server xatosi: %v", err)
		}
	}()

	// Start gRPC server
	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("❌ gRPC listener xatosi: %v", err)
		}
		fmt.Printf("✅ gRPC Server %s-portda tayyor!\n", grpcPort)
		fmt.Println("📡 Registered services: AuthService, UserService, OrderService, ProductService, CategoryService, ShopService, CommonService")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ gRPC server xatosi: %v", err)
		}
	}()

	wg.Wait()
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
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
		address JSONB DEFAULT '{}',
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
	fmt.Println("✅ Eskiz SMS xizmati ulandi!")
}
