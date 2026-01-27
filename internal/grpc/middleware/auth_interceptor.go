package middleware

import (
	"context"
	"database/sql"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxRole   ctxKey = "role"
	ctxShopID ctxKey = "shop_id"
)

// AuthContext carries the authenticated user and shop context.
type AuthContext struct {
	UserID string
	Role   string
	ShopID string
}

// GetAuthContext extracts the AuthContext from the gRPC context if available.
func GetAuthContext(ctx context.Context) *AuthContext {
	val := ctx.Value(ctxUserID)
	if val == nil {
		return nil
	}
	return &AuthContext{
		UserID: val.(string),
		Role:   ctx.Value(ctxRole).(string),
		ShopID: ctx.Value(ctxShopID).(string),
	}
}

// NewAuthInterceptors creates unary and stream interceptors that:
// - Extract Authorization Bearer token from metadata.
// - Validate JWT using the provided secret.
// - Extract X-Shop-ID from metadata for multi-shop operations.
// skipAuthMethods should contain fully-qualified gRPC method names that are public (e.g., /auth.AuthService/Login).
func NewAuthInterceptors(jwtSecret []byte, db *sql.DB, skipAuthMethods map[string]bool) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	unary := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctxWithAuth, err := authenticate(ctx, info.FullMethod, jwtSecret, db, skipAuthMethods)
		if err != nil {
			return nil, err
		}
		return handler(ctxWithAuth, req)
	}

	stream := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctxWithAuth, err := authenticate(ss.Context(), info.FullMethod, jwtSecret, db, skipAuthMethods)
		if err != nil {
			return err
		}
		wrapped := &serverStreamWithContext{ServerStream: ss, ctx: ctxWithAuth}
		return handler(srv, wrapped)
	}

	return unary, stream
}

// authenticate validates JWT and enriches context with user/shop.
func authenticate(ctx context.Context, fullMethod string, jwtSecret []byte, _ *sql.DB, skip map[string]bool) (context.Context, error) {
	if skip != nil && skip[fullMethod] {
		// still capture shop id if provided
		return attachShopID(ctx), nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization header is required")
	}

	raw := authHeaders[0]
	if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization scheme")
	}

	tokenString := strings.TrimSpace(raw[len("bearer "):])
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Error(codes.Unauthenticated, "unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid claims")
	}

	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user_id missing in token")
	}
	if role == "" {
		role = "customer"
	}

	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRole, role)
	ctx = attachShopID(ctx)
	return ctx, nil
}

// attachShopID reads X-Shop-ID from metadata and attaches to context.
func attachShopID(ctx context.Context) context.Context {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		shopIDs := md.Get("x-shop-id")
		if len(shopIDs) > 0 {
			ctx = context.WithValue(ctx, ctxShopID, shopIDs[0])
			return ctx
		}
	}
	ctx = context.WithValue(ctx, ctxShopID, "")
	return ctx
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWithContext) Context() context.Context {
	return s.ctx
}
