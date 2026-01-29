package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/pkg/cache"
	"mebellar-backend/pkg/logger"
	"mebellar-backend/pkg/pb"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CategoryServiceServer struct {
	pb.UnimplementedCategoryServiceServer
	db    *sql.DB
	cache cache.Cache
}

func NewCategoryServiceServer(db *sql.DB, cache cache.Cache) *CategoryServiceServer {
	return &CategoryServiceServer{
		db:    db,
		cache: cache,
	}
}

// ============================================
// CATEGORY CRUD
// ============================================

func (s *CategoryServiceServer) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	// Попытка получить из кэша
	cacheKey := fmt.Sprintf("categories:list:active_%v", req.GetActiveOnly())
	var categories []*pb.Category

	if err := s.cache.Get(cacheKey, &categories); err == nil {
		logger.Debug("Categories loaded from cache",
			zap.String("cache_key", cacheKey),
			zap.Int("count", len(categories)),
		)
		return &pb.ListCategoriesResponse{
			Categories: categories,
		}, nil
	}

	// Cache miss - загружаем из БД
	logger.Debug("Categories cache miss, loading from database")

	where := "1=1"
	if req.GetActiveOnly() {
		where = "is_active = true"
	}

	query := fmt.Sprintf(`
		SELECT id, parent_id, name, slug, icon_url, is_active, sort_order,
			   (SELECT COUNT(*) FROM products WHERE category_id = categories.id AND is_active = true) as product_count
		FROM categories
		WHERE %s
		ORDER BY sort_order, id
	`, where)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	// Build flat list first
	categoryMap := make(map[string]*pb.Category)
	var rootCategories []*pb.Category

	for rows.Next() {
		var id string
		var parentID sql.NullString
		var nameJSON []byte
		var slug, iconURL string
		var isActive bool
		var sortOrder, productCount int32

		if err := rows.Scan(&id, &parentID, &nameJSON, &slug, &iconURL, &isActive, &sortOrder, &productCount); err != nil {
			continue
		}

		nameMap := make(map[string]string)
		json.Unmarshal(nameJSON, &nameMap)

		cat := &pb.Category{
			Id:           id,
			Name:         mapToLocalizedString(nameMap),
			Slug:         slug,
			IconUrl:      iconURL,
			IsActive:     isActive,
			SortOrder:    sortOrder,
			ProductCount: productCount,
		}

		if parentID.Valid {
			cat.ParentId = parentID.String
		}

		categoryMap[id] = cat

		if !parentID.Valid {
			rootCategories = append(rootCategories, cat)
		}
	}

	// Build tree structure
	for _, cat := range categoryMap {
		if cat.ParentId != "" {
			if parent, ok := categoryMap[cat.ParentId]; ok {
				parent.SubCategories = append(parent.SubCategories, cat)
			}
		}
	}

	// Load attributes if requested
	if req.GetIncludeAttributes() {
		for _, cat := range categoryMap {
			attrs, _ := s.getCategoryAttributes(ctx, cat.Id)
			cat.Attributes = attrs
		}
	}

	// Сохраняем в кэш на 1 час
	if err := s.cache.Set(cacheKey, rootCategories, 1*time.Hour); err != nil {
		logger.Warn("Failed to cache categories", zap.Error(err))
	}

	return &pb.ListCategoriesResponse{
		Categories: rootCategories,
		Count:      int32(len(categoryMap)),
	}, nil
}

func (s *CategoryServiceServer) ListFlatCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListFlatCategoriesResponse, error) {
	where := "1=1"
	if req.GetActiveOnly() {
		where = "c.is_active = true"
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.parent_id, c.name, c.icon_url, c.is_active, c.sort_order,
			   p.name as parent_name,
			   (SELECT COUNT(*) FROM products WHERE category_id = c.id AND is_active = true) as product_count
		FROM categories c
		LEFT JOIN categories p ON c.parent_id = p.id
		WHERE %s
		ORDER BY c.sort_order, c.id
	`, where)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var categories []*pb.FlatCategory
	for rows.Next() {
		var id string
		var parentID sql.NullString
		var nameJSON []byte
		var iconURL string
		var isActive bool
		var sortOrder, productCount int32
		var parentNameJSON []byte

		if err := rows.Scan(&id, &parentID, &nameJSON, &iconURL, &isActive, &sortOrder, &parentNameJSON, &productCount); err != nil {
			continue
		}

		nameMap := make(map[string]string)
		json.Unmarshal(nameJSON, &nameMap)

		cat := &pb.FlatCategory{
			Id:           id,
			Name:         mapToLocalizedString(nameMap),
			IconUrl:      iconURL,
			IsActive:     isActive,
			SortOrder:    sortOrder,
			ProductCount: productCount,
		}

		if parentID.Valid {
			cat.ParentId = parentID.String
			parentNameMap := make(map[string]string)
			json.Unmarshal(parentNameJSON, &parentNameMap)
			if name, ok := parentNameMap["uz"]; ok {
				cat.ParentName = name
			}
		}

		categories = append(categories, cat)
	}

	return &pb.ListFlatCategoriesResponse{
		Categories: categories,
		Count:      int32(len(categories)),
	}, nil
}

func (s *CategoryServiceServer) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.CategoryResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "category id is required")
	}

	var id string
	var parentID sql.NullString
	var nameJSON []byte
	var slug, iconURL string
	var isActive bool
	var sortOrder, productCount int32

	err := s.db.QueryRowContext(ctx, `
		SELECT id, parent_id, name, slug, icon_url, is_active, sort_order,
			   (SELECT COUNT(*) FROM products WHERE category_id = categories.id AND is_active = true) as product_count
		FROM categories
		WHERE id = $1
	`, req.GetId()).Scan(&id, &parentID, &nameJSON, &slug, &iconURL, &isActive, &sortOrder, &productCount)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "category not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	nameMap := make(map[string]string)
	json.Unmarshal(nameJSON, &nameMap)

	cat := &pb.Category{
		Id:           id,
		Name:         mapToLocalizedString(nameMap),
		Slug:         slug,
		IconUrl:      iconURL,
		IsActive:     isActive,
		SortOrder:    sortOrder,
		ProductCount: productCount,
	}

	if parentID.Valid {
		cat.ParentId = parentID.String
	}

	// Load subcategories
	subRows, _ := s.db.QueryContext(ctx, `
		SELECT id, name, slug, icon_url, is_active, sort_order,
			   (SELECT COUNT(*) FROM products WHERE category_id = categories.id AND is_active = true) as product_count
		FROM categories WHERE parent_id = $1 ORDER BY sort_order, id
	`, id)
	if subRows != nil {
		defer subRows.Close()
		for subRows.Next() {
			var subID string
			var subNameJSON []byte
			var subSlug, subIconURL string
			var subIsActive bool
			var subSortOrder, subProductCount int32

			if err := subRows.Scan(&subID, &subNameJSON, &subSlug, &subIconURL, &subIsActive, &subSortOrder, &subProductCount); err != nil {
				continue
			}

			subNameMap := make(map[string]string)
			json.Unmarshal(subNameJSON, &subNameMap)

			cat.SubCategories = append(cat.SubCategories, &pb.Category{
				Id:           subID,
				ParentId:     id,
				Name:         mapToLocalizedString(subNameMap),
				Slug:         subSlug,
				IconUrl:      subIconURL,
				IsActive:     subIsActive,
				SortOrder:    subSortOrder,
				ProductCount: subProductCount,
			})
		}
	}

	// Load attributes if requested
	if req.GetIncludeAttributes() {
		cat.Attributes, _ = s.getCategoryAttributes(ctx, id)
	}

	return &pb.CategoryResponse{Category: cat}, nil
}

func (s *CategoryServiceServer) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CategoryResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	id := uuid.NewString()
	nameJSON, _ := json.Marshal(localizedStringToMap(req.GetName()))

	slug := req.GetSlug()
	if slug == "" {
		// Generate slug from name
		if name := req.GetName(); name != nil {
			slug = generateSlug(name.GetUz())
			if slug == "" {
				slug = generateSlug(name.GetEn())
			}
		}
	}

	var parentID interface{}
	if req.GetParentId() != "" {
		parentID = req.GetParentId()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO categories (id, parent_id, name, slug, icon_url, is_active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, parentID, nameJSON, slug, req.GetIconUrl(), req.GetIsActive(), req.GetSortOrder())

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, status.Error(codes.AlreadyExists, "category with this slug already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create category: %v", err)
	}

	return s.GetCategory(ctx, &pb.GetCategoryRequest{Id: id})
}

func (s *CategoryServiceServer) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.CategoryResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "category id is required")
	}

	// Build dynamic update
	updates := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.ParentId != nil {
		updates = append(updates, fmt.Sprintf("parent_id = $%d", argIdx))
		if *req.ParentId == "" {
			args = append(args, nil)
		} else {
			args = append(args, *req.ParentId)
		}
		argIdx++
	}
	if req.Name != nil {
		nameJSON, _ := json.Marshal(localizedStringToMap(req.Name))
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, nameJSON)
		argIdx++
	}
	if req.Slug != nil {
		updates = append(updates, fmt.Sprintf("slug = $%d", argIdx))
		args = append(args, *req.Slug)
		argIdx++
	}
	if req.IconUrl != nil {
		updates = append(updates, fmt.Sprintf("icon_url = $%d", argIdx))
		args = append(args, *req.IconUrl)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(updates) == 0 {
		return s.GetCategory(ctx, &pb.GetCategoryRequest{Id: req.GetId()})
	}

	args = append(args, req.GetId())
	query := fmt.Sprintf("UPDATE categories SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update category: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "category not found")
	}

	return s.GetCategory(ctx, &pb.GetCategoryRequest{Id: req.GetId()})
}

func (s *CategoryServiceServer) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "category id is required")
	}

	// Check for subcategories
	var subCount int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM categories WHERE parent_id = $1", req.GetId()).Scan(&subCount)
	if subCount > 0 {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete category with subcategories")
	}

	// Check for products
	var productCount int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products WHERE category_id = $1", req.GetId()).Scan(&productCount)
	if productCount > 0 {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete category with products")
	}

	// Delete attributes first
	s.db.ExecContext(ctx, "DELETE FROM category_attributes WHERE category_id = $1", req.GetId())

	// Delete category
	result, err := s.db.ExecContext(ctx, "DELETE FROM categories WHERE id = $1", req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete category: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "category not found")
	}

	return &pb.Empty{}, nil
}

// ============================================
// CATEGORY ATTRIBUTES
// ============================================

func (s *CategoryServiceServer) ListCategoryAttributes(ctx context.Context, req *pb.ListCategoryAttributesRequest) (*pb.ListCategoryAttributesResponse, error) {
	if req.GetCategoryId() == "" {
		return nil, status.Error(codes.InvalidArgument, "category_id is required")
	}

	attrs, err := s.getCategoryAttributes(ctx, req.GetCategoryId())
	if err != nil {
		return nil, err
	}

	return &pb.ListCategoryAttributesResponse{
		Attributes: attrs,
		Count:      int32(len(attrs)),
	}, nil
}

func (s *CategoryServiceServer) GetCategoryAttribute(ctx context.Context, req *pb.GetCategoryAttributeRequest) (*pb.CategoryAttributeResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "attribute id is required")
	}

	attr, err := s.getAttributeByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.CategoryAttributeResponse{Attribute: attr}, nil
}

func (s *CategoryServiceServer) CreateCategoryAttribute(ctx context.Context, req *pb.CreateCategoryAttributeRequest) (*pb.CategoryAttributeResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetCategoryId() == "" || req.GetKey() == "" || req.GetType() == "" {
		return nil, status.Error(codes.InvalidArgument, "category_id, key, and type are required")
	}

	// Validate type
	validTypes := []string{"text", "number", "dropdown", "switch"}
	validType := false
	for _, t := range validTypes {
		if req.GetType() == t {
			validType = true
			break
		}
	}
	if !validType {
		return nil, status.Error(codes.InvalidArgument, "invalid type. Valid: text, number, dropdown, switch")
	}

	id := uuid.NewString()
	labelJSON, _ := json.Marshal(localizedStringToMap(req.GetLabel()))
	optionsJSON, _ := json.Marshal(attributeOptionsToSlice(req.GetOptions()))

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO category_attributes (id, category_id, key, type, label, options, is_required, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`, id, req.GetCategoryId(), req.GetKey(), req.GetType(), labelJSON, optionsJSON, req.GetIsRequired(), req.GetSortOrder())

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, status.Error(codes.AlreadyExists, "attribute with this key already exists for this category")
		}
		return nil, status.Errorf(codes.Internal, "failed to create attribute: %v", err)
	}

	return s.GetCategoryAttribute(ctx, &pb.GetCategoryAttributeRequest{Id: id})
}

func (s *CategoryServiceServer) UpdateCategoryAttribute(ctx context.Context, req *pb.UpdateCategoryAttributeRequest) (*pb.CategoryAttributeResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "attribute id is required")
	}

	updates := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if req.Key != nil {
		updates = append(updates, fmt.Sprintf("key = $%d", argIdx))
		args = append(args, *req.Key)
		argIdx++
	}
	if req.Type != nil {
		updates = append(updates, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.Label != nil {
		labelJSON, _ := json.Marshal(localizedStringToMap(req.Label))
		updates = append(updates, fmt.Sprintf("label = $%d", argIdx))
		args = append(args, labelJSON)
		argIdx++
	}
	if len(req.GetOptions()) > 0 {
		optionsJSON, _ := json.Marshal(attributeOptionsToSlice(req.GetOptions()))
		updates = append(updates, fmt.Sprintf("options = $%d", argIdx))
		args = append(args, optionsJSON)
		argIdx++
	}
	if req.IsRequired != nil {
		updates = append(updates, fmt.Sprintf("is_required = $%d", argIdx))
		args = append(args, *req.IsRequired)
		argIdx++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	args = append(args, req.GetId())
	query := fmt.Sprintf("UPDATE category_attributes SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update attribute: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "attribute not found")
	}

	return s.GetCategoryAttribute(ctx, &pb.GetCategoryAttributeRequest{Id: req.GetId()})
}

func (s *CategoryServiceServer) DeleteCategoryAttribute(ctx context.Context, req *pb.DeleteCategoryAttributeRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin or moderator role required")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "attribute id is required")
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM category_attributes WHERE id = $1", req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete attribute: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "attribute not found")
	}

	return &pb.Empty{}, nil
}

// ============================================
// HELPER METHODS
// ============================================

func (s *CategoryServiceServer) getCategoryAttributes(ctx context.Context, categoryID string) ([]*pb.CategoryAttribute, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, category_id, key, type, label, options, is_required, sort_order, created_at, updated_at
		FROM category_attributes
		WHERE category_id = $1
		ORDER BY sort_order, id
	`, categoryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var attrs []*pb.CategoryAttribute
	for rows.Next() {
		attr, err := s.scanAttribute(rows)
		if err != nil {
			continue
		}
		attrs = append(attrs, attr)
	}

	return attrs, nil
}

func (s *CategoryServiceServer) getAttributeByID(ctx context.Context, id string) (*pb.CategoryAttribute, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, category_id, key, type, label, options, is_required, sort_order, created_at, updated_at
		FROM category_attributes
		WHERE id = $1
	`, id)

	attr, err := s.scanAttributeRow(row)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "attribute not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	return attr, nil
}

func (s *CategoryServiceServer) scanAttribute(rows *sql.Rows) (*pb.CategoryAttribute, error) {
	var id, categoryID, key, attrType string
	var labelJSON, optionsJSON []byte
	var isRequired bool
	var sortOrder int32
	var createdAt, updatedAt sql.NullTime

	if err := rows.Scan(&id, &categoryID, &key, &attrType, &labelJSON, &optionsJSON, &isRequired, &sortOrder, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return s.buildAttribute(id, categoryID, key, attrType, labelJSON, optionsJSON, isRequired, sortOrder, createdAt, updatedAt), nil
}

func (s *CategoryServiceServer) scanAttributeRow(row *sql.Row) (*pb.CategoryAttribute, error) {
	var id, categoryID, key, attrType string
	var labelJSON, optionsJSON []byte
	var isRequired bool
	var sortOrder int32
	var createdAt, updatedAt sql.NullTime

	if err := row.Scan(&id, &categoryID, &key, &attrType, &labelJSON, &optionsJSON, &isRequired, &sortOrder, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return s.buildAttribute(id, categoryID, key, attrType, labelJSON, optionsJSON, isRequired, sortOrder, createdAt, updatedAt), nil
}

func (s *CategoryServiceServer) buildAttribute(id, categoryID, key, attrType string, labelJSON, optionsJSON []byte, isRequired bool, sortOrder int32, createdAt, updatedAt sql.NullTime) *pb.CategoryAttribute {
	labelMap := make(map[string]string)
	json.Unmarshal(labelJSON, &labelMap)

	var optionsSlice []map[string]interface{}
	json.Unmarshal(optionsJSON, &optionsSlice)

	attr := &pb.CategoryAttribute{
		Id:         id,
		CategoryId: categoryID,
		Key:        key,
		Type:       attrType,
		Label:      mapToLocalizedString(labelMap),
		IsRequired: isRequired,
		SortOrder:  sortOrder,
	}

	for _, opt := range optionsSlice {
		option := &pb.AttributeOption{}
		if v, ok := opt["value"].(string); ok {
			option.Value = v
		}
		if label, ok := opt["label"].(map[string]interface{}); ok {
			labelStr := make(map[string]string)
			for k, v := range label {
				if s, ok := v.(string); ok {
					labelStr[k] = s
				}
			}
			option.Label = mapToLocalizedString(labelStr)
		}
		attr.Options = append(attr.Options, option)
	}

	if createdAt.Valid {
		attr.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		attr.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	return attr
}

func attributeOptionsToSlice(options []*pb.AttributeOption) []map[string]interface{} {
	result := make([]map[string]interface{}, len(options))
	for i, opt := range options {
		result[i] = map[string]interface{}{
			"value": opt.GetValue(),
			"label": localizedStringToMap(opt.GetLabel()),
		}
	}
	return result
}

func generateSlug(name string) string {
	if name == "" {
		return ""
	}
	slug := ""
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			slug += string(r)
		case r >= 'A' && r <= 'Z':
			slug += string(r + 32)
		case r >= '0' && r <= '9':
			slug += string(r)
		case r == ' ' || r == '-' || r == '_':
			if len(slug) > 0 && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}
	slug = strings.Trim(slug, "-")
	return slug
}
