# 🚀 go-api-server

Учебный REST API на Go с SQLite. Проект написан, чтобы освоить работу с БД, HTTP-хендлеры, middleware и базовый CRUD поверх `net/http`.

## 🛠️ Стек

- **Go 1.22+**
- **SQLite** (`modernc.org/sqlite`, файл `data/shop.db`)
- **JWT** (`github.com/golang-jwt/jwt/v5`) + **bcrypt** (`golang.org/x/crypto/bcrypt`)
- **Стандартная библиотека** — `net/http`, `database/sql`, `encoding/json`

Без фреймворков и ORM: роутинг через `http.ServeMux` (method+path patterns), SQL — параметризованные запросы.

## 📦 Структура проекта

```text
go-api-server/
├── main.go              # роутер, JWT-защита мутаций, :8080
├── models/              # User (с password), Order, валидация, bcrypt
├── storage/             # InitDB / CloseDB, схема SQLite
├── handlers/            # users, orders, auth, helpers
├── middleware/          # AuthMiddleware (Bearer JWT)
├── cmd/seed/seed.go     # тестовые данные (пароль password123)
├── data/                # shop.db
├── Dockerfile
├── docker-compose.yml
├── AGENTS.md
└── go.mod
```

## Запуск

### Локально

```bash
git clone https://github.com/andweb/go-api-server.git
cd go-api-server
go mod tidy

# Если обновляли схему users (password) — удалите старую БД и пересоздайте:
rm -f data/shop.db
go run ./cmd/seed

export JWT_SECRET=dev-secret-change-me
go run main.go
```

Сервер слушает `http://localhost:8080`.  
Сид создаёт 20 пользователей с паролем `password123` (email `userN@examle.com`).

### 🐳 Запуск через Docker Compose

Поднимает **3 экземпляра** API на портах **8080**, **8081** и **8082**. Папка `./data` общая для всех контейнеров.

```bash
# Собрать и запустить в фоне
docker compose up -d

# После изменения кода — обязательно пересобрать образ
docker compose up -d --build

# Смотреть логи всех реплик
docker compose logs -f

# Остановить и убрать контейнеры
docker compose down
```

> 🔁 **Важно:** правки в Go-коде сами в уже запущенные контейнеры не попадут. Нужен rebuild (`--build` или `docker compose build` + `up -d`), иначе крутится старый бинарник.

Проверка:

```bash
curl -s "http://localhost:8080/users?limit=20&offset=0"
curl -s http://localhost:8081/users
curl -s http://localhost:8082/users
```

> ⚠️ SQLite плохо переносит параллельную запись с нескольких процессов. Для учёбы масштабирование ок; в проде лучше одна БД с нормальной конкурентностью (Postgres и т.п.).

### Docker (один контейнер)

```bash
# Собрать образ
docker build -t go-api-server .

# Запуск в фоне (daemon) с именем контейнера
docker run -d --name go-api-server -p 8080:8080 go-api-server

# Остановить и удалить контейнер
docker stop go-api-server
docker rm go-api-server
```

После правок в коде снова `docker build -t go-api-server .`, затем stop/rm и новый `docker run …`.

Пример с сохранением БД на хосте:

```bash
docker run -d --name go-api-server -p 8080:8080 \
  -v "$(pwd)/data:/app/data" go-api-server

docker stop go-api-server && docker rm go-api-server
```

Проверить, что контейнер работает: `docker ps` или `curl -s http://localhost:8080/users`.

Сид в образе отдельно не запускается — для тестовых данных либо volume с уже заполненной БД, либо `POST /users` / `POST /orders` через API.

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

| Метод | Путь | Auth | Описание |
|-------|------|------|----------|
| `POST` | `/register` | нет | регистрация, ответ с JWT |
| `POST` | `/login` | нет | логин, ответ с JWT |
| `GET` | `/users` | нет | список (`?limit=&offset=`) |
| `GET` | `/users/{id}` | нет | один пользователь |
| `POST` | `/users` | Bearer | создать пользователя |
| `PUT` | `/users/{id}` | Bearer | обновить пользователя |
| `DELETE` | `/users/{id}` | Bearer | удалить пользователя |
| `GET` | `/users/{id}/orders` | нет | заказы пользователя |
| `POST` | `/orders` | Bearer | создать заказ |
| `DELETE` | `/orders/{id}` | Bearer | удалить заказ |

Ошибки:

```json
{"error":"not found","code":404,"timestamp":"2026-09-03T12:00:00Z"}
```

## Примеры curl

### Регистрация / логин

```bash
curl -s -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123","name":"Alice"}'

curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@examle.com","password":"password123"}'
```

Пример ответа:

```json
{"token":"<jwt>","user_id":1,"email":"user1@examle.com"}
```

Дальше используйте заголовок:

```bash
TOKEN=... # из ответа login/register
curl -s -H "Authorization: Bearer $TOKEN" ...
```

### Список пользователей

```bash
curl -s "http://localhost:8080/users?limit=20&offset=0"
```

### Создать пользователя (нужен JWT)

```bash
curl -s -X POST http://localhost:8080/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice2@example.com","password":"password123"}'
```

### Заказы пользователя

```bash
curl -s "http://localhost:8080/users/1/orders?limit=20&offset=0"
```

### Создать заказ (нужен JWT)

```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"product":"Laptop","quantity":2,"price":1000}'
```

### Удалить заказ (нужен JWT)

```bash
curl -i -X DELETE http://localhost:8080/orders/1 \
  -H "Authorization: Bearer $TOKEN"
```

## Что дальше

Планы развития проекта:

- [x] unit-тесты для handlers
- [x] Dockerfile (multistage)
- [x] docker-compose (3 реплики + volume для SQLite)
- [x] аутентификация (JWT) и защита мутаций
- [x] единый слой ошибок (`handleError` / `ErrorResponse`)
- [x] валидация входных данных (`ValidateUser`, пагинация limit/offset)
- [x] пагинация для списка users (`limit`/`offset`)
- [x] пагинация для заказов пользователя (`GET /users/{id}/orders`)

## Автор

**andweb** — [github.com/andweb](https://github.com/andweb)
