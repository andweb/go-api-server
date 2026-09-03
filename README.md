# 🚀 go-api-server

Учебный REST API на Go с SQLite. Проект написан, чтобы освоить работу с БД, HTTP-хендлеры, middleware и базовый CRUD поверх `net/http`.

## 🛠️ Стек

- **Go 1.22+**
- **SQLite** (`modernc.org/sqlite`, файл `data/shop.db`)
- **Стандартная библиотека** — `net/http`, `database/sql`, `encoding/json`

Без фреймворков и ORM: роутинг через `http.ServeMux` (method+path patterns), SQL — параметризованные запросы.

## 📦 Структура проекта

```text
go-api-server/
├── main.go           # точка входа, роутер, middleware, сервер :8080
├── models/           # структуры User и Order + JSON-теги
├── storage/          # InitDB / CloseDB, схема SQLite
├── handlers/         # HTTP-хендлеры users/orders + helpers
├── cmd/seed/seed.go  # заполнение тестовыми данными
├── data/             # shop.db (создаётся при запуске)
├── AGENTS.md         # правила для AI-агентов
└── go.mod
```

## Запуск

```bash
# 1. Клонировать репозиторий
git clone https://github.com/andweb/go-api-server.git
cd go-api-server

# 2. Подтянуть зависимости
go mod tidy

# 3. Заполнить БД тестовыми данными (один раз)
go run ./cmd/seed

# 4. Запустить сервер
go run main.go
```

Сервер слушает `http://localhost:8080`.  
Повторный запуск сида при уже заполненной таблице `users` выведет `Database already seeded` и завершится без дублей.

## Тесты

```bash
# все пакеты
go test ./...

# подробный вывод
go test -v ./...

# покрытие handlers + HTML-отчёт
go test ./handlers -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

`coverage.out` и `coverage.html` в git не коммитятся (см. `.gitignore`).

## Эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/users` | список всех пользователей |
| `GET` | `/users/{id}` | один пользователь |
| `POST` | `/users` | создать пользователя |
| `PUT` | `/users/{id}` | обновить пользователя |
| `DELETE` | `/users/{id}` | удалить пользователя |
| `GET` | `/users/{id}/orders` | заказы пользователя |
| `POST` | `/orders` | создать заказ |
| `DELETE` | `/orders/{id}` | удалить заказ |

Ошибки возвращаются в формате:

```json
{"error": "user not found"}
```

Коды: `200` / `201` / `204` при успехе, `400` при невалидном вводе, `404` если запись не найдена.

## Примеры curl

### Список пользователей

```bash
curl -s http://localhost:8080/users
```

### Создать пользователя

```bash
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

Пример ответа:

```json
{"id":21,"name":"Alice","email":"alice@example.com"}
```

### Заказы пользователя

```bash
curl -s http://localhost:8080/users/1/orders
```

### Создать заказ

```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"product":"Laptop","quantity":2,"price":1000}'
```

### Удалить заказ

```bash
curl -i -X DELETE http://localhost:8080/orders/1
```

Ожидаемый ответ при успехе: `204 No Content`.

## Что дальше

Планы развития проекта:

- [x] unit-тесты для handlers
- [ ] Dockerfile и `docker-compose` (API + volume для SQLite)
- [ ] аутентификация (JWT / API key) и ограничение доступа к мутациям
- [ ] валидация входных данных и единый слой ошибок
- [ ] пагинация для списков users/orders

## Автор

**andweb** — [github.com/andweb](https://github.com/andweb)
