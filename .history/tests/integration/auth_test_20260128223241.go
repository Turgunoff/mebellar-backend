//go:build integration

package integration

import (
	"context"
	"database/sql"
	"testing"

	"mebellar-backend/internal/grpc/server"
	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/sms"
	"mebellar-backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_AuthFlow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Полный flow: Register -> Login -> Refresh Token
	ctx := context.Background()

	// 1. Регистрация
	authService := setupAuthService(t, db)

	registerReq := &pb.RegisterRequest{
		FullName: "Integration Test User",
		Phone:    "+998901111111",
		Password: "testpass123",
		Role:     "customer",
	}

	registerResp, err := authService.Register(ctx, registerReq)
	require.NoError(t, err)
	assert.NotEmpty(t, registerResp.AccessToken)
	assert.NotEmpty(t, registerResp.RefreshToken)
	assert.Equal(t, registerReq.Phone, registerResp.User.Phone)

	// 2. Login с теми же credentials
	loginReq := &pb.LoginRequest{
		Phone:    "+998901111111",
		Password: "testpass123",
	}

	loginResp, err := authService.Login(ctx, loginReq)
	require.NoError(t, err)
	assert.NotEmpty(t, loginResp.AccessToken)
	assert.NotEmpty(t, loginResp.RefreshToken)
	assert.Equal(t, registerResp.User.Id, loginResp.User.Id)

	// 3. Проверка что нельзя зарегистрироваться повторно
	_, err = authService.Register(ctx, registerReq)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "зарегистрирован")

	// 4. Refresh token
	refreshReq := &pb.RefreshTokenRequest{
		RefreshToken: loginResp.RefreshToken,
	}

	refreshResp, err := authService.RefreshToken(ctx, refreshReq)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshResp.AccessToken)
	assert.NotEmpty(t, refreshResp.RefreshToken)
	// Access token должен быть новым
	assert.NotEqual(t, loginResp.AccessToken, refreshResp.AccessToken)
}

func TestIntegration_MultipleUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	ctx := context.Background()
	authService := setupAuthService(t, db)

	// Создание нескольких пользователей
	users := []struct {
		fullName string
		phone    string
		password string
		role     string
	}{
		{"User One", "+998901111111", "password123", "customer"},
		{"User Two", "+998902222222", "password456", "seller"},
		{"User Three", "+998903333333", "password789", "customer"},
	}

	registeredUsers := make(map[string]string) // phone -> userID

	for _, u := range users {
		req := &pb.RegisterRequest{
			FullName: u.fullName,
			Phone:    u.phone,
			Password: u.password,
			Role:     u.role,
		}
		resp, err := authService.Register(ctx, req)
		require.NoError(t, err)
		registeredUsers[u.phone] = resp.User.Id
		assert.Equal(t, u.role, resp.User.Role)
	}

	// Все пользователи могут логиниться
	for _, u := range users {
		req := &pb.LoginRequest{
			Phone:    u.phone,
			Password: u.password,
		}
		resp, err := authService.Login(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, registeredUsers[u.phone], resp.User.Id)
	}
}

func setupAuthService(t *testing.T, db *sql.DB) *server.AuthServiceServer {
	t.Helper()
	mockSMS := &sms.MockSMSService{}
	jwtSecret := []byte("test-secret-key-for-testing-32chars")
	return server.NewAuthServiceServer(db, jwtSecret, mockSMS)
}
