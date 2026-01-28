package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"mebellar-backend/internal/grpc/mapper"
	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/models"
	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/sms"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	phone := strings.TrimSpace(req.GetPhone())
	if phone == "" {
		return nil, status.Error(codes.InvalidArgument, "phone is required")
	}

	// Validate phone format (+998XXXXXXXXX)
	phonePattern := `^\+998[0-9]{9}$`
	matched, _ := regexp.MatchString(phonePattern, phone)
	if !matched {
		return nil, status.Error(codes.InvalidArgument, "invalid phone format. Use +998XXXXXXXXX")
	}

	// Check if phone already exists and is active
	var existingID string
	var isActive bool
	err := s.db.QueryRowContext(ctx, "SELECT id, COALESCE(is_active, true) FROM users WHERE phone = $1", phone).Scan(&existingID, &isActive)
	if err == nil && isActive {
		return nil, status.Error(codes.AlreadyExists, "phone number already registered")
	}

	// Generate OTP (5 digits)
	rand.Seed(time.Now().UnixNano())
	code := fmt.Sprintf("%05d", rand.Intn(100000))

	// Store OTP (in production, use Redis with TTL)
	// For now, this is a placeholder - you should implement proper OTP storage
	log.Printf("📱 [gRPC OTP] to %s: %s", phone, code)

	// Send real SMS via Eskiz
	if s.sms != nil {
		if err := s.sms.SendOTP(phone, code); err != nil {
			log.Printf("❌ [SMS ERROR] to %s: %v", phone, err)
			// We might want to return an error here, but for now let's just log it
			// or return a specific error if needed.
			// return nil, status.Errorf(codes.Internal, "failed to send SMS: %v", err)
		}
	}

	return &pb.SendOTPResponse{
		Success: true,
		Message: "Verification code sent",
	}, nil
}

func (s *AuthServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if strings.TrimSpace(req.GetPhone()) == "" || strings.TrimSpace(req.GetPassword()) == "" {
		return nil, status.Error(codes.InvalidArgument, "phone and password are required")
	}

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
	if errors.Is(err, sql.ErrNoRows) || !isActive {
		return nil, status.Error(codes.NotFound, "user not found or inactive")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	access, refresh, err := s.issueTokens(user.ID, user.Phone, user.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to issue token: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         mapper.ToPBUser(&user),
	}, nil
}

func (s *AuthServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if strings.TrimSpace(req.GetFullName()) == "" || strings.TrimSpace(req.GetPhone()) == "" || len(req.GetPassword()) < 6 {
		return nil, status.Error(codes.InvalidArgument, "full_name, phone and password(>=6) required")
	}

	role := req.GetRole()
	if role != "customer" && role != "seller" {
		role = "customer"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "password hash error: %v", err)
	}

	userID := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, full_name, phone, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
	`, userID, req.GetFullName(), req.GetPhone(), string(hash), role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Errorf(codes.Internal, "insert error: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to issue token: %v", err)
	}

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

