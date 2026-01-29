package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/internal/grpc/server"
	"mebellar-backend/pkg/database"
	"mebellar-backend/pkg/logger"
	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/sms"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// getEnv - environment variable olish (default bilan)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt получает integer из environment или возвращает default
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
		log.Printf("⚠️  Invalid integer value for %s: %s, using default: %d", key, val, defaultValue)
	}
	return defaultValue
}

// getEnvDuration получает duration из environment или возвращает default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		log.Printf("⚠️  Invalid duration value for %s: %s, using default: %v", key, val, defaultValue)
	}
	return defaultValue
}

// validateConfig валидирует критические параметры конфигурации
func validateConfig() error {
	environment := getEnv("ENVIRONMENT", "development")

	logger.Debug("Validating configuration", zap.String("environment", environment))

	// 1. Проверка JWT_SECRET
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET is required. Generate one with: openssl rand -base64 32")
	}

	if len(jwtSecret) < 32 {
		logger.Warn("JWT_SECRET is shorter than recommended",
			zap.Int("current_length", len(jwtSecret)),
			zap.Int("recommended", 32),
		)
	}

	// Проверка на дефолтное значение
	forbiddenSecrets := []string{
		"mebellar-super-secret-key-2024",
		"your-secret-key",
		"secret",
	}
	for _, forbidden := range forbiddenSecrets {
		if jwtSecret == forbidden {
			return fmt.Errorf("JWT_SECRET cannot be default value. Generate secure random string")
		}
	}

	// 2. Проверка database credentials в production
	if environment == "production" {
		dbPassword := getEnv("DB_PASSWORD", "")
		if dbPassword == "" || dbPassword == "MebelStrong2024!" {
			return fmt.Errorf("DB_PASSWORD must be set and not default in production")
		}

		sslMode := getEnv("DB_SSLMODE", "disable")
		if sslMode == "disable" {
			return fmt.Errorf("DB_SSLMODE cannot be 'disable' in production")
		}
	}

	// 3. Проверка обязательных переменных
	required := map[string]string{
		"DB_HOST":     getEnv("DB_HOST", ""),
		"DB_PORT":     getEnv("DB_PORT", ""),
		"DB_USER":     getEnv("DB_USER", ""),
		"DB_PASSWORD": getEnv("DB_PASSWORD", ""),
		"DB_NAME":     getEnv("DB_NAME", ""),
	}

	for key, value := range required {
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", key)
		}
	}

	return nil
}

// configureConnectionPool настраивает параметры connection pool для PostgreSQL
func configureConnectionPool(db *sql.DB) {
	// Максимальное количество открытых соединений
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 25)

	// Максимальное количество idle соединений
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 5)

	// Максимальное время жизни соединения
	connMaxLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	// Максимальное время idle для соединения
	connMaxIdleTime := getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	logger.Info("Connection pool configured",
		zap.Int("max_open_connections", maxOpenConns),
		zap.Int("max_idle_connections", maxIdleConns),
		zap.Duration("connection_max_lifetime", connMaxLifetime),
		zap.Duration("connection_max_idle_time", connMaxIdleTime),
	)
}

func main() {
	// 1. Konfiguratsiyani yuklash
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env fayli topilmadi, environment variablelardan foydalaniladi")
	} else {
		fmt.Println("✅ .env fayli yuklandi")
	}

	environment := getEnv("ENVIRONMENT", "development")

	// 2. Logger инициализация
	if err := logger.InitLogger(environment); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Mebellar Backend",
		zap.String("environment", environment),
		zap.String("version", "1.0.0"),
	)

	// 3. Валидация конфигурации
	if err := validateConfig(); err != nil {
		logger.Fatal("Configuration validation failed", zap.Error(err))
	}
	logger.Info("Configuration validated successfully")

	// 4. Загрузка параметров
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "mebel_user")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "mebellar_olami")
	sslMode := getEnv("DB_SSLMODE", "disable")
	sslRootCert := getEnv("DB_SSL_ROOT_CERT", "")
	staticPort := getEnv("STATIC_PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "50051")
	jwtSecret := getEnv("JWT_SECRET", "mebellar-super-secret-key-2024")

	// 5. Предупреждение для production
	if environment == "production" && sslMode == "disable" {
		logger.Fatal("CRITICAL: SSL must be enabled in production! Set DB_SSLMODE=require")
	}

	// 6. Bazaga ulanish
	logger.Info("Connecting to database",
		zap.String("host", dbHost),
		zap.String("port", dbPort),
		zap.String("database", dbName),
		zap.String("user", dbUser),
		zap.String("ssl_mode", sslMode),
	)

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	// Добавить для verify-full режима
	if sslMode == "verify-full" && sslRootCert != "" {
		psqlInfo += fmt.Sprintf(" sslrootcert=%s", sslRootCert)
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		logger.Fatal("Database connection error", zap.Error(err))
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		logger.Fatal("Database ping error", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// 7. Настройка connection pool
	configureConnectionPool(db)

	// 8. Миграции
	logger.Info("Running database migrations")
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		logger.Fatal("Migration error", zap.Error(err))
	}

	// 9. SMS xizmatini ishga tushirish
	logger.Info("Initializing SMS service")
	smsService := initSMSService()

	// 10. gRPC Server setup
	logger.Info("Configuring gRPC server")

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

	// Keepalive settings to prevent stream disconnection
	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      0,                 // Infinite
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5s for pending RPCs to finish
		Time:                  20 * time.Second, // Ping the client every 20 seconds to keep the connection alive
		Timeout:               5 * time.Second,  // Wait 5 seconds for the ping response
	}

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // Minimum time between client pings
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}

	// Chain interceptors: Logger first, then Auth
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kasp),
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.ChainUnaryInterceptor(
			middleware.UnaryLogger,
			unaryAuthInterceptor,
		),
		grpc.ChainStreamInterceptor(
			middleware.StreamLogger,
			streamAuthInterceptor,
		),
	)

	// Register all gRPC services
	authService := server.NewAuthServiceServer(db, []byte(jwtSecret), smsService)
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

	// 11. Start servers
	logger.Info("Starting servers",
		zap.String("static_port", staticPort),
		zap.String("grpc_port", grpcPort),
	)

	var wg sync.WaitGroup
	wg.Add(2)

	// Start Static File Server (for serving uploaded images)
	go func() {
		defer wg.Done()

		mux := http.NewServeMux()

		// Static files for uploads
		fs := http.FileServer(http.Dir("uploads"))
		mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

		// Health check endpoint with connection pool stats
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// Проверка подключения к БД
			if err := db.Ping(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "unhealthy",
					"error":  "database unavailable",
				})
				return
			}

			// Статистика connection pool
			stats := db.Stats()

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "ok",
				"service": "mebellar-backend",
				"database": map[string]interface{}{
					"open_connections":    stats.OpenConnections,
					"in_use":              stats.InUse,
					"idle":                stats.Idle,
					"wait_count":          stats.WaitCount,
					"wait_duration_ms":    stats.WaitDuration.Milliseconds(),
					"max_idle_closed":     stats.MaxIdleClosed,
					"max_lifetime_closed": stats.MaxLifetimeClosed,
				},
			})
		})

		logger.Info("Static file server started",
			zap.String("port", staticPort),
			zap.String("upload_dir", "uploads"),
		)
		if err := http.ListenAndServe(":"+staticPort, mux); err != nil {
			logger.Fatal("Static server error", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			logger.Fatal("gRPC listener error", zap.Error(err))
		}
		logger.Info("gRPC server started",
			zap.String("port", grpcPort),
			zap.Strings("services", []string{
				"AuthService", "UserService", "OrderService",
				"ProductService", "CategoryService", "ShopService", "CommonService",
			}),
		)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	wg.Wait()
}

func initSMSService() sms.SMSService {
	eskizEmail := os.Getenv("ESKIZ_EMAIL")
	eskizPassword := os.Getenv("ESKIZ_PASSWORD")

	if eskizEmail == "" || eskizPassword == "" {
		logger.Warn("ESKIZ_EMAIL or ESKIZ_PASSWORD not set, SMS service will not be initialized")
		return nil
	}
	eskizService := sms.NewEskizService(eskizEmail, eskizPassword)
	go func() {
		if err := eskizService.Login(); err != nil {
			logger.Error("Eskiz login error", zap.Error(err))
		} else {
			logger.Info("Eskiz SMS service connected")
		}
	}()
	return eskizService
}
