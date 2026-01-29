package testutil

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// SetupTestDB создает тестовую базу данных
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbHost := getEnvOrDefault("TEST_DB_HOST", "localhost")
	dbPort := getEnvOrDefault("TEST_DB_PORT", "5432")
	dbUser := getEnvOrDefault("TEST_DB_USER", "mebel_user")
	dbPassword := getEnvOrDefault("TEST_DB_PASSWORD", "")
	dbName := getEnvOrDefault("TEST_DB_NAME", "mebellar_test")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err = db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// CleanupTestDB очищает тестовую БД
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{"users", "seller_profiles", "products", "orders"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}

	db.Close()
}

// CreateTestUser создает тестового пользователя
func CreateTestUser(t *testing.T, db *sql.DB, phone, password, role string) string {
	t.Helper()

	userID := "test-user-" + GenerateRandomString(8)

	_, err := db.Exec(`
        INSERT INTO users (id, full_name, phone, password_hash, role, is_active)
        VALUES ($1, $2, $3, $4, $5, true)
    `, userID, "Test User", phone, password, role)

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return userID
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// GenerateRandomString генерирует случайную строку
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
