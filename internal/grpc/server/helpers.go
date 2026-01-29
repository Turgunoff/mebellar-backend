package server

import (
	"context"
	"database/sql"

	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/pkg/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DeleteEntity is a generic helper function for deleting entities from the database
// It handles role-based authorization, validates the ID, executes the DELETE query,
// and checks if the entity was actually found and deleted.
//
// Parameters:
// - ctx: context
// - db: database connection
// - tableName: name of the table to delete from
// - idValue: the ID value to delete (supports both string and int64)
// - entityName: friendly name of the entity for error messages (e.g., "region", "banner")
func DeleteEntity(ctx context.Context, db *sql.DB, tableName string, idValue interface{}, entityName string) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	// Validate ID is not empty
	switch v := idValue.(type) {
	case string:
		if v == "" {
			return nil, status.Errorf(codes.InvalidArgument, "%s id is required", entityName)
		}
	case int64:
		if v == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "%s id is required", entityName)
		}
	}

	result, err := db.ExecContext(ctx, "DELETE FROM "+tableName+" WHERE id = $1", idValue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete %s: %v", entityName, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "%s not found", entityName)
	}

	return &pb.Empty{}, nil
}

// VerifyShopOwnershipHelper is a generic helper function to verify that a user owns a shop.
// It checks if the shop exists and if the user owns the associated seller profile.
//
// Parameters:
// - ctx: context
// - db: database connection
// - shopID: ID of the shop to verify
// - userID: ID of the user claiming ownership
//
// Returns:
// - error if shop is not found, user doesn't own it, or database error occurs
func VerifyShopOwnershipHelper(ctx context.Context, db *sql.DB, shopID, userID string) error {
	var sellerID string
	err := db.QueryRowContext(ctx, "SELECT seller_id FROM shops WHERE id = $1", shopID).Scan(&sellerID)
	if err == sql.ErrNoRows {
		return status.Error(codes.NotFound, "shop not found")
	}
	if err != nil {
		return status.Errorf(codes.Internal, "query error: %v", err)
	}

	var ownerUserID string
	err = db.QueryRowContext(ctx, "SELECT user_id FROM seller_profiles WHERE id = $1", sellerID).Scan(&ownerUserID)
	if err != nil || ownerUserID != userID {
		return status.Error(codes.PermissionDenied, "you don't own this shop")
	}
	return nil
}

// UpdateUserFieldHelper is a generic helper function for updating a single field in the users table.
// It updates the field value and updated_at timestamp.
//
// Parameters:
// - ctx: context
// - db: database connection
// - fieldName: name of the field to update (e.g., "phone", "email")
// - fieldValue: new value for the field
// - userID: ID of the user to update
//
// Returns:
// - error if the update fails
func UpdateUserFieldHelper(ctx context.Context, db *sql.DB, fieldName string, fieldValue interface{}, userID string) error {
	query := "UPDATE users SET " + fieldName + " = $1, updated_at = NOW() WHERE id = $2"
	_, err := db.ExecContext(ctx, query, fieldValue, userID)
	if err != nil {
		return status.Errorf(codes.Internal, "update error: %v", err)
	}
	return nil
}
