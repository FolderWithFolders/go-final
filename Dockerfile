# Этап сборки приложения
FROM golang:1.23.3-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей и загружаем их
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код и компилируем приложение
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./main.go

# Этап запуска
FROM alpine:3.20

WORKDIR /app

# Копируем бинарник и статические файлы фронтенда
COPY --from=builder /app/server .
COPY web ./web

# Задаём переменные окружения (значения по умолчанию)
ENV TODO_PORT=7540 \
    TODO_DBFILE=/app/scheduler.db \
    TODO_PASSWORD=""

# Открываем порт и указываем точку монтирования для БД
EXPOSE ${TODO_PORT}
VOLUME /app

# Команда запуска сервера
CMD ["./server"]