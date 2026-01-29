package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"mebellar-backend/internal/grpc/mapper"
	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/models"
	"mebellar-backend/pkg/apperror"
	"mebellar-backend/pkg/logger"
	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/sms"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthServiceServer struct {
	pb.UnimplementedAuthServiceServer
	db        *sql.DB
	jwtSecret []byte
	sms       sms.SMSService
}

func NewAuthServiceServer(db *sql.DB, jwtSecret []byte, sms sms.SMSService) *AuthServiceServer {
	return &AuthServiceServer{
		db:        db,
		jwtSecret: jwtSecret,
		sms:       sms,
	}
}

func (s *AuthServiceServer) SendOTP(ctx context.Context, req *pb.SendOTPRequest) (*pb.SendOTPResponse, error) {
	// Валидация
	if err := s.validateSendOTPRequest(req.GetPhone()); err != nil {
		return nil, err.(*apperror.AppError).ToGRPCError()
	}

	phone := strings.TrimSpace(req.GetPhone())

	// Проверка существования пользователя
	var existingID string
	var isActive bool
	err := s.db.QueryRowContext(ctx, "SELECT id, COALESCE(is_active, true) FROM users WHERE phone = $1", phone).Scan(&existingID, &isActive)
	if err == nil && isActive {
		logger.Warn("Attempt to send OTP to existing user",
			zap.String("phone", phone),
			zap.String("user_id", existingID),
		)
		return nil, apperror.NewConflictError("Номер телефона уже зарегистрирован").ToGRPCError()
	}

	// Генерация OTP
	rand.Seed(time.Now().UnixNano())
	code := fmt.Sprintf("%05d", rand.Intn(100000))

	// Отправка SMS
	if s.sms != nil {
		if err := s.sms.SendOTP(phone, code); err != nil {
			logger.Error("Failed to send OTP via SMS",
				zap.String("phone", phone),
				zap.Error(err),
			)
			return nil, apperror.NewInternalError("Не удалось отправить SMS", err).ToGRPCError()
		}
	}

	logger.Info("OTP sent successfully",
		zap.String("phone", phone),
	)

	return &pb.SendOTPResponse{
		Success: true,
		Message: "Код верификации отправлен",
	}, nil
}

func (s *AuthServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Валидация через validator
	if err := s.validateLoginRequest(req.GetPhone(), req.GetPassword()); err != nil {
		return nil, err.(*apperror.AppError).ToGRPCError()
	}

	logger.Info("Login attempt",
		zap.String("phone", req.GetPhone()),
	)

	var user models.User
	query := `
		SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), 
		       COALESCE(role, 'customer'), COALESCE(onesignal_id, ''), COALESCE(has_pin, false),
		       COALESCE(password_hash, ''), created_at, updated_at, COALESCE(is_active, true)
		FROM users
		WHERE phone = $1
	`
	var isActive bool
	err := s.db.QueryRowContext(ctx, query, req.GetPhone()).Scan(
		&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL,
		&user.Role, &user.OneSignalID, &user.HasPin, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt, &isActive,
	)
	if errors.Is(err, sql.ErrNoRows) {
		logger.Warn("Login failed: user not found",
			zap.String("phone", req.GetPhone()),
		)
		return nil, apperror.NewNotFoundError("Пользователь не найден").ToGRPCError()
	}

	if err != nil {
		logger.Error("Database error during login",
			zap.String("phone", req.GetPhone()),
			zap.Error(err),
		)
		return nil, apperror.NewDatabaseError("поиск пользователя", err).ToGRPCError()
	}

	if !isActive {
		logger.Warn("Login attempt for inactive user",
			zap.String("phone", req.GetPhone()),
			zap.String("user_id", user.ID),
		)
		return nil, apperror.NewForbiddenError("Учетная запись деактивирована").ToGRPCError()
	}

	// Проверка пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		logger.Warn("Login failed: invalid password",
			zap.String("phone", req.GetPhone()),
			zap.String("user_id", user.ID),
		)
		return nil, apperror.NewUnauthorizedError("Неверный пароль").ToGRPCError()
	}

	// Генерация токенов
	access, refresh, err := s.issueTokens(user.ID, user.Phone, user.Role)
	if err != nil {
		logger.Error("Failed to issue tokens",
			zap.String("user_id", user.ID),
			zap.Error(err),
		)
		return nil, apperror.NewInternalError("Не удалось создать токен", err).ToGRPCError()
	}

	logger.Info("Login successful",
		zap.String("user_id", user.ID),
		zap.String("phone", user.Phone),
		zap.String("role", user.Role),
	)

	return &pb.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         mapper.ToPBUser(&user),
	}, nil
}

func (s *AuthServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Валидация
	if err := s.validateRegisterRequest(
		req.GetFullName(),
		req.GetPhone(),
		req.GetPassword(),
		req.GetRole(),
	); err != nil {
		return nil, err.(*apperror.AppError).ToGRPCError()
	}

	role := req.GetRole()
	if role != "customer" && role != "seller" {
		role = "customer"
	}

	logger.Info("Registration attempt",
		zap.String("phone", req.GetPhone()),
		zap.String("role", role),
	)

	// Хеширование пароля
	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Password hashing failed",
			zap.String("phone", req.GetPhone()),
			zap.Error(err),
		)
		return nil, apperror.NewInternalError("Ошибка обработки пароля", err).ToGRPCError()
	}

	userID := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, full_name, phone, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
	`, userID, req.GetFullName(), req.GetPhone(), string(hash), role)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			logger.Warn("Registration failed: phone already exists",
				zap.String("phone", req.GetPhone()),
			)
			return nil, apperror.NewConflictError("Номер телефона уже зарегистрирован").ToGRPCError()
		}

		logger.Error("Database error during registration",
			zap.String("phone", req.GetPhone()),
			zap.Error(err),
		)
		return nil, apperror.NewDatabaseError("создание пользователя", err).ToGRPCError()
	}

	user := models.User{
		ID:        userID,
		FullName:  req.GetFullName(),
		Phone:     req.GetPhone(),
		Role:      role,
		HasPin:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	access, refresh, err := s.issueTokens(user.ID, user.Phone, user.Role)
	if err != nil {
		logger.Error("Failed to issue tokens after registration",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, apperror.NewInternalError("Не удалось создать токен", err).ToGRPCError()
	}

	logger.Info("Registration successful",
		zap.String("user_id", userID),
		zap.String("phone", req.GetPhone()),
		zap.String("role", role),
	)

	return &pb.RegisterResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         mapper.ToPBUser(&user),
	}, nil
}

func (s *AuthServiceServer) VerifyOTP(ctx context.Context, req *pb.VerifyOTPRequest) (*pb.VerifyOTPResponse, error) {
	// In the REST layer OTP is tracked in memory; here we assume verification
	// is handled upstream and just return a placeholder success to illustrate flow.
	if strings.TrimSpace(req.GetPhone()) == "" || strings.TrimSpace(req.GetCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "phone and code required")
	}
	return &pb.VerifyOTPResponse{
		Success: true,
		Message: "OTP verified (stub). Wire to SMS/Redis store for production.",
	}, nil
}

func (s *AuthServiceServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token required")
	}
	// For simplicity, treat refresh token as JWT with same secret and claims.
	token, err := jwt.Parse(req.GetRefreshToken(), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Error(codes.Unauthenticated, "unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	userID, _ := claims["user_id"].(string)
	phone, _ := claims["phone"].(string)
	role, _ := claims["role"].(string)
	access, refresh, err := s.issueTokens(userID, phone, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to issue token: %v", err)
	}
	return &pb.RefreshTokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *AuthServiceServer) issueTokens(userID, phone, role string) (string, string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"phone":   phone,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := access.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// refresh token with longer expiry
	claims["exp"] = time.Now().Add(24 * time.Hour * 7).Unix()
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshToken, err := refresh.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// AuthFromContext is a helper for service methods needing user context.
func AuthFromContext(ctx context.Context) *middleware.AuthContext {
	return middleware.GetAuthContext(ctx)
}

