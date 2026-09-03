# AGENTS.md — Go REST API (SQLite)

Инструкции для агентов, работающих с этим репозиторием. Только специфика проекта.

## О проекте

REST API на **Go 1.22+** с **SQLite** (`modernc.org/sqlite`, файл `data/shop.db`).

Модуль: `go-api-server`.

Домен: пользователи (`users`) и заказы (`orders`). Продукты есть в сиде/схеме, отдельного HTTP CRUD для products пока нет.

ID сущностей выдаёт SQLite `AUTOINCREMENT` — в коде ID не назначать при создании.

## Команды

Из корня модуля:

```bash
go mod tidy
go build ./...
go run main.go
go run ./cmd/seed
```

- `go run main.go` — HTTP-сервер (точка входа в корне).
- `go run ./cmd/seed` — заполнение тестовыми данными; повторный запуск при уже заполненных users печатает `Database already seeded` и выходит.
- Сид и сервер ожидают рабочую директорию = корень модуля (путь `data/shop.db` относительный).

Не коммитить секреты. База `data/shop.db` — локальные данные; не считать её исходником API.

## Архитектура

```
models/     # структуры User, Order + JSON-теги
storage/    # InitDB / CloseDB, схема SQLite
handlers/   # HTTP-хендлеры users/orders + helpers
cmd/seed/   # отдельный исполняемый сид (не часть main.go)
data/       # shop.db (создаётся InitDB)
```

### models/

- `User`: `id`, `name`, `email`
- `Order`: `id`, `user_id`, `product`, `quantity`, `price`
- Только DTO/доменные структуры. Без SQL и без HTTP.

### storage/

- Пакет `storage`.
- `InitDB() (*sql.DB, error)` — создаёт `data/` при необходимости, открывает `data/shop.db`, включает `PRAGMA foreign_keys = ON`, создаёт таблицы через `CREATE TABLE IF NOT EXISTS`.
- `CloseDB(db *sql.DB)` — закрытие соединения.
- Таблицы: `users`, `products`, `orders` (`orders.user_id` → `users(id) ON DELETE CASCADE`).
- Драйвер: blank-import `_ "modernc.org/sqlite"`, DSN driver name `"sqlite"`.

### handlers/

- Пакет `handlers`.
- `helpers.go` — общие `writeError`, `respondJSON`, `parseUserID`, `errorBody`.
- `users.go` — CRUD пользователей.
- `orders.go` — заказы пользователя, создание заказа, удаление заказа.
- Хендлеры вида `func Xxx(db *sql.DB) http.HandlerFunc`.
- ID из URL: `r.PathValue("id")` (Go 1.22+).
- JSON body: `json.NewDecoder(r.Body).Decode(...)`.

Ожидаемые маршруты (когда есть router в `main.go`):

| Method | Path | Handler |
|--------|------|---------|
| GET | `/users` | `GetUsers` |
| GET | `/users/{id}` | `GetUser` |
| POST | `/users` | `CreateUser` |
| PUT | `/users/{id}` | `UpdateUser` |
| DELETE | `/users/{id}` | `DeleteUser` |
| GET | `/users/{id}/orders` | `GetUserOrders` |
| POST | `/orders` | `CreateOrder` |
| DELETE | `/orders/{id}` | `DeleteOrder` |

HTTP-коды: успех списка/чтения/обновления — `200`; создание — `201`; удаление — `204`; нет записи — `404` с `{"error":"..."}`; невалидный ввод — `400`.

## Правила кодинга

### Ответы HTTP

- Успешный JSON: только через `respondJSON(w, status, data)`.
- Ошибки: только через `writeError(w, status, msg)` → тело `{"error":"..."}`.
- Для `204 No Content` вызывать `respondJSON(w, http.StatusNoContent, nil)` — при `data == nil` тело не кодировать.
- Не дублировать `w.Header().Set("Content-Type", ...)` / `json.NewEncoder` в хендлерах.
- Общие хелперы держать в `handlers/helpers.go`, не копировать в `users.go` / `orders.go`.

### SQL

- Только параметризованные запросы: `?` + `Prepare` / `Query` / `QueryRow` / `Exec`.
- **Запрещено** собирать SQL через `fmt.Sprintf` / конкатенацию строк с пользовательскими данными.
- Проверка существования: `QueryRow(...).Scan` + `sql.ErrNoRows` → 404.
- После `INSERT` брать ID через `LastInsertId()`.
- После `DELETE`/`UPDATE`, где нужен 404, проверять `RowsAffected()`.

### Пакеты и зависимости

- Модели — в `models`, БД — в `storage`, HTTP — в `handlers`.
- Сид — только `cmd/seed`, не вшивать в `main.go`.
- Импорт моделей в `storage` допустим для связи пакетов; бизнес-логика HTTP не должна жить в `storage`.
- Новый код — в стиле существующих файлов (короткие хендлеры, early return, `defer stmt.Close()`).

### Документация агентов

- Экспортируемые функции (`GetUsers`, `CreateOrder`, …) — с кратким godoc-комментарием.
- Журнал работ проекта ведётся в `WORKLOG.md`: исходник промпта + краткий результат по шагам. При существенных изменениях по запросу пользователя — дописывать шаг туда.

## Чего не делать

- Не назначать `id` вручную при создании записей.
- Не отключать foreign keys без явной причины.
- Не класть второй `package main` рядом с корневым `main.go` (сид — в `cmd/seed`).
- Не добавлять ORM / другой SQL-драйвер без явного запроса.
- Не раздувать `AGENTS.md` общими гайдами по Go — только правила этого репозитория.
