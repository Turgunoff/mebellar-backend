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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ShopServiceServer struct {
	pb.UnimplementedShopServiceServer
	db         *sql.DB
	uploadPath string
}

func NewShopServiceServer(db *sql.DB) *ShopServiceServer {
	return &ShopServiceServer{
		db:         db,
		uploadPath: "./uploads/shops",
	}
}

// ============================================
// PUBLIC ENDPOINTS
// ============================================

func (s *ShopServiceServer) GetShopBySlug(ctx context.Context, req *pb.GetShopBySlugRequest) (*pb.ShopResponse, error) {
	if req.GetSlug() == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	shop, err := s.getShopByField(ctx, "slug", req.GetSlug())
	if err != nil {
		return nil, err
	}
	return &pb.ShopResponse{Shop: shop}, nil
}

func (s *ShopServiceServer) GetPublicSellerProfile(ctx context.Context, req *pb.GetPublicSellerProfileRequest) (*pb.PublicSellerProfileResponse, error) {
	if req.GetSlug() == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}

	var id, shopName, slug string
	var description, logoURL, bannerURL, supportPhone sql.NullString
	var address, socialLinks, workingHours []byte
	var latitude, longitude sql.NullFloat64
	var isVerified bool
	var rating float64

	err := s.db.QueryRowContext(ctx, `
		SELECT id, shop_name, slug, description, logo_url, banner_url, support_phone,
			   address, latitude, longitude, social_links, working_hours, is_verified, rating
		FROM seller_profiles WHERE slug = $1
	`, req.GetSlug()).Scan(&id, &shopName, &slug, &description, &logoURL, &bannerURL, &supportPhone,
		&address, &latitude, &longitude, &socialLinks, &workingHours, &isVerified, &rating)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "seller profile not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	addressMap := make(map[string]string)
	json.Unmarshal(address, &addressMap)

	result := &pb.PublicSellerProfile{
		Id:           id,
		ShopName:     shopName,
		Slug:         slug,
		Address:      mapToLocalizedString(addressMap),
		SocialLinks:  parseSocialLinks(socialLinks),
		WorkingHours: parseWorkingHours(workingHours),
		IsVerified:   isVerified,
		Rating:       rating,
	}
	if description.Valid {
		result.Description = description.String
	}
	if logoURL.Valid {
		result.LogoUrl = logoURL.String
	}
	if bannerURL.Valid {
		result.BannerUrl = bannerURL.String
	}
	if supportPhone.Valid {
		result.SupportPhone = supportPhone.String
	}
	if latitude.Valid {
		result.Latitude = latitude.Float64
	}
	if longitude.Valid {
		result.Longitude = longitude.Float64
	}

	return &pb.PublicSellerProfileResponse{Profile: result}, nil
}

// ============================================
// SELLER ENDPOINTS
// ============================================

func (s *ShopServiceServer) GetMyShops(ctx context.Context, req *pb.GetMyShopsRequest) (*pb.GetMyShopsResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	var sellerID string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM seller_profiles WHERE user_id = $1", auth.UserID).Scan(&sellerID)
	if err == sql.ErrNoRows {
		return &pb.GetMyShopsResponse{Shops: []*pb.Shop{}, Count: 0}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	shops, err := s.listShopsBySeller(ctx, sellerID)
	if err != nil {
		return nil, err
	}

	return &pb.GetMyShopsResponse{Shops: shops, Count: int32(len(shops))}, nil
}

func (s *ShopServiceServer) GetShop(ctx context.Context, req *pb.GetShopRequest) (*pb.ShopResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "shop id is required")
	}

	shop, err := s.getShopByField(ctx, "id", req.GetId())
	if err != nil {
		return nil, err
	}

	if auth.Role != "admin" && auth.Role != "moderator" {
		if err := s.verifyShopOwnership(ctx, req.GetId(), auth.UserID); err != nil {
			return nil, err
		}
	}

	return &pb.ShopResponse{Shop: shop}, nil
}

func (s *ShopServiceServer) CreateShop(ctx context.Context, req *pb.CreateShopRequest) (*pb.ShopResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if auth.Role != "seller" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "seller or admin role required")
	}

	var sellerID string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM seller_profiles WHERE user_id = $1", auth.UserID).Scan(&sellerID)
	if err == sql.ErrNoRows {
		// Auto-create seller profile
		sellerID = uuid.NewString()
		shopName := req.GetName().GetUz()
		if shopName == "" {
			shopName = req.GetName().GetEn()
		}
		if shopName == "" {
			shopName = "My Shop"
		}

		slug := generateSlug(shopName)
		if slug == "" {
			slug = fmt.Sprintf("shop-%s", sellerID[:8])
		} else {
			slug = fmt.Sprintf("%s-%s", slug, sellerID[:8])
		}

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO seller_profiles (id, user_id, shop_name, slug, is_verified, rating, created_at, updated_at)
			VALUES ($1, $2, $3, $4, false, 0, NOW(), NOW())
		`, sellerID, auth.UserID, shopName, slug)

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to auto-create seller profile: %v", err)
		}

		// Update user role to seller if not already admin
		if auth.Role != "admin" {
			s.db.ExecContext(ctx, "UPDATE users SET role = 'seller' WHERE id = $1", auth.UserID)
		}
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	shopID := uuid.NewString()
	nameJSON, _ := json.Marshal(localizedStringToMap(req.GetName()))
	descJSON, _ := json.Marshal(localizedStringToMap(req.GetDescription()))
	addressJSON, _ := json.Marshal(localizedStringToMap(req.GetAddress()))
	workingHoursJSON, _ := json.Marshal(workingHoursToMap(req.GetWorkingHours()))

	slug := generateSlug(req.GetName().GetUz())
	if slug == "" {
		slug = generateSlug(req.GetName().GetEn())
	}
	slug = fmt.Sprintf("%s-%s", slug, shopID[:8])

	isActive := req.GetIsActive()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO shops (id, seller_id, name, description, address, slug, phone, region_id,
			latitude, longitude, working_hours, is_active, is_main, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
	`, shopID, sellerID, nameJSON, descJSON, addressJSON, slug, req.GetPhone(),
		nullInt(int(req.GetRegionId())), nullFloat(req.GetLatitude()), nullFloat(req.GetLongitude()),
		workingHoursJSON, isActive, req.GetIsMain())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create shop: %v", err)
	}

	return s.GetShop(ctx, &pb.GetShopRequest{Id: shopID})
}

func (s *ShopServiceServer) UpdateShop(ctx context.Context, req *pb.UpdateShopRequest) (*pb.ShopResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "shop id is required")
	}

	if auth.Role != "admin" && auth.Role != "moderator" {
		if err := s.verifyShopOwnership(ctx, req.GetId(), auth.UserID); err != nil {
			return nil, err
		}
	}

	updates := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

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
	if req.Address != nil {
		addressJSON, _ := json.Marshal(localizedStringToMap(req.Address))
		updates = append(updates, fmt.Sprintf("address = $%d", argIdx))
		args = append(args, addressJSON)
		argIdx++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *req.Phone)
		argIdx++
	}
	if req.RegionId != nil {
		updates = append(updates, fmt.Sprintf("region_id = $%d", argIdx))
		args = append(args, *req.RegionId)
		argIdx++
	}
	if req.Latitude != nil {
		updates = append(updates, fmt.Sprintf("latitude = $%d", argIdx))
		args = append(args, *req.Latitude)
		argIdx++
	}
	if req.Longitude != nil {
		updates = append(updates, fmt.Sprintf("longitude = $%d", argIdx))
		args = append(args, *req.Longitude)
		argIdx++
	}
	if req.LogoUrl != nil {
		updates = append(updates, fmt.Sprintf("logo_url = $%d", argIdx))
		args = append(args, *req.LogoUrl)
		argIdx++
	}
	if req.BannerUrl != nil {
		updates = append(updates, fmt.Sprintf("banner_url = $%d", argIdx))
		args = append(args, *req.BannerUrl)
		argIdx++
	}
	if req.WorkingHours != nil {
		workingHoursJSON, _ := json.Marshal(workingHoursToMap(req.WorkingHours))
		updates = append(updates, fmt.Sprintf("working_hours = $%d", argIdx))
		args = append(args, workingHoursJSON)
		argIdx++
	}
	if req.IsMain != nil {
		updates = append(updates, fmt.Sprintf("is_main = $%d", argIdx))
		args = append(args, *req.IsMain)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	args = append(args, req.GetId())
	query := fmt.Sprintf("UPDATE shops SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	s.db.ExecContext(ctx, query, args...)

	return s.GetShop(ctx, &pb.GetShopRequest{Id: req.GetId()})
}

func (s *ShopServiceServer) DeleteShop(ctx context.Context, req *pb.DeleteShopRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "shop id is required")
	}

	if auth.Role != "admin" && auth.Role != "moderator" {
		if err := s.verifyShopOwnership(ctx, req.GetId(), auth.UserID); err != nil {
			return nil, err
		}
	}

	var productCount int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products WHERE shop_id = $1", req.GetId()).Scan(&productCount)
	if productCount > 0 {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete shop with products")
	}

	s.db.ExecContext(ctx, "DELETE FROM shops WHERE id = $1", req.GetId())
	return &pb.Empty{}, nil
}

// ============================================
// SELLER PROFILE
// ============================================

func (s *ShopServiceServer) GetSellerProfile(ctx context.Context, req *pb.GetSellerProfileRequest) (*pb.SellerProfileResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	profile, err := s.getSellerProfileByUserID(ctx, auth.UserID)
	if err != nil {
		return nil, err
	}
	return &pb.SellerProfileResponse{Profile: profile}, nil
}

func (s *ShopServiceServer) UpgradeToSeller(ctx context.Context, req *pb.UpgradeToSellerRequest) (*pb.UpgradeToSellerResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if req.GetShopName() == "" {
		return nil, status.Error(codes.InvalidArgument, "shop_name is required")
	}

	var existingID string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM seller_profiles WHERE user_id = $1", auth.UserID).Scan(&existingID)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "you already have a seller profile")
	}

	profileID := uuid.NewString()
	slug := generateSlug(req.GetShopName())
	if slug == "" {
		slug = fmt.Sprintf("shop-%s", profileID[:8])
	}

	addressJSON, _ := json.Marshal(localizedStringToMap(req.GetAddress()))
	socialLinksJSON, _ := json.Marshal(socialLinksToMap(req.GetSocialLinks()))
	workingHoursJSON, _ := json.Marshal(workingHoursToMap(req.GetWorkingHours()))

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO seller_profiles (id, user_id, shop_name, slug, description, support_phone,
			address, social_links, working_hours, is_verified, rating, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, 0, NOW(), NOW())
	`, profileID, auth.UserID, req.GetShopName(), slug, req.GetDescription(), req.GetSupportPhone(),
		addressJSON, socialLinksJSON, workingHoursJSON)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create seller profile: %v", err)
	}

	s.db.ExecContext(ctx, "UPDATE users SET role = 'seller' WHERE id = $1", auth.UserID)
	profile, _ := s.getSellerProfileByUserID(ctx, auth.UserID)

	return &pb.UpgradeToSellerResponse{Success: true, Message: "Successfully upgraded to seller", Profile: profile}, nil
}

func (s *ShopServiceServer) UpdateSellerProfile(ctx context.Context, req *pb.UpdateSellerProfileRequest) (*pb.SellerProfileResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	updates := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if req.ShopName != nil {
		updates = append(updates, fmt.Sprintf("shop_name = $%d", argIdx))
		args = append(args, *req.ShopName)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.LogoUrl != nil {
		updates = append(updates, fmt.Sprintf("logo_url = $%d", argIdx))
		args = append(args, *req.LogoUrl)
		argIdx++
	}
	if req.BannerUrl != nil {
		updates = append(updates, fmt.Sprintf("banner_url = $%d", argIdx))
		args = append(args, *req.BannerUrl)
		argIdx++
	}
	if req.SupportPhone != nil {
		updates = append(updates, fmt.Sprintf("support_phone = $%d", argIdx))
		args = append(args, *req.SupportPhone)
		argIdx++
	}
	if req.Address != nil {
		addressJSON, _ := json.Marshal(localizedStringToMap(req.Address))
		updates = append(updates, fmt.Sprintf("address = $%d", argIdx))
		args = append(args, addressJSON)
		argIdx++
	}
	if req.SocialLinks != nil {
		socialLinksJSON, _ := json.Marshal(socialLinksToMap(req.SocialLinks))
		updates = append(updates, fmt.Sprintf("social_links = $%d", argIdx))
		args = append(args, socialLinksJSON)
		argIdx++
	}
	if req.WorkingHours != nil {
		workingHoursJSON, _ := json.Marshal(workingHoursToMap(req.WorkingHours))
		updates = append(updates, fmt.Sprintf("working_hours = $%d", argIdx))
		args = append(args, workingHoursJSON)
		argIdx++
	}

	args = append(args, auth.UserID)
	query := fmt.Sprintf("UPDATE seller_profiles SET %s WHERE user_id = $%d", strings.Join(updates, ", "), argIdx)
	s.db.ExecContext(ctx, query, args...)

	return s.GetSellerProfile(ctx, &pb.GetSellerProfileRequest{})
}

func (s *ShopServiceServer) UpdateLegalInfo(ctx context.Context, req *pb.UpdateLegalInfoRequest) (*pb.SellerProfileResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	s.db.ExecContext(ctx, `
		UPDATE seller_profiles SET legal_name = $1, tax_id = $2, bank_account = $3, bank_name = $4, updated_at = NOW()
		WHERE user_id = $5
	`, req.GetLegalName(), req.GetTaxId(), req.GetBankAccount(), req.GetBankName(), auth.UserID)

	return s.GetSellerProfile(ctx, &pb.GetSellerProfileRequest{})
}

func (s *ShopServiceServer) DeleteSellerAccount(ctx context.Context, req *pb.DeleteSellerAccountRequest) (*pb.DeleteSellerAccountResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	var shopCount int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shops sh JOIN seller_profiles sp ON sh.seller_id = sp.id WHERE sp.user_id = $1`, auth.UserID).Scan(&shopCount)
	if shopCount > 0 {
		return nil, status.Error(codes.FailedPrecondition, "please delete all shops first")
	}

	s.db.ExecContext(ctx, "DELETE FROM seller_profiles WHERE user_id = $1", auth.UserID)
	s.db.ExecContext(ctx, "UPDATE users SET role = 'customer' WHERE id = $1", auth.UserID)

	return &pb.DeleteSellerAccountResponse{Success: true, Message: "Seller account deleted"}, nil
}

// ============================================
// IMAGE UPLOAD
// ============================================

func (s *ShopServiceServer) UploadShopImage(stream pb.ShopService_UploadShopImageServer) error {
	ctx := stream.Context()
	auth := middleware.GetAuthContext(ctx)
	if auth == nil {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	var metadata *pb.ShopImageMetadata
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
		case *pb.UploadShopImageRequest_Metadata:
			metadata = data.Metadata
			if auth.Role != "admin" {
				if err := s.verifyShopOwnership(ctx, metadata.GetShopId(), auth.UserID); err != nil {
					return err
				}
			}
		case *pb.UploadShopImageRequest_Chunk:
			fileData = append(fileData, data.Chunk...)
		}
	}

	if metadata == nil || len(fileData) == 0 {
		return status.Error(codes.InvalidArgument, "metadata and image data required")
	}

	if !isValidImageType(metadata.GetContentType()) {
		return status.Error(codes.InvalidArgument, "invalid image type")
	}

	ext := getExtensionFromContentType(metadata.GetContentType())
	filename := fmt.Sprintf("%s_%s_%d%s", metadata.GetImageType(), uuid.NewString()[:8], time.Now().Unix(), ext)

	os.MkdirAll(s.uploadPath, 0755)
	filePath := filepath.Join(s.uploadPath, filename)
	os.WriteFile(filePath, fileData, 0644)

	imageURL := fmt.Sprintf("/uploads/shops/%s", filename)

	field := "logo_url"
	if metadata.GetImageType() == "banner" {
		field = "banner_url"
	}
	s.db.ExecContext(ctx, fmt.Sprintf("UPDATE shops SET %s = $1 WHERE id = $2", field), imageURL, metadata.GetShopId())

	return stream.SendAndClose(&pb.UploadShopImageResponse{Success: true, ImageUrl: imageURL, Message: "Image uploaded"})
}

// ============================================
// ADMIN ENDPOINTS
// ============================================

func (s *ShopServiceServer) AdminListShops(ctx context.Context, req *pb.AdminListShopsRequest) (*pb.ListShopsResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}

	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if req.GetSellerId() != "" {
		where = append(where, fmt.Sprintf("sh.seller_id = $%d", argIdx))
		args = append(args, req.GetSellerId())
		argIdx++
	}
	if req.GetActiveOnly() {
		where = append(where, "sh.is_active = true")
	}
	if req.GetSearch() != "" {
		where = append(where, fmt.Sprintf("sh.name::text ILIKE $%d", argIdx))
		args = append(args, "%"+req.GetSearch()+"%")
		argIdx++
	}

	page, limit := int(req.GetPage()), int(req.GetLimit())
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	whereClause := strings.Join(where, " AND ")

	var total int32
	s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM shops sh WHERE %s", whereClause), args...).Scan(&total)

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT sh.id, sh.seller_id, sh.name, sh.description, sh.address, sh.slug, sh.logo_url, sh.banner_url,
			   sh.phone, sh.latitude, sh.longitude, sh.region_id, COALESCE(r.name, '{}'),
			   sh.working_hours, sh.is_active, sh.is_verified, sh.is_main, sh.rating, sh.created_at, sh.updated_at
		FROM shops sh LEFT JOIN regions r ON sh.region_id = r.id
		WHERE %s ORDER BY sh.created_at DESC LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var shops []*pb.Shop
	for rows.Next() {
		shop, _ := s.scanShop(rows)
		if shop != nil {
			shops = append(shops, shop)
		}
	}

	return &pb.ListShopsResponse{Shops: shops, Total: total, Page: int32(page), Limit: int32(limit)}, nil
}

func (s *ShopServiceServer) AdminGetShop(ctx context.Context, req *pb.GetShopRequest) (*pb.ShopResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}
	shop, err := s.getShopByField(ctx, "id", req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.ShopResponse{Shop: shop}, nil
}

func (s *ShopServiceServer) AdminUpdateShop(ctx context.Context, req *pb.AdminUpdateShopRequest) (*pb.ShopResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}

	updates := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if req.IsVerified != nil {
		updates = append(updates, fmt.Sprintf("is_verified = $%d", argIdx))
		args = append(args, *req.IsVerified)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	args = append(args, req.GetId())
	query := fmt.Sprintf("UPDATE shops SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	s.db.ExecContext(ctx, query, args...)

	return s.AdminGetShop(ctx, &pb.GetShopRequest{Id: req.GetId()})
}

func (s *ShopServiceServer) AdminDeleteShop(ctx context.Context, req *pb.DeleteShopRequest) (*pb.Empty, error) {
	auth := middleware.GetAuthContext(ctx)
	if auth == nil || (auth.Role != "admin" && auth.Role != "moderator") {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}
	s.db.ExecContext(ctx, "DELETE FROM shops WHERE id = $1", req.GetId())
	return &pb.Empty{}, nil
}

// ============================================
// HELPERS
// ============================================

func (s *ShopServiceServer) getShopByField(ctx context.Context, field, value string) (*pb.Shop, error) {
	query := fmt.Sprintf(`
		SELECT sh.id, sh.seller_id, sh.name, sh.description, sh.address, sh.slug, sh.logo_url, sh.banner_url,
			   sh.phone, sh.latitude, sh.longitude, sh.region_id, COALESCE(r.name, '{}'),
			   sh.working_hours, sh.is_active, sh.is_verified, sh.is_main, sh.rating, sh.created_at, sh.updated_at
		FROM shops sh LEFT JOIN regions r ON sh.region_id = r.id
		WHERE sh.%s = $1
	`, field)

	row := s.db.QueryRowContext(ctx, query, value)

	// Scan into variables
	var id, sellerID, slug string
	var name, description, address, workingHours, regionName []byte
	var logoURL, bannerURL, phone sql.NullString
	var latitude, longitude sql.NullFloat64
	var regionID sql.NullInt32
	var isActive, isVerified, isMain bool
	var rating float64
	var createdAt, updatedAt time.Time

	err := row.Scan(&id, &sellerID, &name, &description, &address, &slug, &logoURL, &bannerURL,
		&phone, &latitude, &longitude, &regionID, &regionName, &workingHours, &isActive, &isVerified, &isMain, &rating, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "shop not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	shop := s.buildShop(id, sellerID, name, description, address, slug, logoURL, bannerURL, phone, latitude, longitude, regionID, regionName, workingHours, isActive, isVerified, isMain, rating, createdAt, updatedAt)
	return shop, nil
}

func (s *ShopServiceServer) listShopsBySeller(ctx context.Context, sellerID string) ([]*pb.Shop, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sh.id, sh.seller_id, sh.name, sh.description, sh.address, sh.slug, sh.logo_url, sh.banner_url,
			   sh.phone, sh.latitude, sh.longitude, sh.region_id, COALESCE(r.name, '{}'),
			   sh.working_hours, sh.is_active, sh.is_verified, sh.is_main, sh.rating, sh.created_at, sh.updated_at
		FROM shops sh LEFT JOIN regions r ON sh.region_id = r.id
		WHERE sh.seller_id = $1 ORDER BY sh.is_main DESC, sh.created_at DESC
	`, sellerID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var shops []*pb.Shop
	for rows.Next() {
		shop, _ := s.scanShop(rows)
		if shop != nil {
			shops = append(shops, shop)
		}
	}
	return shops, nil
}

func (s *ShopServiceServer) scanShop(rows *sql.Rows) (*pb.Shop, error) {
	var id, sellerID, slug string
	var name, description, address, workingHours, regionName []byte
	var logoURL, bannerURL, phone sql.NullString
	var latitude, longitude sql.NullFloat64
	var regionID sql.NullInt32
	var isActive, isVerified, isMain bool
	var rating float64
	var createdAt, updatedAt time.Time

	err := rows.Scan(&id, &sellerID, &name, &description, &address, &slug, &logoURL, &bannerURL,
		&phone, &latitude, &longitude, &regionID, &regionName, &workingHours, &isActive, &isVerified, &isMain, &rating, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	return s.buildShop(id, sellerID, name, description, address, slug, logoURL, bannerURL, phone, latitude, longitude, regionID, regionName, workingHours, isActive, isVerified, isMain, rating, createdAt, updatedAt), nil
}

func (s *ShopServiceServer) buildShop(id, sellerID string, name, description, address []byte, slug string, logoURL, bannerURL, phone sql.NullString, latitude, longitude sql.NullFloat64, regionID sql.NullInt32, regionName, workingHours []byte, isActive, isVerified, isMain bool, rating float64, createdAt, updatedAt time.Time) *pb.Shop {
	nameMap := make(map[string]string)
	json.Unmarshal(name, &nameMap)
	descMap := make(map[string]string)
	json.Unmarshal(description, &descMap)
	addrMap := make(map[string]string)
	json.Unmarshal(address, &addrMap)
	regionNameMap := make(map[string]string)
	json.Unmarshal(regionName, &regionNameMap)

	shop := &pb.Shop{
		Id:           id,
		SellerId:     sellerID,
		Name:         mapToLocalizedString(nameMap),
		Description:  mapToLocalizedString(descMap),
		Address:      mapToLocalizedString(addrMap),
		Slug:         slug,
		RegionName:   mapToLocalizedString(regionNameMap),
		WorkingHours: parseWorkingHours(workingHours),
		IsActive:     isActive,
		IsVerified:   isVerified,
		IsMain:       isMain,
		Rating:       rating,
		CreatedAt:    timestamppb.New(createdAt),
		UpdatedAt:    timestamppb.New(updatedAt),
	}
	if logoURL.Valid {
		shop.LogoUrl = logoURL.String
	}
	if bannerURL.Valid {
		shop.BannerUrl = bannerURL.String
	}
	if phone.Valid {
		shop.Phone = phone.String
	}
	if latitude.Valid {
		shop.Latitude = latitude.Float64
	}
	if longitude.Valid {
		shop.Longitude = longitude.Float64
	}
	if regionID.Valid {
		shop.RegionId = regionID.Int32
	}
	return shop
}

func (s *ShopServiceServer) verifyShopOwnership(ctx context.Context, shopID, userID string) error {
	return VerifyShopOwnershipHelper(ctx, s.db, shopID, userID)
}

func (s *ShopServiceServer) getSellerProfileByUserID(ctx context.Context, userID string) (*pb.SellerProfile, error) {
	var id, shopName, slug string
	var description, logoURL, bannerURL, legalName, taxID, bankAccount, bankName, supportPhone sql.NullString
	var address, socialLinks, workingHours []byte
	var latitude, longitude sql.NullFloat64
	var isVerified bool
	var rating float64
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT id, shop_name, slug, description, logo_url, banner_url, legal_name, tax_id, bank_account, bank_name,
			   support_phone, address, latitude, longitude, social_links, working_hours, is_verified, rating, created_at, updated_at
		FROM seller_profiles WHERE user_id = $1
	`, userID).Scan(&id, &shopName, &slug, &description, &logoURL, &bannerURL, &legalName, &taxID, &bankAccount, &bankName,
		&supportPhone, &address, &latitude, &longitude, &socialLinks, &workingHours, &isVerified, &rating, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "seller profile not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	addressMap := make(map[string]string)
	json.Unmarshal(address, &addressMap)

	profile := &pb.SellerProfile{
		Id:           id,
		UserId:       userID,
		ShopName:     shopName,
		Slug:         slug,
		Address:      mapToLocalizedString(addressMap),
		SocialLinks:  parseSocialLinks(socialLinks),
		WorkingHours: parseWorkingHours(workingHours),
		IsVerified:   isVerified,
		Rating:       rating,
		CreatedAt:    timestamppb.New(createdAt),
		UpdatedAt:    timestamppb.New(updatedAt),
	}
	if description.Valid {
		profile.Description = description.String
	}
	if logoURL.Valid {
		profile.LogoUrl = logoURL.String
	}
	if bannerURL.Valid {
		profile.BannerUrl = bannerURL.String
	}
	if legalName.Valid {
		profile.LegalName = legalName.String
	}
	if taxID.Valid {
		profile.TaxId = taxID.String
	}
	if bankAccount.Valid {
		profile.BankAccount = bankAccount.String
	}
	if bankName.Valid {
		profile.BankName = bankName.String
	}
	if supportPhone.Valid {
		profile.SupportPhone = supportPhone.String
	}
	if latitude.Valid {
		profile.Latitude = latitude.Float64
	}
	if longitude.Valid {
		profile.Longitude = longitude.Float64
	}
	return profile, nil
}

// Conversion helpers
func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}

func nullFloat(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}

func socialLinksToMap(sl *pb.SocialLinks) map[string]string {
	if sl == nil {
		return map[string]string{}
	}
	return map[string]string{
		"instagram": sl.GetInstagram(),
		"telegram":  sl.GetTelegram(),
		"facebook":  sl.GetFacebook(),
		"website":   sl.GetWebsite(),
		"youtube":   sl.GetYoutube(),
	}
}

func parseSocialLinks(data []byte) *pb.SocialLinks {
	var m map[string]string
	json.Unmarshal(data, &m)
	return &pb.SocialLinks{
		Instagram: m["instagram"],
		Telegram:  m["telegram"],
		Facebook:  m["facebook"],
		Website:   m["website"],
		Youtube:   m["youtube"],
	}
}

func workingHoursToMap(wh *pb.WorkingHours) map[string]interface{} {
	if wh == nil {
		return map[string]interface{}{}
	}
	dayToMap := func(d *pb.DaySchedule) map[string]interface{} {
		if d == nil {
			return nil
		}
		return map[string]interface{}{"open": d.GetOpen(), "close": d.GetClose(), "closed": d.GetClosed()}
	}
	return map[string]interface{}{
		"monday": dayToMap(wh.Monday), "tuesday": dayToMap(wh.Tuesday), "wednesday": dayToMap(wh.Wednesday),
		"thursday": dayToMap(wh.Thursday), "friday": dayToMap(wh.Friday), "saturday": dayToMap(wh.Saturday), "sunday": dayToMap(wh.Sunday),
	}
}

func parseWorkingHours(data []byte) *pb.WorkingHours {
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	parseDay := func(key string) *pb.DaySchedule {
		if d, ok := m[key].(map[string]interface{}); ok {
			return &pb.DaySchedule{
				Open:   getString(d, "open"),
				Close:  getString(d, "close"),
				Closed: getBool(d, "closed"),
			}
		}
		return nil
	}
	return &pb.WorkingHours{
		Monday: parseDay("monday"), Tuesday: parseDay("tuesday"), Wednesday: parseDay("wednesday"),
		Thursday: parseDay("thursday"), Friday: parseDay("friday"), Saturday: parseDay("saturday"), Sunday: parseDay("sunday"),
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
