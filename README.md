# Mebellar Backend API

## 📋 Loyiha Tavsifi

**Mebellar Olami Backend** - Premium mebel marketplace platformasi uchun RESTful API serveri. Go dasturlash tilida yozilgan, PostgreSQL ma'lumotlar bazasi bilan ishlaydi. Platforma mijozlar (customer) va sotuvchilar (seller) uchun to'liq funksionallikni ta'minlaydi.

### Asosiy Vazifalar:

- 🔐 Foydalanuvchi autentifikatsiyasi va autorizatsiyasi (JWT)
- 📱 SMS orqali OTP tasdiqlash (Eskiz.uz integratsiyasi)
- 🛋️ Mahsulotlar boshqaruvi (CRUD operatsiyalari)
- 🏪 Multi-shop arxitektura (bir foydalanuvchi bir nechta do'kon yaratishi mumkin)
- 📦 Buyurtmalar boshqaruvi va real-time kuzatuv (WebSocket)
- 📊 Dashboard statistikasi va analytics
- 🗂️ Kategoriyalar va hududlar boshqaruvi

---

## 🛠️ Texnologik Stek

### Core Technologies:

- **Go 1.23.4** - Backend dasturlash tili
- **PostgreSQL** - Relational ma'lumotlar bazasi
- **JWT (golang-jwt/jwt/v5)** - Token-based autentifikatsiya
- **Gorilla WebSocket** - Real-time aloqa

### Asosiy Kutubxonalar:

- `github.com/lib/pq` - PostgreSQL driver
- `github.com/golang-jwt/jwt/v5` - JWT token yaratish va tekshirish
- `github.com/gorilla/websocket` - WebSocket server
- `golang.org/x/crypto` - Parol hashing (bcrypt)
- `github.com/google/uuid` - UUID generatsiya
- `github.com/joho/godotenv` - Environment variables boshqaruvi
- `github.com/swaggo/swag` - Swagger dokumentatsiya

### Xizmatlar:

- **Eskiz.uz SMS Gateway** - SMS yuborish xizmati
- **Swagger UI** - API dokumentatsiyasi (`/swagger/`)

---

## 📁 Loyiha Strukturasi

Loyiha **Clean Architecture** prinsiplariga asoslangan:

```
mebellar-backend/
├── handlers/          # HTTP request handlers (Controller layer)
│   ├── auth.go       # Autentifikatsiya endpointlari
│   ├── user.go       # Foydalanuvchi profili endpointlari
│   ├── product.go    # Mahsulotlar endpointlari
│   ├── category.go   # Kategoriyalar endpointlari
│   ├── order.go      # Buyurtmalar endpointlari
│   ├── seller_profile.go  # Sotuvchi profili va do'konlar
│   └── region.go     # Hududlar endpointlari
│
├── models/           # Data models (Domain layer)
│   ├── user.go
│   ├── product.go
│   ├── category.go
│   ├── order.go
│   ├── seller_profile.go
│   └── region.go
│
├── pkg/              # Utility packages
│   ├── config/       # Konfiguratsiya boshqaruvi
│   ├── sms/          # SMS xizmati (Eskiz.uz)
│   ├── websocket/    # WebSocket hub va handler
│   └── seed/         # Database seeder (kategoriyalar)
│
├── migrations/       # SQL migration fayllari
│   ├── 002_create_seller_profiles.sql
│   ├── 003_add_delivery_settings.sql
│   ├── 004_add_shop_id_to_products.sql
│   ├── 005_create_regions_table.sql
│   ├── 006_add_product_analytics.sql
│   ├── 007_create_orders_table.sql
│   ├── 008_add_cancellation_reason.sql
│   └── 009_create_cancellation_reasons.sql
│
├── docs/            # Swagger dokumentatsiya (avtomatik generatsiya)
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── uploads/         # Yuklangan fayllar (rasmlar)
├── main.go          # Application entry point
├── go.mod           # Go dependencies
└── go.sum           # Dependency checksums
```

### Arxitektura Qatlamlari:

1. **Handlers Layer** - HTTP request/response boshqaruvi
2. **Models Layer** - Business logic va data strukturalari
3. **Pkg Layer** - Utility funksiyalar va xizmatlar
4. **Database Layer** - PostgreSQL ma'lumotlar bazasi

---

## 🔌 API Endpointlar

### Base URL

```
http://localhost:8081/api
```

### Autentifikatsiya (Auth)

| Method | Endpoint                | Vazifasi                              | Auth |
| ------ | ----------------------- | ------------------------------------- | ---- |
| POST   | `/auth/send-otp`        | OTP kod yuborish (telefon raqamiga)   | ❌   |
| POST   | `/auth/verify-otp`      | OTP kodni tasdiqlash                  | ❌   |
| POST   | `/auth/register`        | Yangi foydalanuvchi ro'yxatdan o'tish | ❌   |
| POST   | `/auth/login`           | Tizimga kirish (telefon + parol)      | ❌   |
| POST   | `/auth/forgot-password` | Parolni tiklash uchun OTP so'rash     | ❌   |
| POST   | `/auth/reset-password`  | Parolni yangilash (OTP bilan)         | ❌   |

### Foydalanuvchi Profili (User)

| Method | Endpoint                     | Vazifasi                                | Auth |
| ------ | ---------------------------- | --------------------------------------- | ---- |
| GET    | `/user/me`                   | Joriy foydalanuvchi profilini olish     | ✅   |
| PUT    | `/user/me`                   | Profilni yangilash (ism, avatar)        | ✅   |
| DELETE | `/user/me`                   | Hisobni o'chirish                       | ✅   |
| POST   | `/user/change-phone/request` | Telefon o'zgartirish - OTP so'rash      | ✅   |
| POST   | `/user/change-phone/verify`  | Telefon o'zgartirish - OTP tasdiqlash   | ✅   |
| POST   | `/user/change-email/request` | Email o'zgartirish - OTP so'rash        | ✅   |
| POST   | `/user/change-email/verify`  | Email o'zgartirish - OTP tasdiqlash     | ✅   |
| POST   | `/user/become-seller`        | Sotuvchi bo'lish (seller roliga o'tish) | ✅   |

### Kategoriyalar (Categories)

| Method | Endpoint                | Vazifasi                                | Auth |
| ------ | ----------------------- | --------------------------------------- | ---- |
| GET    | `/categories`           | Barcha kategoriyalar (daraxt struktura) | ❌   |
| GET    | `/categories?flat=true` | Kategoriyalar tekis ro'yxatda           | ❌   |
| GET    | `/categories/{id}`      | Bitta kategoriya ma'lumotlari           | ❌   |

### Hududlar (Regions)

| Method | Endpoint   | Vazifasi                          | Auth |
| ------ | ---------- | --------------------------------- | ---- |
| GET    | `/regions` | Barcha faol hududlar (viloyatlar) | ❌   |

### Mahsulotlar (Products) - Ommaviy

| Method | Endpoint                     | Vazifasi                    | Auth |
| ------ | ---------------------------- | --------------------------- | ---- |
| GET    | `/products`                  | Barcha mahsulotlar          | ❌   |
| GET    | `/products?category_id={id}` | Kategoriya bo'yicha filter  | ❌   |
| GET    | `/products/new`              | Yangi mahsulotlar           | ❌   |
| GET    | `/products/popular`          | Mashhur mahsulotlar         | ❌   |
| GET    | `/products/{id}`             | Bitta mahsulot ma'lumotlari | ❌   |

### Sotuvchi Do'konlari (Seller Shops)

| Method | Endpoint             | Vazifasi                                | Auth | Headers |
| ------ | -------------------- | --------------------------------------- | ---- | ------- |
| GET    | `/seller/shops`      | Mening do'konlarim ro'yxati             | ✅   | -       |
| POST   | `/seller/shops`      | Yangi do'kon yaratish                   | ✅   | -       |
| GET    | `/seller/shops/{id}` | Do'kon ma'lumotlari                     | ✅   | -       |
| PUT    | `/seller/shops/{id}` | Do'konni yangilash                      | ✅   | -       |
| DELETE | `/seller/shops/{id}` | Do'konni o'chirish                      | ✅   | -       |
| GET    | `/shops/{slug}`      | Ommaviy do'kon sahifasi (slug bo'yicha) | ❌   | -       |

### Sotuvchi Mahsulotlari (Seller Products)

| Method | Endpoint                | Vazifasi                      | Auth | Headers     |
| ------ | ----------------------- | ----------------------------- | ---- | ----------- |
| GET    | `/seller/products`      | Mening mahsulotlarim ro'yxati | ✅   | `X-Shop-ID` |
| POST   | `/seller/products`      | Yangi mahsulot yaratish       | ✅   | `X-Shop-ID` |
| PUT    | `/seller/products/{id}` | Mahsulotni yangilash          | ✅   | `X-Shop-ID` |
| DELETE | `/seller/products/{id}` | Mahsulotni o'chirish          | ✅   | `X-Shop-ID` |

### Buyurtmalar (Orders)

#### Mijoz (Customer) - Ommaviy

| Method | Endpoint  | Vazifasi                | Auth |
| ------ | --------- | ----------------------- | ---- |
| POST   | `/orders` | Yangi buyurtma yaratish | ❌   |

#### Sotuvchi (Seller)

| Method | Endpoint                                     | Vazifasi                        | Auth | Headers     |
| ------ | -------------------------------------------- | ------------------------------- | ---- | ----------- |
| GET    | `/seller/orders`                             | Buyurtmalar ro'yxati            | ✅   | `X-Shop-ID` |
| GET    | `/seller/orders?status={status}`             | Status bo'yicha filter          | ✅   | `X-Shop-ID` |
| GET    | `/seller/orders/stats`                       | Buyurtmalar statistikasi        | ✅   | `X-Shop-ID` |
| PUT    | `/seller/orders/{id}/status?status={status}` | Buyurtma statusini o'zgartirish | ✅   | `X-Shop-ID` |

**Buyurtma statuslari:** `new`, `confirmed`, `shipping`, `completed`, `cancelled`

### Sotuvchi Profili (Seller Profile)

| Method | Endpoint          | Vazifasi                              | Auth | Headers     |
| ------ | ----------------- | ------------------------------------- | ---- | ----------- |
| GET    | `/seller/profile` | Aggregated profil (user + shop stats) | ✅   | `X-Shop-ID` |
| PUT    | `/seller/profile` | Profilni yangilash (ism, parol)       | ✅   | `X-Shop-ID` |
| DELETE | `/seller/account` | Hisobni o'chirish (soft delete)       | ✅   | -           |

### Dashboard va Analytics

| Method | Endpoint                          | Vazifasi               | Auth | Headers     |
| ------ | --------------------------------- | ---------------------- | ---- | ----------- |
| GET    | `/seller/dashboard/stats`         | Dashboard statistikasi | ✅   | `X-Shop-ID` |
| GET    | `/seller/analytics/cancellations` | Bekor qilish tahlili   | ✅   | `X-Shop-ID` |

### Umumiy (Common)

| Method | Endpoint                       | Vazifasi                        | Auth |
| ------ | ------------------------------ | ------------------------------- | ---- |
| GET    | `/common/cancellation-reasons` | Bekor qilish sabablari ro'yxati | ❌   |

### Debug (Development)

| Method | Endpoint                       | Vazifasi                  | Auth |
| ------ | ------------------------------ | ------------------------- | ---- |
| POST   | `/debug/seed-orders?count={n}` | Test buyurtmalar yaratish | ✅   |

### WebSocket (Real-time)

| Protocol | Endpoint                                | Vazifasi                      | Auth |
| -------- | --------------------------------------- | ----------------------------- | ---- |
| WS       | `/ws/orders?token={JWT}&shop_id={UUID}` | Real-time buyurtmalar kuzatuv | ✅   |

### Static Files

| Method | Endpoint              | Vazifasi          |
| ------ | --------------------- | ----------------- |
| GET    | `/uploads/{filename}` | Yuklangan rasmlar |

### Swagger Dokumentatsiya

| Method | Endpoint              | Vazifasi   |
| ------ | --------------------- | ---------- |
| GET    | `/swagger/index.html` | Swagger UI |

---

## 🔐 Autentifikatsiya

API JWT (JSON Web Token) orqali himoyalangan endpointlar uchun `Authorization` headerida token yuboriladi:

```
Authorization: Bearer {token}
```

Token 7 kun amal qiladi va quyidagi ma'lumotlarni o'z ichiga oladi:

- `user_id` - Foydalanuvchi ID
- `phone` - Telefon raqami
- `role` - Rol (customer, seller)
- `exp` - Amal qilish muddati

---

## ⚙️ O'rnatish va Ishga Tushirish

### Talablar:

- Go 1.24.0 yoki yuqori versiya
- PostgreSQL 12+
- `.env` fayl (konfiguratsiya)

### 1. Loyihani klonlash va dependencies o'rnatish

```bash
cd mebellar-backend
go mod download
```

### 2. Environment Variables sozlash

`.env` fayl yaratish:

```bash
cp .env.example .env
```

Kerakli qiymatlarni tahrirlash:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=mebel_user
DB_PASSWORD=your_password
DB_NAME=mebellar_olami

# SSL Mode (production uchun REQUIRED!)
DB_SSLMODE=require

# Connection Pool
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
DB_CONN_MAX_IDLE_TIME=5m

# Server
SERVER_PORT=8081
STATIC_PORT=8081
GRPC_PORT=50051

# JWT Secret (32+ characters required!)
# Generate: openssl rand -base64 32
JWT_SECRET=your-super-secret-jwt-key-minimum-32-chars

# SMS Service (Eskiz.uz)
ESKIZ_EMAIL=your@email.com
ESKIZ_PASSWORD=your_password

# Environment
ENVIRONMENT=development

# Logging
LOG_LEVEL=info
```

### 🔐 Xavfsizlik (MUHIM!)

#### JWT Secret Generatsiyasi

**Production uchun MAJBURIY!** Kuchli tasodifiy secret yarating:

```bash
openssl rand -base64 32
```

Natijani `.env` fayliga qo'shing:

```env
JWT_SECRET=wOqJ3f7xM9kL2pN5tR8vY1zC4bH6jK0nQ3sU7wA9e=
```

**Qoidalar**:

- ✅ Minimum 32 ta belgi
- ❌ Default qiymatlarni ishlatmang
- ❌ Git'ga commit qilmang
- ✅ Har bir muhit uchun alohida secret

#### SSL/TLS Konfiguratsiyasi

**Production muhitida SSL MAJBURIY!**

```env
# Production
DB_SSLMODE=require

# Yoki sertifikat bilan
DB_SSLMODE=verify-full
DB_SSL_ROOT_CERT=/path/to/server-ca.pem
```

SSL rejimlari:

- `disable` - SSL o'chirilgan (faqat development)
- `require` - SSL majburiy
- `verify-ca` - Server sertifikatini tekshirish
- `verify-full` - Sertifikat + hostname tekshirish (tavsiya)

### 3. Database va Migrationlar

```bash
# PostgreSQL da database yaratish
createdb mebellar_olami

# Barcha migratsiyalarni avtomatik qo'llash
make migrate-up

# Migratsiya versiyasini ko'rish
make migrate-version

# Oxirgi migratsiyani bekor qilish
make migrate-down

# Yangi migratsiya yaratish
make migrate-create NAME=add_new_feature

# Bazani reset qilish (⚠️ barcha ma'lumotlar o'chadi!)
make db-reset
```

### 4. Serverni ishga tushirish

```bash
# Development mode
make run

# Yoki to'g'ridan-to'g'ri
go run main.go

# Build qilish
make build
./bin/mebellar-backend
```

**Server muvaffaqiyatli ishga tushganda**:

```
✅ .env fayli yuklandi
2024-01-28T10:30:45+0500  INFO  Starting Mebellar Backend  {"environment": "development", "version": "1.0.0"}
2024-01-28T10:30:45+0500  INFO  Configuration validated successfully
2024-01-28T10:30:45+0500  INFO  Database connected successfully
2024-01-28T10:30:45+0500  INFO  Connection pool configured  {"max_open_connections": 25}
✅ Все миграции применены успешно (текущая версия: 29)
2024-01-28T10:30:45+0500  INFO  Starting servers  {"static_port": "8081", "grpc_port": "50051"}
```

### 📝 Structured Logging

Loyihada [zap](https://github.com/uber-go/zap) logger ishlatiladi.

**Log darajalari**:

```env
LOG_LEVEL=debug   # Batafsil ma'lumot (development)
LOG_LEVEL=info    # Standart (default)
LOG_LEVEL=warn    # Ogohlantirishlar
LOG_LEVEL=error   # Faqat xatoliklar
```

**Development** (rangli chiqish):

```
2024-01-28T10:30:45+0500  INFO  Starting server  {"port": "8081"}
```

**Production** (JSON format):

```json
{
  "level": "info",
  "timestamp": "2024-01-28T10:30:45Z",
  "msg": "Starting server",
  "port": "8081"
}
```

### 🏥 Health Check va Monitoring

Server holatini tekshirish:

```bash
curl http://localhost:8081/health
```

Javob (connection pool statistikasi bilan):

```json
{
  "status": "ok",
  "service": "mebellar-backend",
  "database": {
    "open_connections": 5,
    "in_use": 2,
    "idle": 3,
    "wait_count": 0,
    "wait_duration_ms": 0,
    "max_idle_closed": 0,
    "max_lifetime_closed": 0
  }
}
```

### 5. Swagger dokumentatsiyasini generatsiya qilish

```bash
# Swagger CLI o'rnatish
go install github.com/swaggo/swag/cmd/swag@latest

# Dokumentatsiyani generatsiya qilish
swag init
```

---

## 🧪 Testing

### API ni test qilish

1. **Swagger UI** orqali: `http://localhost:8081/swagger/index.html`
2. **Postman** yoki **cURL** orqali
3. **Flutter ilovalar** orqali integratsiya

### Test Endpointlar

```bash
# Health check
curl http://localhost:8081/api/categories

# OTP yuborish
curl -X POST http://localhost:8081/api/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{"phone": "+998901234567"}'
```

---

## 📊 Database Strukturasi

### Asosiy Jadvalar:

- `users` - Foydalanuvchilar
- `seller_profiles` - Sotuvchi profillari (Multi-Shop)
- `products` - Mahsulotlar
- `categories` - Kategoriyalar
- `orders` - Buyurtmalar
- `regions` - Hududlar (viloyatlar)
- `cancellation_reasons` - Bekor qilish sabablari

### Migrations:

Barcha migration fayllar `migrations/` papkasida joylashgan va ketma-ket bajarilishi kerak.

---

## 🔧 Konfiguratsiya

Konfiguratsiya `.env` fayl orqali boshqariladi. Barcha sozlamalar `pkg/config/config.go` da yuklanadi.

### Muhim Sozlamalar:

- **JWT_SECRET** - Token imzolash uchun maxfiy kalit
- **ESKIZ_EMAIL/PASSWORD** - SMS xizmati uchun (agar bo'sh bo'lsa, mock rejimida ishlaydi)
- **DB\_\*** - Database ulanish parametrlari

---

## 📝 API Response Format

Barcha API javoblari quyidagi formatda qaytadi:

```json
{
  "success": true,
  "message": "Muvaffaqiyatli",
  "data": { ... }
}
```

Xatolik holatida:

```json
{
  "success": false,
  "message": "Xatolik xabari"
}
```

---

## 🚀 Production Deployment

1. Environment variables ni production qiymatlariga o'zgartirish
2. `ENVIRONMENT=production` o'rnatish
3. Database backup yaratish
4. HTTPS sozlash (reverse proxy orqali: Nginx, Caddy)
5. Process manager ishlatish (systemd, PM2, Supervisord)

---

## 🧪 Тестирование

### Запуск всех тестов

```bash
make test
```

### Unit тесты

```bash
make test-unit
```

### Integration тесты

```bash
make test-integration
```

### Coverage отчет

```bash
make test-coverage
```

### Настройка тестовой БД

```bash
make test-db-setup
```

### Переменные для тестов

Файл `.env.test`:

```env
TEST_DB_HOST=localhost
TEST_DB_PORT=5432
TEST_DB_USER=mebel_user
TEST_DB_PASSWORD=
TEST_DB_NAME=mebellar_test
```

---

## � Docker Deployment

### Quick Start с Docker Compose

```bash
# 1. Клонировать репозиторий
git clone https://github.com/Turgunoff/mebellar-backend.git
cd mebellar-backend

# 2. Создать .env файл
cp .env.example .env
# Отредактировать .env (установить JWT_SECRET, пароли и т.д.)

# 3. Запустить все сервисы
docker-compose up -d

# 4. Проверить статус
docker-compose ps

# 5. Посмотреть логи
docker-compose logs -f backend
```

### Development окружение

```bash
# Запустить только БД и Redis
docker-compose -f docker-compose.dev.yml up -d

# Запустить backend локально
go run main.go
```

### Production deployment

```bash
# Build production image
make docker-prod-build VERSION=1.0.0

# Deploy на сервере
docker-compose up -d
```

### Docker команды

```bash
# Rebuild и restart
docker-compose up -d --build

# Остановить все
docker-compose down

# Удалить volumes (⚠️ удалит данные!)
docker-compose down -v

# Посмотреть логи
docker-compose logs -f [service_name]

# Выполнить команду в контейнере
docker-compose exec backend sh
```

---

## ⚡ Performance & Caching

### Redis Configuration

Проект использует Redis для:

✅ Rate limiting (защита от DDOS)  
✅ Кэширование категорий, регионов  
✅ Session management (опционально)

### Настройка кэша

```env
# .env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Rate Limiting

| Endpoint       | Лимит       | Окно     |
| -------------- | ----------- | -------- |
| /auth/login    | 5 запросов  | 1 минута |
| /auth/send-otp | 3 запроса   | 1 минута |
| /auth/register | 3 запроса   | 1 минута |
| Остальные      | 60 запросов | 1 минута |

---

## 🚀 CI/CD Pipeline

### GitHub Actions Workflows

Проект использует автоматический CI/CD:

**На каждый Push/PR:**

✅ Lint & Code Quality Check  
✅ Unit Tests (с coverage ≥70%)  
✅ Integration Tests  
✅ Security Scan  
✅ Docker Image Build

**На Push в main:**

✅ Deploy to Staging  
✅ Smoke Tests

### Локальный запуск CI проверок

```bash
# Lint
make lint

# Tests
make test

# Coverage
make test-coverage

# Build Docker
make docker-build
```

---

## 🔒 Security Best Practices

### Production Checklist

- ✅ SSL enabled для PostgreSQL
- ✅ JWT_SECRET минимум 32 символа
- ✅ Rate limiting включен
- ✅ CORS правильно настроен
- ✅ Input validation на всех endpoints
- ✅ Secrets не в коде (через .env)
- ✅ Non-root Docker user
- ✅ Security scanning в CI/CD
- ✅ Structured logging (нет логирования OTP/паролей)

### Рекомендации

🔄 Регулярно обновлять зависимости  
🔒 Использовать secrets manager (AWS Secrets, Vault)  
📊 Настроить monitoring (Prometheus + Grafana)  
🔐 Включить 2FA для админов  
💾 Настроить автоматические бэкапы БД

---

## 📚 Qo'shimcha Ma'lumotlar

- **Swagger UI**: `http://localhost:8081/swagger/index.html`
- **WebSocket**: Real-time buyurtmalar kuzatuv uchun
- **Multi-Shop**: Bir foydalanuvchi bir nechta do'kon yaratishi mumkin
- **File Uploads**: `uploads/` papkasida saqlanadi
- **Health Check**: `http://localhost:8081/health`

---

## 👥 Mualliflar

Mebellar Olami Development Team

---

## 📄 License

Proprietary - All rights reserved
