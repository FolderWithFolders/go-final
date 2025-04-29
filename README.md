<<<<<<< HEAD
# Планировщик задач (TODO List)

Веб-сервер для управления задачами с поддержкой повторяющихся событий и SQLite-базой данных.

---

## 📌 Основные функции
- Создание/редактирование/удаление задач
- Поддержка повторяющихся задач (ежедневно, еженедельно, ежемесячно, ежегодно)
- Автоматический расчет следующей даты для повторяющихся задач
- Поиск задач по дате, заголовку или комментарию
- Базовая аутентификация через JWT-токен
- Готовый Docker-образ для развертывания

---

## ✅ Выполненны все задания со звёздочкой

---

## 🚀 Запуск локально

### Шаги:
1. Клонировать репозиторий:
- git clone https://github.com/yourusername/todo-app.git
- cd todo-app

2. Настроить переменные окружения (пример `.env`):
- TODO_PORT=7540
- TODO_DBFILE=./scheduler.db
- TODO_PASSWORD="ваш-пароль"

3. Запустить сервер:
   go run main.go

4. Открыть в браузере:  
   http://localhost:7540

---

## 🧪 Запуск тестов

### Настройки тестов (через переменные окружения):
- `TODO_PORT=7540` — порт сервера (по умолчанию 7540)
- `TODO_DBFILE=../scheduler.db` — путь к тестовой БД
- `FULL_NEXTDATE=true` — расширенная проверка дат 
- `SEARCH=true` — тестирование поиска

### Команды:
- Запуск всех тестов:
  go test -v ./...

---

## 🐳 Запуск через Docker

### 1. Dockerfile
FROM golang:latest AS builder \
WORKDIR /app \
COPY go.mod go.sum ./ \
RUN go mod download \
COPY . . \
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./main.go

FROM ubuntu:latest \
WORKDIR /app \
COPY --from=builder /app/server . \
COPY web ./web \
ENV TODO_PORT=7540 \
    TODO_DBFILE=/app/scheduler.db \
    TODO_PASSWORD="ваш-пароль" \
EXPOSE ${TODO_PORT} \
VOLUME /app \
CMD ["./server"] 

### 2. Сборка образа:
docker build -t todo-app .

### 3. Запуск контейнера:
docker run -d \
  -p 7540:7540 \
  -v $(pwd)/scheduler.db:/app/scheduler.db \
  -e TODO_PASSWORD="ваш-пароль" \
  todo-app

---

## 🔧 Технические детали
- **Формат даты**: `YYYYMMDD`
- **Повторяющиеся задачи**:
  - `d 3` — каждые 3 дня
  - `y` — ежегодно
  - `w 1,3,5` — понедельник, среда, пятница
  - `m 5,15 3,6,9` — 5-го и 15-го числа в марте, июне, сентябре
- **База данных**: Файл `scheduler.db` создается автоматически при первом запуске.

---