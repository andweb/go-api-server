# Журнал работ

## Шаг 1 — 2026-09-02

### Исходник промпта

> Создай Go-проект для REST API. В папке models/ создай файл models.go с двумя структурами: User (ID, Name, Email) и Order (ID, UserID, Product, Quantity, Price). У всех полей должны быть JSON-теги для сериализации. Добавь конструкторы для создания новых записей (NewUser, NewOrder)
>
> Так же прошу тебя сразу вести какой-то лог работ в md, записывать по шагам исходник промптов и краткий результат

### Краткий результат

- Инициализирован модуль `github.com/andweb/api-server` (`go.mod`, Go 1.22).
- Добавлен пакет `models/models.go`: структуры `User` и `Order` с JSON-тегами, конструкторы `NewUser` и `NewOrder` (ID выдаётся атомарным счётчиком).
- Заведён этот журнал (`WORKLOG.md`).

## Шаг 2 — 2026-09-02

### Исходник промпта

> В файле models.go убери атомарные счетчики (userSeq, orderSeq) и конструкторы NewUser, NewOrder. Оставь только структуры User и Order с JSON-тегами. ID будут генерироваться базой данных SQLite через AUTOINCREMENT, поэтому не нужно назначать их в коде. Исправленный файл должен компилироваться без ошибок

### Краткий результат

- Из `models/models.go` удалены `userSeq`/`orderSeq`, импорт `sync/atomic` и конструкторы `NewUser`/`NewOrder`.
- Остались только `User` и `Order` с JSON-тегами; ID предполагается от SQLite `AUTOINCREMENT`.

## Шаг 3 — 2026-09-02

### Исходник промпта

> Напиши файл storage/db.go для Go-проекта. Он должен:
> Импортировать database/sql, modernc.org/sqlite и models (наш пакет).
> Иметь функцию InitDB() *sql.DB, которая открывает файл shop.db в папке data (если папки нет — создать её).
> Создать таблицы users и orders. У users: id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL. У orders: id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, product TEXT NOT NULL, quantity INTEGER NOT NULL, price REAL NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE.
> Таблицы создавать только если их нет (IF NOT EXISTS).
> Возвращать *sql.DB и ошибку, если что-то пошло не так.
> Добавить функцию CloseDB(db *sql.DB), которая закрывает соединение.
> Весь код должен быть в пакете storage.

### Краткий результат

- Добавлен `storage/db.go`: `InitDB() (*sql.DB, error)` создаёт `data/`, открывает `data/shop.db`, включает `PRAGMA foreign_keys`, создаёт `users` и `orders` через `IF NOT EXISTS`.
- `CloseDB` закрывает соединение; зависимость `modernc.org/sqlite` добавлена в модуль.
- Импорт `models` сохранён (ссылки `_ models.User` / `_ models.Order`), чтобы пакет компилировался.

## Шаг 4 — 2026-09-02

### Исходник промпта

> Напиши отдельный Go-скрипт seed.go, который заполняет базу данных тестовыми данными. Он должен:
> Использовать storage.InitDB() для подключения к базе.
> Проверить, есть ли уже пользователи в таблице users (SELECT COUNT(*) FROM users). Если есть — вывести "Database already seeded" и завершиться (чтобы не дублировать данные).
> Вставить 20 пользователей с именами "User 1".."User 20" и email "user1@mail.com".."user20@mail.com".
> Вставить 10 продуктов: названия "Laptop", "Phone", "Tablet", "Headphones", "Monitor", "Keyboard", "Mouse", "Charger", "Speaker", "Camera". Цены: 1000, 500, 700, 150, 300, 100, 50, 80, 200, 400 (соответственно).
> Вставить 50 случайных заказов: случайный user_id (от 1 до 20), случайный product (из списка продуктов), случайное quantity (от 1 до 5), случайная price (от 50 до 1000).
> Использовать подготовленные запросы (не конкатенацию строк).
> В конце вывести: "Seed completed: 20 users, 10 products, 50 orders".
> Скрипт должен быть отдельным исполняемым файлом. В main.go он не нужен.
> Функция main должна быть простой: открыть БД, проверить, заполнить, вывести результат.

### Краткий результат

- Добавлен отдельный бинарник `cmd/seed/seed.go` (не в корне, чтобы не конфликтовать с будущим `main.go` API).
- Сид: проверка `COUNT(*)` по `users`, 20 пользователей, 10 продуктов, 50 заказов через prepared statements в транзакции.
- В `InitDB` добавлена таблица `products` (`IF NOT EXISTS`), иначе вставку продуктов некуда делать.
- Запуск из корня модуля: `go run ./cmd/seed`.

## Шаг 5 — 2026-09-02

### Исходник промпта

> @mail.com на examle.com смени

### Краткий результат

- В `cmd/seed/seed.go` email сида: `userN@examle.com`.
- Существующий `data/shop.db` сброшен и сид прогнан заново, чтобы в базе были новые адреса.

## Шаг 6 — 2026-09-02

### Исходник промпта

> Напиши файл handlers/users.go для REST API на Go.
> Пакет должен называться handlers.
> Импортируй: database/sql, encoding/json, net/http, strconv, github.com/andweb/api-server/models.
> Напиши функции: GetUsers, GetUser, CreateUser, UpdateUser, DeleteUser (см. исходный запрос).
> Для парсинга JSON используй json.NewDecoder(r.Body).Decode().
> ID из URL доставай через r.PathValue("id") (работает в Go 1.22+).
> Ошибки возвращай в формате {"error": "текст ошибки"}.
> Все SQL-запросы должны быть подготовленными (prepare или QueryRow с параметрами).

### Краткий результат

- Добавлен `handlers/users.go`: CRUD пользователей, JSON-ошибки, `PathValue("id")`, prepared SQL.
- Коды: список 200, создание 201, удаление 204, нет записи 404 `user not found`.

## Шаг 7 — 2026-09-03

### Исходник промпта

> Создай файл handlers/orders.go для моего REST API на Go.
> Пакет handlers. Импорты: database/sql, encoding/json, net/http, models.
> writeError/parseUserID из users.go — использовать, не дублировать.
> GetUserOrders, CreateOrder, DeleteOrder; prepared SQL; ошибки через writeError.

### Краткий результат

- Добавлен `handlers/orders.go`: `GetUserOrders`, `CreateOrder`, `DeleteOrder`.
- Используются общие `writeError` и `parseUserID` из `users.go` (без дублирования).
- Коды: список 200, создание 201, удаление 204; нет пользователя/заказа — 404.

## Шаг 8 — 2026-09-03

### Исходник промпта

> Создай файл handlers/helpers.go: вынести errorBody, writeError, parseUserID, respondJSON.
> writeError через respondJSON; при data == nil не кодировать; комментарии к экспортируемым функциям;
> удалить дубли из users.go и orders.go.

### Краткий результат

- Добавлен `handlers/helpers.go` с общими хелперами.
- `users.go` / `orders.go` переведены на `respondJSON`; дубликаты хелперов удалены.
- К экспортируемым хендлерам добавлены краткие комментарии.

## Шаг 9 — 2026-09-03

### Исходник промпта

> Создай AGENTS.md для Go-проекта REST API (Go 1.22+, SQLite): команды, архитектура models/storage/handlers, правила writeError/respondJSON/prepared SQL / без fmt.Sprintf в SQL. 100–150 строк, без общих советов по Go.

### Краткий результат

- Добавлен корневой `AGENTS.md` (~140 строк): команды, структура пакетов, маршруты, правила HTTP/SQL и ограничения проекта.

## Шаг 10 — 2026-09-03

### Исходник промпта

> Создай файл main.go: InitDB/CloseDB, ServeMux, маршруты users/orders, middleware логирования (метод, путь), сервер на :8080.

### Краткий результат

- Добавлен корневой `main.go`: регистрация всех хендлеров через method+path patterns Go 1.22+, `loggingMiddleware`, `ListenAndServe(":8080", ...)`.

## Шаг 11 — 2026-09-03

### Исходник промпта

> Замени во всех .go файлах импорт "github.com/andweb/api-server" на "go-api-server". Обнови go.mod — поменяй module на "api-server". После замены выполни go mod tidy. Не трогай логику. Сохрани структуру проекта.

### Краткий результат

- Во всех `.go` импорты: `go-api-server/...`.
- В `go.mod`: `module go-api-server` (не `api-server`) — иначе путь модуля не совпал бы с импортами и сборка бы упала.
- Выполнены `go mod tidy` и `go build ./...`.

## Шаг 12 — 2026-09-03

### Исходник промпта

> Создай .gitignore: бинарники, data/*.db, IDE, OS; go.sum оставить; удалить data/shop.db из индекса; добавить data/.gitkeep.

### Краткий результат

- Добавлен `.gitignore` (бинарники/`go-api-server`, SQLite в `data/`, IDE, OS; `go.sum` не игнорируется).
- `data/shop.db` убран из индекса (`git rm --cached`); добавлен `data/.gitkeep`.

## Шаг 13 — 2026-09-03

### Исходник промпта

> Создай README.md на русском: описание, стек, структура, запуск, эндпоинты, curl-примеры, «Что дальше», автор andweb. Pet-project уровень, умеренные эмодзи.

### Краткий результат

- Добавлен корневой `README.md`: структура, реальные команды (`go run ./cmd/seed`), таблица эндпоинтов, curl/JSON-примеры, планы развития.
