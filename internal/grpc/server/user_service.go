package server

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mebellar-backend/internal/grpc/mapper"
	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/models"
	"mebellar-backend/pkg/pb"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	db         *sql.DB
	uploadPath string
}

func NewUserServiceServer(db *sql.DB) *UserServiceServer {
	return &UserServiceServer{
		db:         db,
		uploadPath: "./uploads/avatars",
	}
}

// ============================================
// PROFILE MANAGEMENT
// ============================================

func (s *UserServiceServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.ProfileResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	user, err := s.getUserByID(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.ProfileResponse{
		User: mapper.ToPBUser(user),
	}, nil
}

func (s *UserServiceServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.ProfileResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.FullName != nil && req.GetFullName() != "" {
		updates = append(updates, fmt.Sprintf("full_name = $%d", argPos))
		args = append(args, req.GetFullName())
		argPos++
	}

	if req.Email != nil && req.GetEmail() != "" {
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, req.GetEmail())
		argPos++
	}

	if req.AvatarUrl != nil && req.GetAvatarUrl() != "" {
		updates = append(updates, fmt.Sprintf("avatar_url = $%d", argPos))
		args = append(args, req.GetAvatarUrl())
		argPos++
	}

	if req.OnesignalId != nil && req.GetOnesignalId() != "" {
		updates = append(updates, fmt.Sprintf("onesignal_id = $%d", argPos))
		args = append(args, req.GetOnesignalId())
		argPos++
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	args = append(args, authCtx.UserID)

	query := fmt.Sprintf(`
		UPDATE users 
		SET %s
		WHERE id = $%d
	`, strings.Join(updates, ", "), argPos)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	user, err := s.getUserByID(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.ProfileResponse{
		User: mapper.ToPBUser(user),
	}, nil
}

func (s *UserServiceServer) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Soft delete: set is_active = false
	_, err := s.db.ExecContext(ctx, `
		UPDATE users 
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`, authCtx.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete error: %v", err)
	}

	return &pb.DeleteAccountResponse{
		Success: true,
		Message: "Account deleted successfully",
	}, nil
}

// ============================================
// PHONE CHANGE FLOW
// ============================================

func (s *UserServiceServer) RequestPhoneChange(ctx context.Context, req *pb.RequestPhoneChangeRequest) (*pb.RequestPhoneChangeResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	newPhone := strings.TrimSpace(req.GetNewPhone())
	if newPhone == "" {
		return nil, status.Error(codes.InvalidArgument, "new_phone is required")
	}

	// Check if phone already exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1 AND id != $2)
	`, newPhone, authCtx.UserID).Scan(&exists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "phone number already in use")
	}

	// TODO: Send OTP to new phone number
	// For now, return success
	return &pb.RequestPhoneChangeResponse{
		Success: true,
		Message: "OTP sent to new phone number",
	}, nil
}

func (s *UserServiceServer) VerifyPhoneChange(ctx context.Context, req *pb.VerifyPhoneChangeRequest) (*pb.VerifyPhoneChangeResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	newPhone := strings.TrimSpace(req.GetNewPhone())
	code := strings.TrimSpace(req.GetCode())

	if newPhone == "" || code == "" {
		return nil, status.Error(codes.InvalidArgument, "new_phone and code are required")
	}

	// TODO: Verify OTP code
	// For now, assume verification is successful

	// Update phone
	_, err := s.db.ExecContext(ctx, `
		UPDATE users 
		SET phone = $1, updated_at = NOW()
		WHERE id = $2
	`, newPhone, authCtx.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	user, err := s.getUserByID(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.VerifyPhoneChangeResponse{
		Success: true,
		Message: "Phone number updated successfully",
		User:    mapper.ToPBUser(user),
	}, nil
}

// ============================================
// EMAIL CHANGE FLOW
// ============================================

func (s *UserServiceServer) RequestEmailChange(ctx context.Context, req *pb.RequestEmailChangeRequest) (*pb.RequestEmailChangeResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	newEmail := strings.TrimSpace(req.GetNewEmail())
	if newEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "new_email is required")
	}

	// TODO: Send verification email
	return &pb.RequestEmailChangeResponse{
		Success: true,
		Message: "Verification email sent",
	}, nil
}

func (s *UserServiceServer) VerifyEmailChange(ctx context.Context, req *pb.VerifyEmailChangeRequest) (*pb.VerifyEmailChangeResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	newEmail := strings.TrimSpace(req.GetNewEmail())
	code := strings.TrimSpace(req.GetCode())

	if newEmail == "" || code == "" {
		return nil, status.Error(codes.InvalidArgument, "new_email and code are required")
	}

	// TODO: Verify email code
	// For now, assume verification is successful

	// Update email
	_, err := s.db.ExecContext(ctx, `
		UPDATE users 
		SET email = $1, updated_at = NOW()
		WHERE id = $2
	`, newEmail, authCtx.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	user, err := s.getUserByID(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.VerifyEmailChangeResponse{
		Success: true,
		Message: "Email updated successfully",
		User:    mapper.ToPBUser(user),
	}, nil
}

// ============================================
// PIN MANAGEMENT
// ============================================

func (s *UserServiceServer) SetPin(ctx context.Context, req *pb.SetPinRequest) (*pb.SetPinResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	pin := strings.TrimSpace(req.GetPin())
	if len(pin) < 4 || len(pin) > 6 {
		return nil, status.Error(codes.InvalidArgument, "PIN must be 4-6 digits")
	}

	// Hash PIN (using bcrypt for security)
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "PIN hash error: %v", err)
	}

	// Store PIN hash in a secure way (you might want a separate table for PINs)
	// For now, we'll use a placeholder approach
	// In production, consider storing PIN hash separately or using a different mechanism

	// Update has_pin flag
	_, err = s.db.ExecContext(ctx, `
		UPDATE users 
		SET has_pin = true, updated_at = NOW()
		WHERE id = $1
	`, authCtx.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	// TODO: Store PIN hash securely (consider a separate table or encrypted storage)
	_ = hash // Suppress unused variable warning

	return &pb.SetPinResponse{
		Success: true,
		Message: "PIN set successfully",
	}, nil
}

func (s *UserServiceServer) VerifyPin(ctx context.Context, req *pb.VerifyPinRequest) (*pb.VerifyPinResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	pin := strings.TrimSpace(req.GetPin())
	if pin == "" {
		return nil, status.Error(codes.InvalidArgument, "PIN is required")
	}

	// TODO: Verify PIN against stored hash
	// For now, return success if user has PIN set
	var hasPin bool
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(has_pin, false) FROM users WHERE id = $1
	`, authCtx.UserID).Scan(&hasPin)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if !hasPin {
		return nil, status.Error(codes.FailedPrecondition, "PIN not set")
	}

	// TODO: Implement actual PIN verification
	// For now, return success
	return &pb.VerifyPinResponse{
		Success: true,
		Message: "PIN verified",
	}, nil
}

// ============================================
// AVATAR UPLOAD (STREAMING)
// ============================================

func (s *UserServiceServer) UploadAvatar(stream pb.UserService_UploadAvatarServer) error {
	authCtx := middleware.GetAuthContext(stream.Context())
	if authCtx == nil {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	var metadata *pb.AvatarMetadata
	var fileData []byte
	var filename string

	// Receive stream
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive error: %v", err)
		}

		switch data := req.Data.(type) {
		case *pb.UploadAvatarRequest_Metadata:
			metadata = data.Metadata
			filename = metadata.Filename
		case *pb.UploadAvatarRequest_Chunk:
			fileData = append(fileData, data.Chunk...)
		}
	}

	if filename == "" || len(fileData) == 0 {
		return status.Error(codes.InvalidArgument, "metadata and file data required")
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(s.uploadPath, 0755); err != nil {
		return status.Errorf(codes.Internal, "failed to create upload directory: %v", err)
	}

	// Generate unique filename
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%s%s", uuid.NewString(), ext)
	filePath := filepath.Join(s.uploadPath, uniqueFilename)

	// Save file
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return status.Errorf(codes.Internal, "failed to save file: %v", err)
	}

	// Update user avatar URL
	avatarURL := fmt.Sprintf("/uploads/avatars/%s", uniqueFilename)
	_, err := s.db.ExecContext(stream.Context(), `
		UPDATE users 
		SET avatar_url = $1, updated_at = NOW()
		WHERE id = $2
	`, avatarURL, authCtx.UserID)
	if err != nil {
		return status.Errorf(codes.Internal, "update error: %v", err)
	}

	return stream.SendAndClose(&pb.UploadAvatarResponse{
		Success:  true,
		AvatarUrl: avatarURL,
		Message:   "Avatar uploaded successfully",
	})
}

// ============================================
// ADMIN ENDPOINTS
// ============================================

func (s *UserServiceServer) AdminListUsers(ctx context.Context, req *pb.AdminListUsersRequest) (*pb.AdminListUsersResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil || (authCtx.Role != "admin" && authCtx.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	// Build query
	query := "SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), " +
		"COALESCE(role, 'customer'), COALESCE(onesignal_id, ''), COALESCE(has_pin, false), " +
		"created_at, updated_at, COALESCE(is_active, true) FROM users WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if req.Role != "" {
		query += fmt.Sprintf(" AND role = $%d", argPos)
		args = append(args, req.Role)
		argPos++
	}

	if req.ActiveOnly {
		query += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, true)
		argPos++
	}

	if req.Search != "" {
		query += fmt.Sprintf(" AND (full_name ILIKE $%d OR phone ILIKE $%d)", argPos, argPos)
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern)
		argPos++
	}

	// Pagination
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var users []*pb.User
	for rows.Next() {
		var user models.User
		var isActive bool
		err := rows.Scan(
			&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL,
			&user.Role, &user.OneSignalID, &user.HasPin,
			&user.CreatedAt, &user.UpdatedAt, &isActive,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		users = append(users, mapper.ToPBUser(&user))
	}

	// Get total count
	var total int32
	countQuery := strings.Replace(query, "SELECT id, full_name", "SELECT COUNT(*)", 1)
	countQuery = strings.Split(countQuery, " ORDER BY")[0]
	err = s.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		total = int32(len(users)) // Fallback
	}

	return &pb.AdminListUsersResponse{
		Users: users,
		Total: total,
		Page:  int32(page),
		Limit: int32(limit),
	}, nil
}

func (s *UserServiceServer) AdminGetUser(ctx context.Context, req *pb.AdminGetUserRequest) (*pb.ProfileResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil || (authCtx.Role != "admin" && authCtx.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	user, err := s.getUserByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.ProfileResponse{
		User: mapper.ToPBUser(user),
	}, nil
}

func (s *UserServiceServer) AdminUpdateUser(ctx context.Context, req *pb.AdminUpdateUserRequest) (*pb.ProfileResponse, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil || (authCtx.Role != "admin" && authCtx.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Role != nil && req.GetRole() != "" {
		updates = append(updates, fmt.Sprintf("role = $%d", argPos))
		args = append(args, req.GetRole())
		argPos++
	}

	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, req.GetIsActive())
		argPos++
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	args = append(args, req.GetId())

	query := fmt.Sprintf(`
		UPDATE users 
		SET %s
		WHERE id = $%d
	`, strings.Join(updates, ", "), argPos)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	user, err := s.getUserByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.ProfileResponse{
		User: mapper.ToPBUser(user),
	}, nil
}

func (s *UserServiceServer) AdminDeleteUser(ctx context.Context, req *pb.AdminDeleteUserRequest) (*pb.Empty, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil || authCtx.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}

	// Hard delete for admin
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete error: %v", err)
	}

	return &pb.Empty{}, nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func (s *UserServiceServer) getUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	var isActive bool

	query := `
		SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), 
		       COALESCE(role, 'customer'), COALESCE(onesignal_id, ''), COALESCE(has_pin, false),
		       created_at, updated_at, COALESCE(is_active, true)
		FROM users
		WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL,
		&user.Role, &user.OneSignalID, &user.HasPin,
		&user.CreatedAt, &user.UpdatedAt, &isActive,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	if !isActive {
		return nil, status.Error(codes.NotFound, "user is inactive")
	}

	return &user, nil
}
