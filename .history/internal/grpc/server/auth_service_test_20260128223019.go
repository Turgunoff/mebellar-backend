package server

import (
	"context"
	"testing"

	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/sms"
	"mebellar-backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	// Создаем mock SMS сервис
	mockSMS := &sms.MockSMSService{}

	// Создаем auth service
	jwtSecret := []byte("test-secret-key-for-testing-32chars")
	authService := NewAuthServiceServer(db, jwtSecret, mockSMS)

	tests := []struct {
		name        string
		request     *pb.RegisterRequest
		wantErr     bool
		errContains string
	}{
		{
			name: "Успешная регистрация",
			request: &pb.RegisterRequest{
				FullName: "Test User",
				Phone:    "+998901234567",
				Password: "password123",
				Role:     "customer",
			},
			wantErr: false,
		},
		{
			name: "Пустое имя",
			request: &pb.RegisterRequest{
				FullName: "",
				Phone:    "+998901234567",
				Password: "password123",
			},
			wantErr:     true,
			errContains: "обязательно",
		},
		{
			name: "Неверный формат телефона",
			request: &pb.RegisterRequest{
				FullName: "Test User",
				Phone:    "123456789",
				Password: "password123",
			},
			wantErr:     true,
			errContains: "формат",
		},
		{
			name: "Короткий пароль",
			request: &pb.RegisterRequest{
				FullName: "Test User",
				Phone:    "+998901234567",
				Password: "123",
			},
			wantErr:     true,
			errContains: "минимум",
		},
		{
			name: "Пароль без букв",
			request: &pb.RegisterRequest{
				FullName: "Test User",
				Phone:    "+998901234568",
				Password: "12345678",
			},
			wantErr:     true,
			errContains: "буквы",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := authService.Register(ctx, tt.request)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
				assert.NotNil(t, resp.User)
				assert.Equal(t, tt.request.Phone, resp.User.Phone)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	mockSMS := &sms.MockSMSService{}
	jwtSecret := []byte("test-secret-key-for-testing-32chars")
	authService := NewAuthServiceServer(db, jwtSecret, mockSMS)

	// Подготовка: создаем тестового пользователя
	password := "testpassword123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	_, err := db.Exec(`
		INSERT INTO users (id, full_name, phone, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
	`, "test-user-1", "Test User", "+998901234567", string(hashedPassword), "customer")
	require.NoError(t, err)

	tests := []struct {
		name        string
		phone       string
		password    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "Успешный вход",
			phone:    "+998901234567",
			password: password,
			wantErr:  false,
		},
		{
			name:        "Неверный пароль",
			phone:       "+998901234567",
			password:    "wrongpassword",
			wantErr:     true,
			errContains: "пароль",
		},
		{
			name:        "Пользователь не найден",
			phone:       "+998909999999",
			password:    password,
			wantErr:     true,
			errContains: "не найден",
		},
		{
			name:        "Пустой телефон",
			phone:       "",
			password:    password,
			wantErr:     true,
			errContains: "обязателен",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &pb.LoginRequest{
				Phone:    tt.phone,
				Password: tt.password,
			}

			resp, err := authService.Login(ctx, req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
				assert.Equal(t, tt.phone, resp.User.Phone)
			}
		})
	}
}

func TestAuthService_SendOTP(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	mockSMS := &sms.MockSMSService{}
	jwtSecret := []byte("test-secret-key-for-testing-32chars")
	authService := NewAuthServiceServer(db, jwtSecret, mockSMS)

	tests := []struct {
		name        string
		phone       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "Валидный телефон",
			phone:   "+998901234567",
			wantErr: false,
		},
		{
			name:        "Неверный формат",
			phone:       "123456789",
			wantErr:     true,
			errContains: "формат",
		},
		{
			name:        "Пустой телефон",
			phone:       "",
			wantErr:     true,
			errContains: "обязателен",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &pb.SendOTPRequest{
				Phone: tt.phone,
			}

			resp, err := authService.SendOTP(ctx, req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.True(t, resp.Success)
			}
		})
	}
}

func TestAuthService_DuplicateRegistration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	mockSMS := &sms.MockSMSService{}
	jwtSecret := []byte("test-secret-key-for-testing-32chars")
	authService := NewAuthServiceServer(db, jwtSecret, mockSMS)

	// Первая регистрация
	ctx := context.Background()
	registerReq := &pb.RegisterRequest{
		FullName: "Test User",
		Phone:    "+998901234567",
		Password: "password123",
		Role:     "customer",
	}

	resp1, err := authService.Register(ctx, registerReq)
	require.NoError(t, err)
	assert.NotNil(t, resp1)

	// Попытка повторной регистрации
	resp2, err := authService.Register(ctx, registerReq)
	require.Error(t, err)
	assert.Nil(t, resp2)
	assert.Contains(t, err.Error(), "зарегистрирован")
}
