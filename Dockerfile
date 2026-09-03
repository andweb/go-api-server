# Этап 1: сборка статического бинарника (CGO выключен — modernc.org/sqlite без CGO).
FROM golang:1.22-alpine AS builder

WORKDIR /app

# go.mod требует go >= 1.25 — auto подтянет toolchain; CGO off для статики.
ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=auto

# Кэш зависимостей: сначала только модули.
COPY go.mod go.sum ./
RUN go mod download

# Исходники и сборка.
COPY . .
RUN go build -o api-server .

# Этап 2: минимальный runtime-образ (alpine — есть FS для SQLite в data/).
FROM alpine:latest

WORKDIR /app

# Только бинарник и каталог data/ (схема/файл БД создаст InitDB при старте).
COPY --from=builder /app/api-server .
COPY --from=builder /app/data ./data

EXPOSE 8080

CMD ["./api-server"]
