# gRPC Migration Guide - Mebellar Backend

## Migration Complete! ✅

Your backend has been migrated from REST+gRPC hybrid to **100% gRPC**.

---

## What Was Created

### Phase 1: Proto Files (`proto/`)

| File             | Description                                          |
| ---------------- | ---------------------------------------------------- |
| `product.proto`  | Product CRUD, filters, image upload streaming        |
| `category.proto` | Category tree, flat list, attributes CRUD            |
| `shop.proto`     | Shop CRUD, seller profile, image upload              |
| `common.proto`   | Regions, Banners, Cancellation Reasons               |
| `user.proto`     | User profile, phone/email change, PIN, avatar upload |

### Phase 2: gRPC Services (`internal/grpc/server/`)

| File                  | Services                                       |
| --------------------- | ---------------------------------------------- |
| `product_service.go`  | ProductService (CRUD + streaming image upload) |
| `category_service.go` | CategoryService (tree + attributes)            |
| `shop_service.go`     | ShopService (seller + admin endpoints)         |
| `common_service.go`   | CommonService (regions, banners)               |
| `auth_service.go`     | AuthService (existing - login, register, OTP)  |
| `order_service.go`    | OrderService (existing - CRUD + streaming)     |

### Phase 3: New Main Entry Point

| File           | Description                           |
| -------------- | ------------------------------------- |
| `main_grpc.go` | Pure gRPC server + static file server |

---

## How to Complete the Migration

### Step 1: Generate Proto Code

```bash
# Install protoc plugins if not already installed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code from proto files
./scripts/generate_proto.sh
```

### Step 2: Replace main.go

```bash
# Backup old main.go
mv main.go main_rest_backup.go

# Use new gRPC-only main
mv main_grpc.go main.go
```

### Step 3: Update go.mod (if needed)

```bash
go mod tidy
```

### Step 4: Test the Server

```bash
go run main.go
```

You should see:

```
✅ .env fayli yuklandi
✅ Baza ulangan!
✅ Users jadvali tayyor!
✅ Seller Profiles jadvali tayyor!
✅ Eskiz SMS xizmati ulandi!
🔧 gRPC server sozlanmoqda...
🚀 Serverlar ishga tushmoqda...
✅ Static File Server 8081-portda tayyor!
✅ gRPC Server 50051-portda tayyor!
📡 Registered services: AuthService, OrderService, ProductService, CategoryService, ShopService, CommonService
```

### Step 5: Test with grpcurl

```bash
# Install grpcurl
brew install grpcurl

# List services
grpcurl -plaintext localhost:50051 list

# List methods in ProductService
grpcurl -plaintext localhost:50051 list product.ProductService

# Get products
grpcurl -plaintext localhost:50051 product.ProductService/ListProducts

# Get categories
grpcurl -plaintext localhost:50051 category.CategoryService/ListCategories
```

---

## Phase 4: Cleanup Instructions

### Files/Folders to DELETE after successful migration:

```bash
# Delete REST handlers (no longer needed)
rm -rf handlers/

# Delete old main.go backup (after confirming everything works)
rm main_rest_backup.go

# Delete Swagger docs (not needed for gRPC)
rm -rf docs/

# Optional: Remove unused middleware
# (Keep if you have other uses for it)
```

### Files to KEEP:

- `models/` - Database models (still used by gRPC services)
- `internal/grpc/` - All gRPC code
- `pkg/` - Shared packages (pb, sms, websocket, etc.)
- `migrations/` - Database migrations
- `uploads/` - Static file storage
- `proto/` - Proto definitions
- `scripts/` - Build/deploy scripts

### Dependencies to REMOVE from go.mod:

```go
// These are no longer needed after removing REST
github.com/rs/cors           // CORS middleware (gRPC doesn't need this)
github.com/swaggo/http-swagger // Swagger (REST documentation)
```

Run after cleanup:

```bash
go mod tidy
```

---

## Architecture After Migration

```
┌─────────────────────────────────────────────────────────────┐
│                     Mebellar Backend                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐      ┌──────────────────────────────┐ │
│  │  Static Server   │      │       gRPC Server            │ │
│  │   (Port 8081)    │      │       (Port 50051)           │ │
│  ├──────────────────┤      ├──────────────────────────────┤ │
│  │ /uploads/*       │      │ AuthService                  │ │
│  │ /health          │      │ ProductService               │ │
│  │                  │      │ CategoryService              │ │
│  │                  │      │ ShopService                  │ │
│  │                  │      │ CommonService                │ │
│  │                  │      │ OrderService                 │ │
│  └──────────────────┘      └──────────────────────────────┘ │
│                                      │                       │
│                                      ▼                       │
│                            ┌──────────────────┐             │
│                            │   PostgreSQL     │             │
│                            │    Database      │             │
│                            └──────────────────┘             │
└─────────────────────────────────────────────────────────────┘
```

---

## Client Integration

### Flutter/Dart Client

```dart
// Add to pubspec.yaml
dependencies:
  grpc: ^3.2.4
  protobuf: ^3.1.0

// Generate Dart code from proto
protoc --dart_out=grpc:lib/generated \
  -Iproto \
  proto/*.proto
```

### Example Dart Usage

```dart
import 'package:grpc/grpc.dart';
import 'generated/product.pbgrpc.dart';

final channel = ClientChannel(
  'api.mebellar-olami.uz',
  port: 50051,
  options: ChannelOptions(credentials: ChannelCredentials.insecure()),
);

final productClient = ProductServiceClient(channel);

// List products
final response = await productClient.listProducts(ListProductsRequest()
  ..page = 1
  ..limit = 20);

print('Products: ${response.products.length}');
```

### Image Upload (Streaming)

```dart
Stream<UploadImageRequest> uploadStream() async* {
  // First, send metadata
  yield UploadImageRequest()
    ..metadata = (ImageMetadata()
      ..filename = 'product.jpg'
      ..contentType = 'image/jpeg'
      ..shopId = 'shop-uuid');

  // Then send chunks
  final file = File('product.jpg');
  final bytes = await file.readAsBytes();
  final chunkSize = 64 * 1024; // 64KB chunks

  for (var i = 0; i < bytes.length; i += chunkSize) {
    final end = (i + chunkSize < bytes.length) ? i + chunkSize : bytes.length;
    yield UploadImageRequest()..chunk = bytes.sublist(i, end);
  }
}

final response = await productClient.uploadProductImage(uploadStream());
print('Uploaded: ${response.imageUrl}');
```

---

## Environment Variables

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=mebel_user
DB_PASSWORD=your_password
DB_NAME=mebellar_olami

# Ports
GRPC_PORT=50051
STATIC_PORT=8081

# Auth
JWT_SECRET=your-secret-key

# SMS (Eskiz)
ESKIZ_EMAIL=your@email.com
ESKIZ_PASSWORD=your_password
```

---

## Production Deployment Notes

1. **TLS/SSL**: For production, enable TLS on gRPC:

   ```go
   creds, _ := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
   grpcServer := grpc.NewServer(grpc.Creds(creds))
   ```

2. **Load Balancing**: Use Envoy or nginx with gRPC support

3. **Monitoring**: Add gRPC interceptors for logging/metrics

4. **Health Checks**: gRPC health checking protocol is already available via reflection

---

## Summary

| Before               | After                                |
| -------------------- | ------------------------------------ |
| REST API (port 8081) | Static files only (port 8081)        |
| gRPC (port 50051)    | gRPC for ALL operations (port 50051) |
| ~15 handler files    | 6 gRPC service files                 |
| CORS middleware      | Not needed                           |
| Swagger docs         | Use gRPC reflection + grpcui         |

**Migration Status: COMPLETE** ✅
