package config

import (
	"log"
	"os"
)

// Config - dastur konfiguratsiyasi
type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Server
	ServerPort string

	// Eskiz SMS Gateway
	EskizEmail    string
	EskizPassword string

	// JWT
	JWTSecret string

	// Gemini AI
	GeminiAPIKey string

	// Environment
	Environment string // "development", "production"
}

// LoadConfig - konfiguratsiyani yuklash
func LoadConfig() *Config {
	cfg := &Config{
		// Database defaults
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "mebel_user"),
		DBPassword: getEnv("DB_PASSWORD", "MebelStrong2024!"),
		DBName:     getEnv("DB_NAME", "mebellar_olami"),

		// Server
		ServerPort: getEnv("SERVER_PORT", "8081"),

		// Eskiz SMS
		EskizEmail:    getEnv("ESKIZ_EMAIL", ""),
		EskizPassword: getEnv("ESKIZ_PASSWORD", ""),

		// JWT
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),

		// Gemini AI
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	// Log config (passwordlarni yashirish)
	log.Println("⚙️ Configuration loaded:")
	log.Printf("   DB: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	log.Printf("   Server Port: %s", cfg.ServerPort)
	log.Printf("   Eskiz Email: %s", maskString(cfg.EskizEmail))
	log.Printf("   Environment: %s", cfg.Environment)

	return cfg
}

// getEnv - environment variable olish (default bilan)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// maskString - stringni yashirish (email@domain.com -> em***@domain.com)
func maskString(s string) string {
	if len(s) < 4 {
		return "***"
	}
	if len(s) < 8 {
		return s[:2] + "***"
	}
	return s[:2] + "***" + s[len(s)-4:]
}

// IsDevelopment - development muhitmi
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction - production muhitmi
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// HasEskizConfig - Eskiz konfiguratsiyasi bormi
func (c *Config) HasEskizConfig() bool {
	return c.EskizEmail != "" && c.EskizPassword != ""
}
