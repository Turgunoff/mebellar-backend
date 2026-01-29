# План: Критическая безопасность и инфраструктура backend

Внедрение пяти критических улучшений за 1-2 недели: SSL для PostgreSQL, обязательная валидация конфигурации, автоматические миграции БД, connection pool и структурированное логирование. Фокус на безопасности перед production-запуском.

## Шаги

1. **Включить SSL для PostgreSQL**: Обновить [main.go](mebellar-backend/main.go) добавив переменные окружения `DB_SSLMODE` и `DB_SSL_ROOT_CERT`, валидацию production-конфигурации, создать `.env.example` с документацией SSL-режимов

2. **Добавить валидацию конфигурации**: Создать функцию `validateConfig()` в [main.go](mebellar-backend/main.go) проверяющую минимальную длину JWT_SECRET (32 символа), блокирующую дефолтные значения, требующую `sslmode!=disable` в production

3. **Внедрить golang-migrate**: Создать [pkg/database/migrate.go](mebellar-backend/pkg/database/migrate.go), удалить `createUsersTable()` и `createSellerProfilesTable()` из [main.go](mebellar-backend/main.go), переименовать 29 файлов в [migrations/](mebellar-backend/migrations/) добавив `.up.sql`/`.down.sql` суффиксы, создать `Makefile` с командами управления

4. **Настроить connection pool**: Добавить функции `getEnvInt()`, `getEnvDuration()`, `configureConnectionPool()` в [main.go](mebellar-backend/main.go) с параметрами `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`, расширить `/health` endpoint статистикой пула

5. **Интегрировать zap logger**: Создать [pkg/logger/logger.go](mebellar-backend/pkg/logger/logger.go), заменить все `log.Println()`, `log.Fatal()`, `fmt.Printf()` на структурированные вызовы в [main.go](mebellar-backend/main.go), [internal/grpc/server/auth_service.go](mebellar-backend/internal/grpc/server/auth_service.go) и других сервисах, удалить логирование OTP-кодов

## Дополнительные соображения

1. **Миграции**: Обнаружен дубликат [008_add_cancellation_reason.sql](mebellar-backend/migrations/008_add_cancellation_reason.sql) и [008_add_user_is_active.sql](mebellar-backend/migrations/008_add_user_is_active.sql) - объединить или переименовать перед внедрением golang-migrate?

2. **Зависимости**: Добавить в `go.mod`: `github.com/golang-migrate/migrate/v4`, `go.uber.org/zap`. Запустить `go get` и `go mod tidy`?

3. **Тестирование**: После изменений проверить старт сервера с невалидной конфигурацией (короткий JWT_SECRET, `sslmode=disable` в production), rollback миграций, health endpoint с pool stats?
