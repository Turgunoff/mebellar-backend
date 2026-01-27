package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryLogger logs unary gRPC requests with method, IP, status, duration, and error.
func UnaryLogger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	
	// Extract client IP
	clientIP := getClientIP(ctx)
	
	// Call the handler
	resp, err := handler(ctx, req)
	
	// Calculate duration
	duration := time.Since(start)
	
	// Get status code
	statusCode := status.Code(err)
	
	// Log the request
	if err != nil {
		log.Printf("❌ [gRPC] %s | IP: %s | Status: %s | Duration: %v | Error: %v",
			info.FullMethod, clientIP, statusCode, duration, err)
	} else {
		log.Printf("✅ [gRPC] %s | IP: %s | Status: %s | Duration: %v",
			info.FullMethod, clientIP, statusCode, duration)
	}
	
	return resp, err
}

// StreamLogger logs streaming gRPC requests with start/end and duration.
func StreamLogger(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	clientIP := getClientIP(ss.Context())
	
	log.Printf("🔄 [gRPC Stream] %s | IP: %s | Started",
		info.FullMethod, clientIP)
	
	err := handler(srv, ss)
	
	duration := time.Since(start)
	statusCode := status.Code(err)
	
	if err != nil {
		log.Printf("❌ [gRPC Stream] %s | IP: %s | Status: %s | Duration: %v | Error: %v",
			info.FullMethod, clientIP, statusCode, duration, err)
	} else {
		log.Printf("✅ [gRPC Stream] %s | IP: %s | Status: %s | Duration: %v",
			info.FullMethod, clientIP, statusCode, duration)
	}
	
	return err
}

// getClientIP extracts the client IP address from the gRPC context.
func getClientIP(ctx context.Context) string {
	// Try to get IP from peer
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	
	// Try to get from metadata (X-Forwarded-For or X-Real-IP)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if forwarded := md.Get("x-forwarded-for"); len(forwarded) > 0 {
			return forwarded[0]
		}
		if realIP := md.Get("x-real-ip"); len(realIP) > 0 {
			return realIP[0]
		}
	}
	
	return "unknown"
}
