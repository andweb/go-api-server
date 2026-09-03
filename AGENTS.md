# AGENTS.md — Go REST API (SQLite)

Инструкции для агентов, работающих с этим репозиторием. Только специфика проекта.

## О проекте

REST API на **Go 1.22+** (в `go.mod` сейчас может быть выше из‑за зависимостей) с **SQLite** (`modernc.org/sqlite`, без CGO, файл `data/shop.db`).

Модуль: `go-api-server`.

Домен: пользователи (`users`) и заказы (`orders`). Продукты есть в сиде/схеме, отдельного HTTP CRUD для products нет.

ID сущностей выдаёт SQLite `AUTOINCREMENT` — в коде ID не назначать при создании.

## Команды

Из корня модуля:

```bash
go mod tidy
go build ./...
go run main.go
go run ./cmd/seed
go test ./...
go test -v ./handlers
go test ./handlers -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Docker / Compose (плагин `docker compose`, не `docker-compose`):

```bash
docker build -t go-api-server .
docker compose up -d --build
docker compose logs -f
docker compose down
```

**После любого изменения кода** образ нужно пересобрать: `docker compose up -d --build` или `docker build` + перезапуск контейнера. Иначе в контейнере останется старый бинарник.

- Сид и сервер ожидают cwd = корень модуля (`data/shop.db` относительный).
- `coverage.out` / `coverage.html` / `*.db` — в `.gitignore`, не коммитить.
- База `data/shop.db` — локальные данные, не исходник API.

## Архитектура

```
main.go              # ServeMux, logging middleware, :8080
models/              # User, Order + JSON-теги
storage/             # InitDB, CloseDB, Migrate, Schema
handlers/            # users, orders, helpers + *_test.go
cmd/seed/            # отдельный сид (не в main.go)
data/                # shop.db + .gitkeep
Dockerfile           # multistage, CGO_ENABLED=0, бинарник api-server
docker-compose.yml   # 3 реплики api, порты 8080–8082, volume ./data
```

### models/

- `User`: `id`, `name`, `email`
- `Order`: `id`, `user_id`, `product`, `quantity`, `price`
- Только структуры. Без SQL и без HTTP.

### storage/

- `InitDB() (*sql.DB, error)` → `data/shop.db`, `PRAGMA foreign_keys = ON`, затем `Migrate`.
- `Migrate(db)` / `Schema` — общая схема для InitDB и тестов (in-memory SQLite).
- `CloseDB(db *sql.DB)`.
- Таблицы: `users`, `products`, `orders` (`ON DELETE CASCADE` на `user_id`).
- Драйвер: `_ "modernc.org/sqlite"`, имя драйвера `"sqlite"`.

### handlers/

- `helpers.go` — `handleError`, `ErrorResponse`, `respondJSON`, `parseUserID`.
- `users.go` — CRUD + пагинация `GetUsers`.
- `orders.go` — `GetUserOrders` (пагинация), `CreateOrder`, `DeleteOrder`.
- Хендлеры: `func Xxx(db *sql.DB) http.HandlerFunc`.
- ID из URL: `r.PathValue("id")`.
- Тесты: `httptest` + `setupTestDB` (SQLite `:memory:` + `storage.Migrate`).

| Method | Path | Handler | Заметки |
|--------|------|---------|---------|
| GET | `/users` | `GetUsers` | `?limit=&offset=`; ответ `{data,total,limit,offset}` |
| GET | `/users/{id}` | `GetUser` | |
| POST | `/users` | `CreateUser` | |
| PUT | `/users/{id}` | `UpdateUser` | |
| DELETE | `/users/{id}` | `DeleteUser` | 204 / 404 |
| GET | `/users/{id}/orders` | `GetUserOrders` | та же пагинация; `COUNT(*)` по `user_id` |
| POST | `/orders` | `CreateOrder` | |
| DELETE | `/orders/{id}` | `DeleteOrder` | 204 / 404 |

Пагинация (`GetUsers`, `GetUserOrders`): default `limit=20`, `offset=0`; `limit > 100` → 400 `"limit cannot exceed 100"`; нечисловые query → 400; SQL `LIMIT ? OFFSET ?` + `SELECT COUNT(*)`.

HTTP-коды: `200` / `201` / `204`; `400`/`404`/`500` через `handleError` → `{"error","code","timestamp"}`.

## Правила кодинга

### Ответы HTTP

- Успех: `respondJSON(w, status, data)`.
- Ошибки: `handleError(w, err, status, msg)` → `{"error":"...","code":N,"timestamp":"..."}`.
- Пустой `msg` → текст по статусу (`bad request` / `not found` / `internal error`); детали `err` только в лог.
- `204`: `respondJSON(w, http.StatusNoContent, nil)` (nil → без тела).
- Хелперы только в `helpers.go`, не дублировать.

### SQL

- Только `?` + `Prepare` / `Query` / `QueryRow` / `Exec`.
- **Запрещено** `fmt.Sprintf` / конкатенация SQL с пользовательскими данными.
- Нет записи: `sql.ErrNoRows` или `RowsAffected() == 0` → 404.
- После `INSERT` — `LastInsertId()`.

### Docker

- Multistage: `golang:1.22-alpine` (при необходимости `GOTOOLCHAIN=auto`) → `alpine:latest`.
- Бинарник: `api-server`; в образ — бинарник + `data/`.
- Compose: сервис `api`, `deploy.replicas: 3`, порты `8080-8082:8080`, volume `./data`.
- Не задавать `container_name` вместе с `replicas`.
- Поле `version:` в compose не использовать (Compose v2+/v5).
- Общий SQLite на несколько реплик — только для учёбы (гонки записи).

### Документация

- Экспортируемые хендлеры — с кратким godoc.
- `WORKLOG.md` — шаги: исходник промпта + краткий результат.
- Держать `AGENTS.md` и README в синхроне с реальным поведением API (пагинация, Docker rebuild и т.д.).

## Чего не делать

- Не назначать `id` вручную при создании.
- Не отключать foreign keys без причины.
- Не класть второй `package main` рядом с `main.go` (сид — `cmd/seed`).
- Не добавлять ORM / другой SQL-драйвер без явного запроса.
- Не коммитить `coverage.*`, `data/*.db`, `.env`.
- Не раздувать `AGENTS.md` общими гайдами по Go.
