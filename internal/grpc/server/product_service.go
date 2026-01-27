package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/pkg/pb"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductServiceServer struct {
	pb.UnimplementedProductServiceServer
	db         *sql.DB
	uploadPath string
}

func NewProductServiceServer(db *sql.DB) *ProductServiceServer {
	return &ProductServiceServer{
		db:         db,
		uploadPath: "./uploads/products",
	}
}

// ============================================
// PUBLIC ENDPOINTS
// ============================================

func (s *ProductServiceServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	product, err := s.getProductByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Increment view count if requested
	if req.GetIncrementView() {
		go func() {
			s.db.Exec("UPDATE products SET view_count = view_count + 1 WHERE id = $1", req.GetId())
		}()
	}

	return &pb.ProductResponse{Product: product}, nil
}

func (s *ProductServiceServer) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	return s.listProductsInternal(ctx, req.GetFilters(), int(req.GetPage()), int(req.GetLimit()), "")
}

func (s *ProductServiceServer) ListNewArrivals(ctx context.Context, req *pb.ListNewArrivalsRequest) (*pb.ListProductsResponse, error) {
	filters := &pb.ProductFilters{
		IsNew:      true,
		IsActive:   true,
		CategoryId: req.GetCategoryId(),
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	return s.listProductsInternal(ctx, filters, 1, limit, "created_at DESC")
}

func (s *ProductServiceServer) ListPopularProducts(ctx context.Context, req *pb.ListPopularProductsRequest) (*pb.ListProductsResponse, error) {
	filters := &pb.ProductFilters{
		IsPopular:  true,
		IsActive:   true,
		CategoryId: req.GetCategoryId(),
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	return s.listProductsInternal(ctx, filters, 1, limit, "sold_count DESC, view_count DESC")
}

func (s *ProductServiceServer) ListProductsGroupedBySubcategory(ctx context.Context, req *pb.ListProductsGroupedBySubcategoryRequest) (*pb.ListProductsGroupedBySubcategoryResponse, error) {
	parentID := req.GetParentCategoryId()
	productsPerCategory := int(req.GetProductsPerCategory())
	if productsPerCategory <= 0 {
		productsPerCategory = 4
	}

	// Get subcategories
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug FROM categories 
		WHERE parent_id = $1 AND is_active = true 
		ORDER BY sort_order, id
	`, parentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch categories: %v", err)
	}
	defer rows.Close()

	var groups []*pb.CategoryProductsGroup
	for rows.Next() {
		var catID, slug string
		var nameJSON []byte
		if err := rows.Scan(&catID, &nameJSON, &slug); err != nil {
			continue
		}

		nameMap := make(map[string]string)
		json.Unmarshal(nameJSON, &nameMap)

		// Get products for this category
		filters := &pb.ProductFilters{CategoryId: catID, IsActive: true}
		resp, _ := s.listProductsInternal(ctx, filters, 1, productsPerCategory, "created_at DESC")

		if resp != nil && len(resp.Products) > 0 {
			groups = append(groups, &pb.CategoryProductsGroup{
				CategoryId:   catID,
				CategoryName: mapToLocalizedString(nameMap),
				CategorySlug: slug,
				Products:     resp.Products,
				Total:        resp.Total,
				HasMore:      resp.Total > int32(productsPerCategory),
			})
		}
	}

	return &pb.ListProductsGroupedBySubcategoryResponse{
		Groups: groups,
		Count:  int32(len(groups)),
	}, nil
}

// ============================================
// SELLER ENDPOINTS
// ============================================

func (s *ProductServiceServer) ListSellerProducts(ctx context.Context, req *pb.ListSellerProductsRequest) (*pb.ListProductsResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	shopID := req.GetShopId()
	if shopID == "" {
		shopID = auth.ShopID
	}
	if shopID == "" {
		return nil, status.Error(codes.InvalidArgument, "shop_id is required")
	}

	// Verify shop ownership
	if err := s.verifyShopOwnership(ctx, shopID, auth.UserID); err != nil {
		return nil, err
	}

	filters := req.GetFilters()
	if filters == nil {
		filters = &pb.ProductFilters{}
	}
	filters.ShopId = shopID

	return s.listProductsInternal(ctx, filters, int(req.GetPage()), int(req.GetLimit()), "")
}

func (s *ProductServiceServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if auth.Role != "seller" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "seller or admin role required")
	}

	shopID := req.GetShopId()
	if shopID == "" {
		return nil, status.Error(codes.InvalidArgument, "shop_id is required")
	}

	// Verify shop ownership (unless admin)
	if auth.Role != "admin" {
		if err := s.verifyShopOwnership(ctx, shopID, auth.UserID); err != nil {
			return nil, err
		}
	}

	productID := uuid.NewString()
	nameJSON, _ := json.Marshal(localizedStringToMap(req.GetName()))
	descJSON, _ := json.Marshal(localizedStringToMap(req.GetDescription()))
	specsJSON, _ := json.Marshal(structToMap(req.GetSpecs()))
	variantsJSON, _ := json.Marshal(variantsToSlice(req.GetVariants()))
	deliveryJSON, _ := json.Marshal(deliverySettingsToMap(req.GetDeliverySettings()))

	var discountPrice *float64
	if req.GetDiscountPrice() > 0 {
		dp := req.GetDiscountPrice()
		discountPrice = &dp
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO products (
			id, shop_id, category_id, name, description, price, discount_price,
			images, specs, variants, delivery_settings, is_new, is_popular, is_active,
			rating, view_count, sold_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 0, 0, 0, NOW())
	`, productID, shopID, nullString(req.GetCategoryId()), nameJSON, descJSON,
		req.GetPrice(), discountPrice, pq.Array(req.GetImages()), specsJSON, variantsJSON,
		deliveryJSON, req.GetIsNew(), req.GetIsPopular(), req.GetIsActive())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	return s.GetProduct(ctx, &pb.GetProductRequest{Id: productID})
}

func (s *ProductServiceServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	productID := req.GetId()
	if productID == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	// Get existing product to verify ownership
	var shopID string
	err := s.db.QueryRowContext(ctx, "SELECT shop_id FROM products WHERE id = $1", productID).Scan(&shopID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// Verify ownership (unless admin)
	if auth.Role != "admin" {
		if err := s.verifyShopOwnership(ctx, shopID, auth.UserID); err != nil {
			return nil, err
		}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.CategoryId != nil {
		updates = append(updates, fmt.Sprintf("category_id = $%d", argIdx))
		args = append(args, nullString(*req.CategoryId))
		argIdx++
	}
	if req.Name != nil {
		nameJSON, _ := json.Marshal(localizedStringToMap(req.Name))
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, nameJSON)
		argIdx++
	}
	if req.Description != nil {
		descJSON, _ := json.Marshal(localizedStringToMap(req.Description))
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, descJSON)
		argIdx++
	}
	if req.Price != nil {
		updates = append(updates, fmt.Sprintf("price = $%d", argIdx))
		args = append(args, *req.Price)
		argIdx++
	}
	if req.DiscountPrice != nil {
		updates = append(updates, fmt.Sprintf("discount_price = $%d", argIdx))
		args = append(args, *req.DiscountPrice)
		argIdx++
	}
	if len(req.GetImages()) > 0 {
		updates = append(updates, fmt.Sprintf("images = $%d", argIdx))
		args = append(args, pq.Array(req.GetImages()))
		argIdx++
	}
	if req.GetSpecs() != nil {
		specsJSON, _ := json.Marshal(structToMap(req.GetSpecs()))
		updates = append(updates, fmt.Sprintf("specs = $%d", argIdx))
		args = append(args, specsJSON)
		argIdx++
	}
	if len(req.GetVariants()) > 0 {
		variantsJSON, _ := json.Marshal(variantsToSlice(req.GetVariants()))
		updates = append(updates, fmt.Sprintf("variants = $%d", argIdx))
		args = append(args, variantsJSON)
		argIdx++
	}
	if req.DeliverySettings != nil {
		deliveryJSON, _ := json.Marshal(deliverySettingsToMap(req.DeliverySettings))
		updates = append(updates, fmt.Sprintf("delivery_settings = $%d", argIdx))
		args = append(args, deliveryJSON)
		argIdx++
	}
	if req.IsNew != nil {
		updates = append(updates, fmt.Sprintf("is_new = $%d", argIdx))
		args = append(args, *req.IsNew)
		argIdx++
	}
	if req.IsPopular != nil {
		updates = append(updates, fmt.Sprintf("is_popular = $%d", argIdx))
		args = append(args, *req.IsPopular)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(updates) == 0 {
		return s.GetProduct(ctx, &pb.GetProductRequest{Id: productID})
	}

	args = append(args, productID)
	query := fmt.Sprintf("UPDATE products SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}

	return s.GetProduct(ctx, &pb.GetProductRequest{Id: productID})
}

func (s *ProductServiceServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	productID := req.GetId()
	if productID == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	// Get shop_id and verify ownership
	var shopID string
	err := s.db.QueryRowContext(ctx, "SELECT shop_id FROM products WHERE id = $1", productID).Scan(&shopID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if auth.Role != "admin" {
		if err := s.verifyShopOwnership(ctx, shopID, auth.UserID); err != nil {
			return nil, err
		}
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM products WHERE id = $1", productID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
	}

	return &pb.Empty{}, nil
}

func (s *ProductServiceServer) ToggleProductStatus(ctx context.Context, req *pb.ToggleProductStatusRequest) (*pb.ProductResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	productID := req.GetId()
	if productID == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	// Get shop_id and verify ownership
	var shopID string
	err := s.db.QueryRowContext(ctx, "SELECT shop_id FROM products WHERE id = $1", productID).Scan(&shopID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if auth.Role != "admin" {
		if err := s.verifyShopOwnership(ctx, shopID, auth.UserID); err != nil {
			return nil, err
		}
	}

	_, err = s.db.ExecContext(ctx, "UPDATE products SET is_active = $1 WHERE id = $2", req.GetIsActive(), productID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to toggle status: %v", err)
	}

	return s.GetProduct(ctx, &pb.GetProductRequest{Id: productID})
}

// ============================================
// IMAGE UPLOAD (Streaming)
// ============================================

func (s *ProductServiceServer) UploadProductImage(stream pb.ProductService_UploadProductImageServer) error {
	ctx := stream.Context()
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	var metadata *pb.ImageMetadata
	var fileData []byte

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive: %v", err)
		}

		switch data := req.Data.(type) {
		case *pb.UploadImageRequest_Metadata:
			metadata = data.Metadata
			// Verify shop ownership if shop_id provided
			if metadata.GetShopId() != "" && auth.Role != "admin" {
				if err := s.verifyShopOwnership(ctx, metadata.GetShopId(), auth.UserID); err != nil {
					return err
				}
			}
		case *pb.UploadImageRequest_Chunk:
			fileData = append(fileData, data.Chunk...)
		}
	}

	if metadata == nil {
		return status.Error(codes.InvalidArgument, "metadata is required")
	}
	if len(fileData) == 0 {
		return status.Error(codes.InvalidArgument, "no image data received")
	}

	// Validate content type
	contentType := metadata.GetContentType()
	if !isValidImageType(contentType) {
		return status.Error(codes.InvalidArgument, "invalid image type. Supported: image/jpeg, image/png, image/webp")
	}

	// Generate filename and save
	ext := getExtensionFromContentType(contentType)
	filename := fmt.Sprintf("%s_%d%s", uuid.NewString()[:8], time.Now().Unix(), ext)

	// Create directory if needed
	if err := os.MkdirAll(s.uploadPath, 0755); err != nil {
		return status.Errorf(codes.Internal, "failed to create upload directory: %v", err)
	}

	filePath := filepath.Join(s.uploadPath, filename)
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return status.Errorf(codes.Internal, "failed to save image: %v", err)
	}

	imageURL := fmt.Sprintf("/uploads/products/%s", filename)

	return stream.SendAndClose(&pb.UploadImageResponse{
		Success:  true,
		ImageUrl: imageURL,
		Message:  "Image uploaded successfully",
	})
}

func (s *ProductServiceServer) UploadProductImages(stream pb.ProductService_UploadProductImagesServer) error {
	ctx := stream.Context()
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	var currentMetadata *pb.ImageMetadata
	var currentFileData []byte
	var uploadedURLs []string

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Save last file if there's pending data
			if currentMetadata != nil && len(currentFileData) > 0 {
				url, err := s.saveImageFile(currentMetadata, currentFileData)
				if err == nil {
					uploadedURLs = append(uploadedURLs, url)
				}
			}
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive: %v", err)
		}

		switch data := req.Data.(type) {
		case *pb.UploadImageRequest_Metadata:
			// Save previous file if exists
			if currentMetadata != nil && len(currentFileData) > 0 {
				url, err := s.saveImageFile(currentMetadata, currentFileData)
				if err == nil {
					uploadedURLs = append(uploadedURLs, url)
				}
			}
			currentMetadata = data.Metadata
			currentFileData = nil

			// Verify shop ownership
			if currentMetadata.GetShopId() != "" && auth.Role != "admin" {
				if err := s.verifyShopOwnership(ctx, currentMetadata.GetShopId(), auth.UserID); err != nil {
					return err
				}
			}
		case *pb.UploadImageRequest_Chunk:
			currentFileData = append(currentFileData, data.Chunk...)
		}
	}

	return stream.SendAndClose(&pb.BulkUploadImagesResponse{
		Success:   true,
		ImageUrls: uploadedURLs,
		Message:   fmt.Sprintf("Uploaded %d images", len(uploadedURLs)),
	})
}

// ============================================
// HELPER METHODS
// ============================================

func (s *ProductServiceServer) getProductByID(ctx context.Context, id string) (*pb.Product, error) {
	var p struct {
		ID               string
		ShopID           string
		CategoryID       sql.NullString
		Name             []byte
		Description      []byte
		Price            float64
		DiscountPrice    sql.NullFloat64
		Images           pq.StringArray
		Specs            []byte
		Variants         []byte
		DeliverySettings []byte
		Rating           float64
		ViewCount        int32
		SoldCount        int32
		IsNew            bool
		IsPopular        bool
		IsActive         bool
		CreatedAt        time.Time
		ShopName         []byte
		ShopLogo         sql.NullString
	}

	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.shop_id, p.category_id, p.name, p.description, p.price, p.discount_price,
			   p.images, p.specs, p.variants, p.delivery_settings, p.rating, p.view_count, p.sold_count,
			   p.is_new, p.is_popular, p.is_active, p.created_at,
			   COALESCE(s.name, '{}') as shop_name, s.logo_url
		FROM products p
		LEFT JOIN shops s ON p.shop_id = s.id
		WHERE p.id = $1
	`, id).Scan(
		&p.ID, &p.ShopID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
		&p.Images, &p.Specs, &p.Variants, &p.DeliverySettings, &p.Rating, &p.ViewCount, &p.SoldCount,
		&p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt, &p.ShopName, &p.ShopLogo,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	return s.scanToProduct(&p), nil
}

func (s *ProductServiceServer) listProductsInternal(ctx context.Context, filters *pb.ProductFilters, page, limit int, orderBy string) (*pb.ListProductsResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Build WHERE clause
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filters != nil {
		if filters.GetCategoryId() != "" {
			where = append(where, fmt.Sprintf("p.category_id = $%d", argIdx))
			args = append(args, filters.GetCategoryId())
			argIdx++
		}
		if filters.GetShopId() != "" {
			where = append(where, fmt.Sprintf("p.shop_id = $%d", argIdx))
			args = append(args, filters.GetShopId())
			argIdx++
		}
		if filters.GetIsNew() {
			where = append(where, "p.is_new = true")
		}
		if filters.GetIsPopular() {
			where = append(where, "p.is_popular = true")
		}
		if filters.GetIsActive() {
			where = append(where, "p.is_active = true")
		}
		if filters.GetMinPrice() > 0 {
			where = append(where, fmt.Sprintf("p.price >= $%d", argIdx))
			args = append(args, filters.GetMinPrice())
			argIdx++
		}
		if filters.GetMaxPrice() > 0 {
			where = append(where, fmt.Sprintf("p.price <= $%d", argIdx))
			args = append(args, filters.GetMaxPrice())
			argIdx++
		}
		if filters.GetSearch() != "" {
			where = append(where, fmt.Sprintf("(p.name::text ILIKE $%d OR p.description::text ILIKE $%d)", argIdx, argIdx))
			args = append(args, "%"+filters.GetSearch()+"%")
			argIdx++
		}
	}

	whereClause := strings.Join(where, " AND ")

	// Determine ORDER BY
	if orderBy == "" {
		switch filters.GetSortBy() {
		case "price_asc":
			orderBy = "p.price ASC"
		case "price_desc":
			orderBy = "p.price DESC"
		case "newest":
			orderBy = "p.created_at DESC"
		case "popular":
			orderBy = "p.sold_count DESC, p.view_count DESC"
		default:
			orderBy = "p.created_at DESC"
		}
	}

	// Count total
	var total int32
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products p WHERE %s", whereClause)
	s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	// Fetch products
	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT p.id, p.shop_id, p.category_id, p.name, p.description, p.price, p.discount_price,
			   p.images, p.specs, p.variants, p.delivery_settings, p.rating, p.view_count, p.sold_count,
			   p.is_new, p.is_popular, p.is_active, p.created_at,
			   COALESCE(s.name, '{}') as shop_name, s.logo_url
		FROM products p
		LEFT JOIN shops s ON p.shop_id = s.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var products []*pb.Product
	for rows.Next() {
		var p struct {
			ID               string
			ShopID           string
			CategoryID       sql.NullString
			Name             []byte
			Description      []byte
			Price            float64
			DiscountPrice    sql.NullFloat64
			Images           pq.StringArray
			Specs            []byte
			Variants         []byte
			DeliverySettings []byte
			Rating           float64
			ViewCount        int32
			SoldCount        int32
			IsNew            bool
			IsPopular        bool
			IsActive         bool
			CreatedAt        time.Time
			ShopName         []byte
			ShopLogo         sql.NullString
		}
		err := rows.Scan(
			&p.ID, &p.ShopID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
			&p.Images, &p.Specs, &p.Variants, &p.DeliverySettings, &p.Rating, &p.ViewCount, &p.SoldCount,
			&p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt, &p.ShopName, &p.ShopLogo,
		)
		if err != nil {
			continue
		}
		products = append(products, s.scanToProduct(&p))
	}

	return &pb.ListProductsResponse{
		Products: products,
		Total:    total,
		Page:     int32(page),
		Limit:    int32(limit),
	}, nil
}

func (s *ProductServiceServer) scanToProduct(p *struct {
	ID               string
	ShopID           string
	CategoryID       sql.NullString
	Name             []byte
	Description      []byte
	Price            float64
	DiscountPrice    sql.NullFloat64
	Images           pq.StringArray
	Specs            []byte
	Variants         []byte
	DeliverySettings []byte
	Rating           float64
	ViewCount        int32
	SoldCount        int32
	IsNew            bool
	IsPopular        bool
	IsActive         bool
	CreatedAt        time.Time
	ShopName         []byte
	ShopLogo         sql.NullString
}) *pb.Product {
	nameMap := make(map[string]string)
	json.Unmarshal(p.Name, &nameMap)

	descMap := make(map[string]string)
	json.Unmarshal(p.Description, &descMap)

	specsMap := make(map[string]interface{})
	json.Unmarshal(p.Specs, &specsMap)
	specs, _ := structpb.NewStruct(specsMap)

	shopNameMap := make(map[string]string)
	json.Unmarshal(p.ShopName, &shopNameMap)

	product := &pb.Product{
		Id:          p.ID,
		ShopId:      p.ShopID,
		Name:        mapToLocalizedString(nameMap),
		Description: mapToLocalizedString(descMap),
		Price:       p.Price,
		Images:      []string(p.Images),
		Specs:       specs,
		Rating:      p.Rating,
		ViewCount:   p.ViewCount,
		SoldCount:   p.SoldCount,
		IsNew:       p.IsNew,
		IsPopular:   p.IsPopular,
		IsActive:    p.IsActive,
		CreatedAt:   timestamppb.New(p.CreatedAt),
	}

	if p.CategoryID.Valid {
		product.CategoryId = p.CategoryID.String
	}
	if p.DiscountPrice.Valid {
		product.DiscountPrice = p.DiscountPrice.Float64
		// Calculate discount percent
		if p.Price > 0 {
			product.DiscountPercent = int32(((p.Price - p.DiscountPrice.Float64) / p.Price) * 100)
			product.HasDiscount = p.DiscountPrice.Float64 < p.Price
		}
	}

	// Shop info
	if uz, ok := shopNameMap["uz"]; ok {
		product.ShopName = uz
	}
	if p.ShopLogo.Valid {
		product.ShopLogo = p.ShopLogo.String
	}

	// Parse delivery settings
	var deliveryMap map[string]interface{}
	json.Unmarshal(p.DeliverySettings, &deliveryMap)
	product.DeliverySettings = mapToDeliverySettings(deliveryMap)

	// Parse variants
	var variantsSlice []map[string]interface{}
	json.Unmarshal(p.Variants, &variantsSlice)
	for _, v := range variantsSlice {
		product.Variants = append(product.Variants, mapToProductVariant(v))
	}

	return product
}

func (s *ProductServiceServer) verifyShopOwnership(ctx context.Context, shopID, userID string) error {
	var sellerID string
	err := s.db.QueryRowContext(ctx, "SELECT seller_id FROM shops WHERE id = $1", shopID).Scan(&sellerID)
	if err == sql.ErrNoRows {
		return status.Error(codes.NotFound, "shop not found")
	}
	if err != nil {
		return status.Errorf(codes.Internal, "query error: %v", err)
	}

	// Check if user owns this seller profile
	var ownerUserID string
	err = s.db.QueryRowContext(ctx, "SELECT user_id FROM seller_profiles WHERE id = $1", sellerID).Scan(&ownerUserID)
	if err != nil || ownerUserID != userID {
		return status.Error(codes.PermissionDenied, "you don't own this shop")
	}

	return nil
}

func (s *ProductServiceServer) saveImageFile(metadata *pb.ImageMetadata, data []byte) (string, error) {
	if !isValidImageType(metadata.GetContentType()) {
		return "", fmt.Errorf("invalid image type")
	}

	ext := getExtensionFromContentType(metadata.GetContentType())
	filename := fmt.Sprintf("%s_%d%s", uuid.NewString()[:8], time.Now().Unix(), ext)

	if err := os.MkdirAll(s.uploadPath, 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(s.uploadPath, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("/uploads/products/%s", filename), nil
}

// ============================================
// CONVERSION HELPERS
// ============================================

func mapToLocalizedString(m map[string]string) *pb.LocalizedString {
	if m == nil {
		return &pb.LocalizedString{}
	}
	return &pb.LocalizedString{
		Uz: m["uz"],
		Ru: m["ru"],
		En: m["en"],
	}
}

func localizedStringToMap(ls *pb.LocalizedString) map[string]string {
	if ls == nil {
		return map[string]string{}
	}
	return map[string]string{
		"uz": ls.Uz,
		"ru": ls.Ru,
		"en": ls.En,
	}
}

func structToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return s.AsMap()
}

func variantsToSlice(variants []*pb.ProductVariant) []map[string]interface{} {
	result := make([]map[string]interface{}, len(variants))
	for i, v := range variants {
		result[i] = map[string]interface{}{
			"name":           v.GetName(),
			"value":          v.GetValue(),
			"price_modifier": v.GetPriceModifier(),
			"images":         v.GetImages(),
			"attributes":     structToMap(v.GetAttributes()),
		}
	}
	return result
}

func deliverySettingsToMap(ds *pb.DeliverySettings) map[string]interface{} {
	if ds == nil {
		return map[string]interface{}{}
	}
	regionalPrices := make([]map[string]interface{}, len(ds.GetRegionalPrices()))
	for i, rp := range ds.GetRegionalPrices() {
		regionalPrices[i] = map[string]interface{}{
			"ids":           rp.GetRegionIds(),
			"price":         rp.GetPrice(),
			"name":          rp.GetName(),
			"delivery_days": rp.GetDeliveryDays(),
		}
	}
	return map[string]interface{}{
		"has_installation":    ds.GetHasInstallation(),
		"installation_price":  ds.GetInstallationPrice(),
		"home_region_price":   ds.GetHomeRegionPrice(),
		"is_home_region_free": ds.GetIsHomeRegionFree(),
		"home_delivery_days":  ds.GetHomeDeliveryDays(),
		"regional_prices":     regionalPrices,
	}
}

func mapToDeliverySettings(m map[string]interface{}) *pb.DeliverySettings {
	if m == nil {
		return nil
	}

	ds := &pb.DeliverySettings{}
	if v, ok := m["has_installation"].(bool); ok {
		ds.HasInstallation = v
	}
	if v, ok := m["installation_price"].(float64); ok {
		ds.InstallationPrice = v
	}
	if v, ok := m["home_region_price"].(float64); ok {
		ds.HomeRegionPrice = v
	}
	if v, ok := m["is_home_region_free"].(bool); ok {
		ds.IsHomeRegionFree = v
	}
	if v, ok := m["home_delivery_days"].(string); ok {
		ds.HomeDeliveryDays = v
	}
	if rp, ok := m["regional_prices"].([]interface{}); ok {
		for _, item := range rp {
			if rpMap, ok := item.(map[string]interface{}); ok {
				group := &pb.RegionalPriceGroup{}
				if ids, ok := rpMap["ids"].([]interface{}); ok {
					for _, id := range ids {
						if s, ok := id.(string); ok {
							group.RegionIds = append(group.RegionIds, s)
						}
					}
				}
				if v, ok := rpMap["price"].(float64); ok {
					group.Price = v
				}
				if v, ok := rpMap["name"].(string); ok {
					group.Name = v
				}
				if v, ok := rpMap["delivery_days"].(string); ok {
					group.DeliveryDays = v
				}
				ds.RegionalPrices = append(ds.RegionalPrices, group)
			}
		}
	}
	return ds
}

func mapToProductVariant(m map[string]interface{}) *pb.ProductVariant {
	v := &pb.ProductVariant{}
	if name, ok := m["name"].(string); ok {
		v.Name = name
	}
	if value, ok := m["value"].(string); ok {
		v.Value = value
	}
	if pm, ok := m["price_modifier"].(float64); ok {
		v.PriceModifier = pm
	}
	if images, ok := m["images"].([]interface{}); ok {
		for _, img := range images {
			if s, ok := img.(string); ok {
				v.Images = append(v.Images, s)
			}
		}
	}
	if attrs, ok := m["attributes"].(map[string]interface{}); ok {
		v.Attributes, _ = structpb.NewStruct(attrs)
	}
	return v
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func isValidImageType(contentType string) bool {
	validTypes := []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	for _, t := range validTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}
