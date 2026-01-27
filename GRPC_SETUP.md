# gRPC Migration Setup Guide

This document describes the gRPC migration setup for the Mebellar Backend.

## Project Structure

```
mebellar-backend/
├── proto/                    # Protocol Buffer definitions
│   ├── auth.proto
│   └── order.proto
├── pkg/pb/                   # Generated Go code from proto files
├── internal/grpc/
│   ├── middleware/           # gRPC interceptors (auth, etc.)
│   ├── server/               # gRPC service implementations
│   └── mapper/               # Domain <-> Proto mappers
└── scripts/
    └── generate_proto.sh     # Script to generate proto code
```

## Prerequisites

1. **protoc** - Protocol Buffer compiler
   ```bash
   # macOS
   brew install protobuf
   
   # Linux
   apt-get install protobuf-compiler
   ```

2. **protoc-gen-go** - Go plugin for protoc
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   ```

3. **protoc-gen-go-grpc** - gRPC Go plugin
   ```bash
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

4. **gRPC Go dependencies**
   ```bash
   go get google.golang.org/grpc
   go get google.golang.org/protobuf
   ```

## Generating Proto Code

Run the generation script:

```bash
./scripts/generate_proto.sh
```

Or manually:

```bash
protoc \
  --go_out=./pkg/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=./pkg/pb \
  --go-grpc_opt=paths=source_relative \
  --proto_path=./proto \
  ./proto/*.proto
```

## Running the Server

The server now runs both HTTP (port 8081) and gRPC (port 50051) concurrently:

```bash
go run main.go
```

### Environment Variables

- `SERVER_PORT` - HTTP server port (default: 8081)
- `GRPC_PORT` - gRPC server port (default: 50051)
- `JWT_SECRET` - JWT signing secret
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database config

## gRPC Services

### AuthService

- `Login` - User login (public)
- `Register` - User registration (public)
- `VerifyOTP` - OTP verification (public)
- `RefreshToken` - Token refresh (public)

### OrderService

- `CreateOrder` - Create new order (authenticated)
- `GetOrder` - Get order by ID (authenticated)
- `UpdateOrderStatus` - Update order status (authenticated)
- `DeleteOrder` - Delete order (authenticated)
- `ListOrders` - List orders with pagination (authenticated)
- `StreamOrders` - Server-side streaming for real-time order updates (authenticated)

## Authentication

gRPC uses JWT tokens passed via metadata:

```go
// Client example
md := metadata.New(map[string]string{
    "authorization": "Bearer <token>",
    "x-shop-id": "<shop-id>",  // Optional, for multi-shop context
})
ctx := metadata.NewOutgoingContext(context.Background(), md)
```

The interceptor automatically:
- Extracts JWT from `authorization` metadata
- Validates token and extracts user context
- Extracts `x-shop-id` for multi-shop operations
- Attaches user/shop info to context for service methods

## Testing with grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# Call a method
grpcurl -plaintext \
  -H "authorization: Bearer <token>" \
  -H "x-shop-id: <shop-id>" \
  -d '{"phone": "+998901234567", "password": "password123"}' \
  localhost:50051 auth.AuthService/Login
```

## Migration Notes

- HTTP server remains on port 8081 for backward compatibility
- WebSocket `/ws/orders` is replaced by `StreamOrders` gRPC method
- All existing REST endpoints continue to work
- gRPC services reuse existing database logic and models
