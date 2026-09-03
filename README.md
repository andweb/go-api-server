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
├── Dockerfile        # multistage-сборка образа
├── docker-compose.yml # 3 реплики API на 8080–8082
├── AGENTS.md         # правила для AI-агентов
└── go.mod
```

## Запуск

### Локально

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

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/users` | список пользователей (`?limit=&offset=`, по умолчанию 20/0) |
| `GET` | `/users/{id}` | один пользователь |
| `POST` | `/users` | создать пользователя |
| `PUT` | `/users/{id}` | обновить пользователя |
| `DELETE` | `/users/{id}` | удалить пользователя |
| `GET` | `/users/{id}/orders` | заказы пользователя (`?limit=&offset=`, по умолчанию 20/0) |
| `POST` | `/orders` | создать заказ |
| `DELETE` | `/orders/{id}` | удалить заказ |

Ошибки возвращаются в формате:

```json
{"error":"not found","code":404,"timestamp":"2026-09-03T12:00:00Z"}
```

Коды: `200` / `201` / `204` при успехе, `400` при невалидном вводе, `404` если запись не найдена.

## Примеры curl

### Список пользователей

```bash
curl -s "http://localhost:8080/users?limit=20&offset=0"
```

Пример ответа:

```json
{"data":[{"id":1,"name":"User 1","email":"user1@examle.com"}],"total":20,"limit":20,"offset":0}
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
curl -s "http://localhost:8080/users/1/orders?limit=20&offset=0"
```

Пример ответа:

```json
{"data":[{"id":1,"user_id":1,"product":"Laptop","quantity":1,"price":1000}],"total":5,"limit":20,"offset":0}
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
- [x] Dockerfile (multistage)
- [x] docker-compose (3 реплики + volume для SQLite)
- [ ] аутентификация (JWT / API key) и ограничение доступа к мутациям
- [x] единый слой ошибок (`handleError` / `ErrorResponse`)
- [ ] валидация входных данных
- [x] пагинация для списка users (`limit`/`offset`)
- [x] пагинация для заказов пользователя (`GET /users/{id}/orders`)

## Автор

**andweb** — [github.com/andweb](https://github.com/andweb)
