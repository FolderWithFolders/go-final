package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go1f/pkg/db"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DateFormat = "20060102"

func Init() {
	http.HandleFunc("/api/signin", handleSignIn)
	http.HandleFunc("/api/nextdate", nextDateHandler)
	http.HandleFunc("/api/task", authMiddleware(taskHandler))
	http.HandleFunc("/api/tasks", authMiddleware(tasksHandler))
	http.HandleFunc("/api/task/done", authMiddleware(handleTaskDone))
}

// Обработчик для /api/nextdate
func nextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowParam := r.FormValue("now")
	dateParam := r.FormValue("date")
	repeat := r.FormValue("repeat")

	var now time.Time
	if nowParam == "" {
		now = time.Now()
	} else {
		var err error
		now, err = time.Parse(DateFormat, nowParam)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "некорректный параметр 'now'")
			return
		}
	}

	nextDate, err := NextDate(now, dateParam, repeat)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}

// Основной обработчик для /api/task
func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		handleDeleteTask(w, r)
	case http.MethodGet:
		handleGetTask(w, r)
	case http.MethodPost:
		handleAddTask(w, r)
	case http.MethodPut:
		handleUpdateTask(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

// Обработчик GET /api/task?id=...
func handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, r, http.StatusBadRequest, "Не указан идентификатор")
		return
	}

	task, err := db.GetTask(id)
	if err != nil {
		writeError(w, r, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]string{
		"id":      strconv.FormatInt(task.ID, 10),
		"date":    task.Date,
		"title":   task.Title,
		"comment": task.Comment,
		"repeat":  task.Repeat,
	})
}

// Обработчик POST /api/task
// Обработчик POST /api/task
func handleAddTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, r, http.StatusBadRequest, "Ошибка десериализации JSON")
		return
	}

	task := db.Task{
		Date:    request.Date,
		Title:   request.Title,
		Comment: request.Comment,
		Repeat:  request.Repeat,
	}

	if task.Title == "" {
		writeError(w, r, http.StatusBadRequest, "Не указан заголовок задачи")
		return
	}

	now := time.Now().Truncate(24 * time.Hour)
	if task.Date == "" {
		task.Date = now.Format(DateFormat)
	}

	t, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Некорректный формат даты")
		return
	}
	t = t.Truncate(24 * time.Hour)

	if task.Repeat != "" {
		_, err := NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "Некорректное правило повторения")
			return
		}
	}

	if t.Before(now) {
		if task.Repeat == "" {
			task.Date = now.Format(DateFormat)
		} else {
			nextDate, err := NextDate(now, task.Date, task.Repeat)
			if err != nil {
				writeError(w, r, http.StatusBadRequest, err.Error())
				return
			}
			task.Date = nextDate
		}
	}

	// Добавляем задачу и получаем сгенерированный ID
	id, err := db.AddTask(&task)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{"id": id})
}

// Обработчик PUT /api/task
func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID      string `json:"id"`
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, r, http.StatusBadRequest, "Ошибка десериализации JSON")
		return
	}

	// Преобразование строки ID в int64
	id, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Некорректный ID задачи")
		return
	}

	task := db.Task{
		ID:      id,
		Date:    request.Date,
		Title:   request.Title,
		Comment: request.Comment,
		Repeat:  request.Repeat,
	}

	// Проверка обязательных полей
	if task.Title == "" {
		writeError(w, r, http.StatusBadRequest, "Не указан заголовок задачи")
		return
	}

	// Коррекция даты
	now := time.Now().Truncate(24 * time.Hour)
	if task.Date == "" {
		task.Date = now.Format(DateFormat)
	}

	parsedDate, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Некорректный формат даты")
		return
	}

	// Если дата в прошлом и есть правило повторения
	if parsedDate.Before(now) && task.Repeat != "" {
		nextDate, err := NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		task.Date = nextDate
	}

	// Обновление задачи в БД
	if err := db.UpdateTask(&task); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{})
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, r, http.StatusBadRequest, "Не указан ID")
		return
	}

	if err := db.DeleteTask(id); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{})
}

// Обработчик POST /api/task/done
func handleTaskDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, r, http.StatusBadRequest, "Не указан ID")
		return
	}

	// Получаем задачу
	task, err := db.GetTask(id)
	if err != nil {
		writeError(w, r, http.StatusNotFound, err.Error())
		return
	}

	now := time.Now().Truncate(24 * time.Hour)
	if task.Repeat == "" {
		// Удаляем одноразовую задачу
		if err := db.DeleteTask(id); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		// Рассчитываем следующую дату
		nextDate, err := NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		// Обновляем дату
		if err := db.UpdateDate(nextDate, id); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{})
}

// Обработчик POST /api/signin
func handleSignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	var request struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, r, http.StatusBadRequest, "Ошибка формата запроса")
		return
	}

	envPassword := os.Getenv("TODO_PASSWORD")
	if envPassword == "" {
		writeJSON(w, r, http.StatusOK, map[string]string{"token": ""})
		return
	}

	if request.Password != envPassword {
		writeError(w, r, http.StatusUnauthorized, "Неверный пароль")
		return
	}

	// Генерация JWT-токена
	hash := sha256.Sum256([]byte(envPassword))
	claims := jwt.MapClaims{
		"hash": hex.EncodeToString(hash[:]),
		"exp":  time.Now().Add(8 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(envPassword))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}

	// Установка куки
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(8 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Для HTTPS установить true
	})

	writeJSON(w, r, http.StatusOK, map[string]string{"token": tokenString})
}

// Middleware для проверки аутентификации (исправленная версия)
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		envPassword := os.Getenv("TODO_PASSWORD")
		if envPassword == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Проверка куки
		cookie, err := r.Cookie("token")
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "Требуется аутентификация")
			return
		}

		// Валидация токена
		token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			return []byte(envPassword), nil
		})
		if err != nil || !token.Valid {
			writeError(w, r, http.StatusUnauthorized, "Неверный токен")
			return
		}

		// Проверка хэша пароля
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, r, http.StatusUnauthorized, "Ошибка формата токена")
			return
		}

		currentHash := sha256.Sum256([]byte(envPassword))
		if hex.EncodeToString(currentHash[:]) != claims["hash"] {
			writeError(w, r, http.StatusUnauthorized, "Токен устарел")
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Вспомогательные функции// Вспомогательные функции
func writeJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Установка куки для /api/signin
	if r != nil && strings.Contains(r.URL.Path, "/api/signin") {
		if m, ok := data.(map[string]string); ok && m["token"] != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    m["token"],
				Expires:  time.Now().Add(8 * time.Hour),
				Path:     "/",
				HttpOnly: true,
			})
		}
	}

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Ошибка сериализации JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, r, status, map[string]string{"error": message})
}
