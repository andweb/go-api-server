# AGENTS.md — Go REST API (SQLite)

Инструкции для агентов, работающих с этим репозиторием. Только специфика проекта.

## О проекте

REST API на **Go 1.22+** (в `go.mod` может быть выше) с **SQLite** (`modernc.org/sqlite`, без CGO), **JWT** и **bcrypt**.

Модуль: `go-api-server`.

Домен: пользователи (`users`) и заказы (`orders`). Auth: `POST /register`, `POST /login`.

ID выдаёт SQLite `AUTOINCREMENT` — не назначать вручную.

## Команды

```bash
go mod tidy
go build ./...
rm -f data/shop.db   # после смены схемы users
go run ./cmd/seed    # пароль сида: password123
export JWT_SECRET=dev-secret-change-me
go run main.go
go test ./...
swag init -g main.go -o docs   # после смены @-комментариев swag
docker compose up -d --build
```

**После изменения кода** — `docker compose up -d --build`.  
Секрет JWT: `JWT_SECRET` (дефолт в коде/compose: `dev-secret-change-me`).  
Swagger UI: `http://localhost:8080/swagger/index.html`.

## Архитектура

```
main.go           # ServeMux, AuthMiddleware, /swagger/
docs/             # сгенерировано swag (docs.go, swagger.json/yaml)
models/           # User(+password), Order, Validate*, HashPassword
storage/          # InitDB, Migrate, Schema
handlers/         # users, orders, auth, helpers (+ swag annotations)
middleware/       # AuthMiddleware, GetUserIDFromContext
cmd/seed/         # сид с bcrypt-паролями
```

### models/

- `User`: `id`, `name`, `email`, `password` (`omitempty` в JSON; в ответах очищать).
- `HashPassword` / `CheckPassword` (bcrypt).
- `ValidateUser`, `ValidateEmail` (email — regexp).

### storage/

- Таблица `users`: `password TEXT NOT NULL`, `email UNIQUE`.
- Смена схемы → удалить `data/shop.db` и пересидить (`CREATE TABLE IF NOT EXISTS` не мигрирует старые таблицы).

### auth / middleware

- `handlers.Register` / `Login` → `{"token","user_id","email"}`.
- `generateJWT(userID, email)` — HS256, claims `user_id` + `email`, TTL 24h.
- `middleware.AuthMiddleware` — `Authorization: Bearer <token>`.
- `GetUserIDFromContext(r)` — id из context.

### Маршруты

| Method | Path | Auth |
|--------|------|------|
| POST | `/register`, `/login` | нет |
| GET | `/users`, `/users/{id}`, `/users/{id}/orders` | нет |
| POST/PUT/DELETE | `/users…`, POST/DELETE `/orders…` | Bearer JWT |
| GET | `/swagger/` | нет (UI + spec) |

Пагинация через `parsePagination`. Ошибки через `handleError` → `{"error","code","timestamp"}`.

`CreateUser` требует `password` (≥6), хеширует перед INSERT.  
Сид: все users с паролем `password123`.

Swagger: метаданные в `main.go` (`@title`, `@securityDefinitions.apikey BearerAuth`); аннотации на хендлерах Auth/Users/Orders; импорт `_ "go-api-server/docs"`; после правок комментариев — `swag init -g main.go -o docs`. Экспортированные типы для схем: `AuthRequest`, `AuthResponse`, `UsersPage`, `OrdersPage`.

## Правила

- SQL только с `?`, без `fmt.Sprintf` в запросах.
- Пароль/хеш не отдавать в JSON ответов.
- Не коммитить `data/*.db`, `coverage.*`, `.env`.
- После смены swag-комментариев перегенерировать `docs/` (`swag init`).
- Не раздувать AGENTS общими гайдами по Go.
