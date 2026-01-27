package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/pkg/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CommonServiceServer struct {
	pb.UnimplementedCommonServiceServer
	db *sql.DB
}

func NewCommonServiceServer(db *sql.DB) *CommonServiceServer {
	return &CommonServiceServer{db: db}
}

// ============================================
// REGIONS
// ============================================

func (s *CommonServiceServer) ListRegions(ctx context.Context, req *pb.ListRegionsRequest) (*pb.ListRegionsResponse, error) {
	where := "1=1"
	if req.GetActiveOnly() {
		where = "is_active = true"
	}

	query := fmt.Sprintf(`
		SELECT id, name, COALESCE(name_jsonb, '{}'), code, is_active, ordering, created_at, updated_at
		FROM regions WHERE %s ORDER BY ordering, id
	`, where)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var regions []*pb.Region
	for rows.Next() {
		var id int32
		var name string
		var nameJSONB []byte
		var code sql.NullString
		var isActive bool
		var ordering int32
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&id, &name, &nameJSONB, &code, &isActive, &ordering, &createdAt, &updatedAt); err != nil {
			continue
		}

		nameMap := make(map[string]string)
		json.Unmarshal(nameJSONB, &nameMap)

		region := &pb.Region{
			Id:             id,
			Name:           name,
			NameLocalized:  mapToLocalizedString(nameMap),
			IsActive:       isActive,
			Ordering:       ordering,
		}
		if code.Valid {
			region.Code = code.String
		}
		if createdAt.Valid {
			region.CreatedAt = timestamppb.New(createdAt.Time)
		}
		if updatedAt.Valid {
			region.UpdatedAt = timestamppb.New(updatedAt.Time)
		}

		regions = append(regions, region)
	}

	return &pb.ListRegionsResponse{
		Regions: regions,
		Count:   int32(len(regions)),
	}, nil
}

func (s *CommonServiceServer) GetRegion(ctx context.Context, req *pb.GetRegionRequest) (*pb.RegionResponse, error) {
	if req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "region id is required")
	}

	var id int32
	var name string
	var nameJSONB []byte
	var code sql.NullString
	var isActive bool
	var ordering int32
	var createdAt, updatedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(name_jsonb, '{}'), code, is_active, ordering, created_at, updated_at
		FROM regions WHERE id = $1
	`, req.GetId()).Scan(&id, &name, &nameJSONB, &code, &isActive, &ordering, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "region not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	nameMap := make(map[string]string)
	json.Unmarshal(nameJSONB, &nameMap)

	region := &pb.Region{
		Id:            id,
		Name:          name,
		NameLocalized: mapToLocalizedString(nameMap),
		IsActive:      isActive,
		Ordering:      ordering,
	}
	if code.Valid {
		region.Code = code.String
	}
	if createdAt.Valid {
		region.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		region.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	return &pb.RegionResponse{Region: region}, nil
}

func (s *CommonServiceServer) CreateRegion(ctx context.Context, req *pb.CreateRegionRequest) (*pb.RegionResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	nameJSON, _ := json.Marshal(localizedStringToMap(req.GetName()))
	legacyName := req.GetName().GetUz()
	if legacyName == "" {
		legacyName = req.GetName().GetEn()
	}

	var id int32
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO regions (name, name_jsonb, code, is_active, ordering, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, legacyName, nameJSON, req.GetCode(), req.GetIsActive(), req.GetOrdering()).Scan(&id)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, status.Error(codes.AlreadyExists, "region with this code already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create region: %v", err)
	}

	return s.GetRegion(ctx, &pb.GetRegionRequest{Id: id})
}

func (s *CommonServiceServer) UpdateRegion(ctx context.Context, req *pb.UpdateRegionRequest) (*pb.RegionResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "region id is required")
	}

	updates := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		nameJSON, _ := json.Marshal(localizedStringToMap(req.Name))
		updates = append(updates, fmt.Sprintf("name_jsonb = $%d", argIdx))
		args = append(args, nameJSON)
		argIdx++

		legacyName := req.Name.GetUz()
		if legacyName == "" {
			legacyName = req.Name.GetEn()
		}
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, legacyName)
		argIdx++
	}
	if req.Code != nil {
		updates = append(updates, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *req.Code)
		argIdx++
	}
	if req.Ordering != nil {
		updates = append(updates, fmt.Sprintf("ordering = $%d", argIdx))
		args = append(args, *req.Ordering)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	args = append(args, req.GetId())
	query := fmt.Sprintf("UPDATE regions SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update region: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "region not found")
	}

	return s.GetRegion(ctx, &pb.GetRegionRequest{Id: req.GetId()})
}

func (s *CommonServiceServer) DeleteRegion(ctx context.Context, req *pb.DeleteRegionRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "region id is required")
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM regions WHERE id = $1", req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete region: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "region not found")
	}

	return &pb.Empty{}, nil
}

// ============================================
// BANNERS
// ============================================

func (s *CommonServiceServer) ListBanners(ctx context.Context, req *pb.ListBannersRequest) (*pb.ListBannersResponse, error) {
	where := "1=1"
	if req.GetActiveOnly() {
		where = "is_active = true"
	}

	query := fmt.Sprintf(`
		SELECT id, title, subtitle, image_url, target_type, target_id, sort_order, is_active, created_at
		FROM banners WHERE %s ORDER BY sort_order, created_at DESC
	`, where)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var banners []*pb.Banner
	for rows.Next() {
		var id string
		var title, subtitle []byte
		var imageURL, targetType string
		var targetID sql.NullString
		var sortOrder int32
		var isActive bool
		var createdAt sql.NullTime

		if err := rows.Scan(&id, &title, &subtitle, &imageURL, &targetType, &targetID, &sortOrder, &isActive, &createdAt); err != nil {
			continue
		}

		titleMap := make(map[string]string)
		json.Unmarshal(title, &titleMap)
		subtitleMap := make(map[string]string)
		json.Unmarshal(subtitle, &subtitleMap)

		banner := &pb.Banner{
			Id:         id,
			Title:      mapToLocalizedString(titleMap),
			Subtitle:   mapToLocalizedString(subtitleMap),
			ImageUrl:   imageURL,
			TargetType: targetType,
			SortOrder:  sortOrder,
			IsActive:   isActive,
		}
		if targetID.Valid {
			banner.TargetId = targetID.String
		}
		if createdAt.Valid {
			banner.CreatedAt = timestamppb.New(createdAt.Time)
		}

		banners = append(banners, banner)
	}

	return &pb.ListBannersResponse{
		Banners: banners,
		Count:   int32(len(banners)),
	}, nil
}

func (s *CommonServiceServer) GetBanner(ctx context.Context, req *pb.GetBannerRequest) (*pb.BannerResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "banner id is required")
	}

	var id string
	var title, subtitle []byte
	var imageURL, targetType string
	var targetID sql.NullString
	var sortOrder int32
	var isActive bool
	var createdAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, subtitle, image_url, target_type, target_id, sort_order, is_active, created_at
		FROM banners WHERE id = $1
	`, req.GetId()).Scan(&id, &title, &subtitle, &imageURL, &targetType, &targetID, &sortOrder, &isActive, &createdAt)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "banner not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	titleMap := make(map[string]string)
	json.Unmarshal(title, &titleMap)
	subtitleMap := make(map[string]string)
	json.Unmarshal(subtitle, &subtitleMap)

	banner := &pb.Banner{
		Id:         id,
		Title:      mapToLocalizedString(titleMap),
		Subtitle:   mapToLocalizedString(subtitleMap),
		ImageUrl:   imageURL,
		TargetType: targetType,
		SortOrder:  sortOrder,
		IsActive:   isActive,
	}
	if targetID.Valid {
		banner.TargetId = targetID.String
	}
	if createdAt.Valid {
		banner.CreatedAt = timestamppb.New(createdAt.Time)
	}

	return &pb.BannerResponse{Banner: banner}, nil
}

func (s *CommonServiceServer) CreateBanner(ctx context.Context, req *pb.CreateBannerRequest) (*pb.BannerResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	id := uuid.NewString()
	titleJSON, _ := json.Marshal(localizedStringToMap(req.GetTitle()))
	subtitleJSON, _ := json.Marshal(localizedStringToMap(req.GetSubtitle()))

	var targetID interface{}
	if req.GetTargetId() != "" {
		targetID = req.GetTargetId()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO banners (id, title, subtitle, image_url, target_type, target_id, sort_order, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, id, titleJSON, subtitleJSON, req.GetImageUrl(), req.GetTargetType(), targetID, req.GetSortOrder(), req.GetIsActive())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create banner: %v", err)
	}

	return s.GetBanner(ctx, &pb.GetBannerRequest{Id: id})
}

func (s *CommonServiceServer) UpdateBanner(ctx context.Context, req *pb.UpdateBannerRequest) (*pb.BannerResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "banner id is required")
	}

	updates := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Title != nil {
		titleJSON, _ := json.Marshal(localizedStringToMap(req.Title))
		updates = append(updates, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, titleJSON)
		argIdx++
	}
	if req.Subtitle != nil {
		subtitleJSON, _ := json.Marshal(localizedStringToMap(req.Subtitle))
		updates = append(updates, fmt.Sprintf("subtitle = $%d", argIdx))
		args = append(args, subtitleJSON)
		argIdx++
	}
	if req.ImageUrl != nil {
		updates = append(updates, fmt.Sprintf("image_url = $%d", argIdx))
		args = append(args, *req.ImageUrl)
		argIdx++
	}
	if req.TargetType != nil {
		updates = append(updates, fmt.Sprintf("target_type = $%d", argIdx))
		args = append(args, *req.TargetType)
		argIdx++
	}
	if req.TargetId != nil {
		updates = append(updates, fmt.Sprintf("target_id = $%d", argIdx))
		args = append(args, *req.TargetId)
		argIdx++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(updates) == 0 {
		return s.GetBanner(ctx, &pb.GetBannerRequest{Id: req.GetId()})
	}

	args = append(args, req.GetId())
	query := fmt.Sprintf("UPDATE banners SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update banner: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "banner not found")
	}

	return s.GetBanner(ctx, &pb.GetBannerRequest{Id: req.GetId()})
}

func (s *CommonServiceServer) DeleteBanner(ctx context.Context, req *pb.DeleteBannerRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "banner id is required")
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM banners WHERE id = $1", req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete banner: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "banner not found")
	}

	return &pb.Empty{}, nil
}

// ============================================
// CANCELLATION REASONS
// ============================================

func (s *CommonServiceServer) ListCancellationReasons(ctx context.Context, req *pb.ListCancellationReasonsRequest) (*pb.ListCancellationReasonsResponse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, text, is_active FROM cancellation_reasons WHERE is_active = true ORDER BY id
	`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var reasons []*pb.CancellationReason
	for rows.Next() {
		var id string
		var text []byte
		var isActive bool

		if err := rows.Scan(&id, &text, &isActive); err != nil {
			continue
		}

		textMap := make(map[string]string)
		json.Unmarshal(text, &textMap)

		reasons = append(reasons, &pb.CancellationReason{
			Id:       id,
			Text:     mapToLocalizedString(textMap),
			IsActive: isActive,
		})
	}

	return &pb.ListCancellationReasonsResponse{Reasons: reasons}, nil
}
