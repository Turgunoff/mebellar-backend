# RBAC (Role-Based Access Control) Implementation

## Overview
This backend now supports Role-Based Access Control with four roles:
- **customer**: Regular users (default)
- **seller**: Shop owners
- **moderator**: Can access admin panel (read-only access)
- **admin**: Full access to admin panel

## Setup Instructions

### 1. Run Migrations

First, run the migrations to add role constraints and create the first admin user:

```bash
# Connect to your PostgreSQL database
psql -U your_user -d mebellar_olami

# Run migrations
\i migrations/011_add_role_constraint.sql
\i migrations/012_insert_first_admin.sql
```

Or if you have a migration runner:
```bash
# Run all migrations
psql -U your_user -d mebellar_olami -f migrations/011_add_role_constraint.sql
psql -U your_user -d mebellar_olami -f migrations/012_insert_first_admin.sql
```

### 2. Generate Admin Password Hash

The migration uses a placeholder hash. To generate the correct bcrypt hash for "admin_password":

```bash
cd /Users/eldor/Projects/mebellar-backend
go run scripts/generate_admin_hash.go
```

Copy the generated hash and update `migrations/012_insert_first_admin.sql` with the correct hash.

**OR** manually update the password hash in the migration file by running:

```go
package main
import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("admin_password"), 10)
    fmt.Println(string(hash))
}
```

### 3. Default Admin Credentials

After running the migration, you can log in with:
- **Phone**: `+998901234567`
- **Password**: `admin_password`
- **Role**: `admin`

⚠️ **IMPORTANT**: Change the admin password immediately after first login!

## API Endpoints

### Admin Endpoints (Protected)

All admin endpoints require authentication with `admin` or `moderator` role.

#### Get Dashboard Stats
```
GET /api/admin/dashboard-stats
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_users": 150,
    "total_sellers": 25,
    "total_products": 500,
    "total_orders": 1200,
    "total_revenue": 125000000.50,
    "active_users": 85,
    "pending_orders": 45,
    "completed_orders": 1100,
    "last_updated": "2024-01-15 10:30:00"
  }
}
```

## Security Features

### 1. Role Validation in Register
- Users can only register as `customer` or `seller`
- `admin` and `moderator` roles can only be assigned via database migrations or direct SQL

### 2. RequireRole Middleware
- Checks JWT token for user role
- Validates role against allowed roles list
- Returns `403 Forbidden` if role doesn't match

### 3. JWT Token Includes Role
- Role is included in JWT claims when token is generated
- Middleware extracts role from token (with database fallback)

## Testing

### Test Admin Access

1. **Login as Admin:**
```bash
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+998901234567",
    "password": "admin_password"
  }'
```

2. **Access Admin Endpoint:**
```bash
curl -X GET http://localhost:8081/api/admin/dashboard-stats \
  -H "Authorization: Bearer {token_from_step_1}"
```

3. **Test Customer Access (Should Fail):**
```bash
# Login as customer
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+998901111111",
    "password": "customer_password"
  }'

# Try to access admin endpoint (should return 403)
curl -X GET http://localhost:8081/api/admin/dashboard-stats \
  -H "Authorization: Bearer {customer_token}"
```

## Adding New Admin Endpoints

To add a new admin endpoint:

1. Create handler in `handlers/admin.go`
2. Protect with `RequireRole` middleware in `main.go`:

```go
http.HandleFunc("/api/admin/your-endpoint", 
    corsMiddleware(
        handlers.RequireRole(db, "admin", "moderator")(
            handlers.YourHandler(db)
        )
    )
)
```

## Database Schema

The `users` table now has:
- `role VARCHAR(20) NOT NULL DEFAULT 'customer'`
- CHECK constraint: `role IN ('customer', 'seller', 'moderator', 'admin')`

## Future Enhancements

- [ ] Permission-based access control (granular permissions)
- [ ] Admin user management endpoints
- [ ] Role assignment endpoints (for admins)
- [ ] Audit logging for admin actions
