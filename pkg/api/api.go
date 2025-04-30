package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go1f/pkg/config"
	"go1f/pkg/dateutil"
	"go1f/pkg/db"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	DateFormat      = "20060102"
	DefaultPageSize = 50
)

type API struct {
	store  *db.Store
	config *config.Config
}

func NewAPI(store *db.Store, cfg *config.Config) *API {
	return &API{store: store, config: cfg}
}

func (a *API) Init() {
	http.HandleFunc("/api/signin", a.handleSignIn)
	http.HandleFunc("/api/nextdate", a.nextDateHandler)
	http.HandleFunc("/api/task", a.authMiddleware(a.taskHandler))
	http.HandleFunc("/api/tasks", a.authMiddleware(a.tasksHandler))
	http.HandleFunc("/api/task/done", a.authMiddleware(a.handleTaskDone))
}

// Структуры для сериализации задач
type JSONTask struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TasksResp struct {
	Tasks []JSONTask `json:"tasks"`
}

// Обработчик GET /api/tasks
func (a *API) tasksHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")
	limit := DefaultPageSize

	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			a.writeError(w, r, http.StatusBadRequest, "некорректный параметр limit")
			return
		}
	}

	tasks, err := a.store.Tasks(limit, search)
	if err != nil {
		a.writeError(w, r, http.StatusInternalServerError, "ошибка получения задач")
		return
	}

	jsonTasks := make([]JSONTask, 0, len(tasks))
	for _, task := range tasks {
		jsonTasks = append(jsonTasks, JSONTask{
			ID:      strconv.FormatInt(task.ID, 10),
			Date:    task.Date,
			Title:   task.Title,
			Comment: task.Comment,
			Repeat:  task.Repeat,
		})
	}

	a.writeJSON(w, r, http.StatusOK, TasksResp{Tasks: jsonTasks})
}

// Обработчик /api/nextdate
func (a *API) nextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowParam := r.FormValue("now")
	dateParam := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Проверка обязательных параметров
	if dateParam == "" || repeat == "" {
		a.writeError(w, r, http.StatusBadRequest, "не указаны параметры date или repeat")
		return
	}

	// Парсинг даты 'now'
	var now time.Time
	if nowParam == "" {
		now = time.Now()
	} else {
		var err error
		now, err = time.Parse(dateutil.DateFormat, nowParam)
		if err != nil {
			a.writeError(w, r, http.StatusBadRequest, "некорректный параметр 'now'")
			return
		}
	}

	// Вызов функции из пакета dateutil
	nextDate, err := dateutil.NextDate(now, dateParam, repeat)
	if err != nil {
		a.writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Отправка ответа
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))

	log.Printf("Запрос /api/nextdate: now=%s, date=%s, repeat=%s", nowParam, dateParam, repeat)
}

// Основной обработчик для /api/task
func (a *API) taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		a.handleDeleteTask(w, r)
	case http.MethodGet:
		a.handleGetTask(w, r)
	case http.MethodPost:
		a.handleAddTask(w, r)
	case http.MethodPut:
		a.handleUpdateTask(w, r)
	default:
		a.writeError(w, r, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

// Обработчик GET /api/task?id=...
func (a *API) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		a.writeError(w, r, http.StatusBadRequest, "Не указан идентификатор")
		return
	}

	task, err := a.store.GetTask(id)
	if err != nil {
		a.writeError(w, r, http.StatusNotFound, err.Error())
		return
	}

	a.writeJSON(w, r, http.StatusOK, map[string]string{
		"id":      strconv.FormatInt(task.ID, 10),
		"date":    task.Date,
		"title":   task.Title,
		"comment": task.Comment,
		"repeat":  task.Repeat,
	})
}

// Обработчик POST /api/task
func (a *API) handleAddTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.writeError(w, r, http.StatusBadRequest, "Ошибка десериализации JSON")
		return
	}

	task := db.Task{
		Date:    request.Date,
		Title:   request.Title,
		Comment: request.Comment,
		Repeat:  request.Repeat,
	}

	if task.Title == "" {
		a.writeError(w, r, http.StatusBadRequest, "Не указан заголовок задачи")
		return
	}

	now := time.Now().Truncate(24 * time.Hour)
	if task.Date == "" {
		task.Date = now.Format(DateFormat)
	}

	t, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		a.writeError(w, r, http.StatusBadRequest, "Некорректный формат даты")
		return
	}
	t = t.Truncate(24 * time.Hour)

	if task.Repeat != "" {
		_, err := dateutil.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			a.writeError(w, r, http.StatusBadRequest, "Некорректное правило повторения")
			return
		}
	}

	if t.Before(now) {
		if task.Repeat == "" {
			task.Date = now.Format(DateFormat)
		} else {
			nextDate, err := dateutil.NextDate(now, task.Date, task.Repeat)
			if err != nil {
				a.writeError(w, r, http.StatusBadRequest, err.Error())
				return
			}
			task.Date = nextDate
		}
	}

	id, err := a.store.AddTask(&task)
	if err != nil {
		a.writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	a.writeJSON(w, r, http.StatusOK, map[string]interface{}{"id": id})
}

// Обработчик PUT /api/task
func (a *API) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID      string `json:"id"`
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.writeError(w, r, http.StatusBadRequest, "Ошибка десериализации JSON")
		return
	}

	id, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		a.writeError(w, r, http.StatusBadRequest, "Некорректный ID задачи")
		return
	}

	task := db.Task{
		ID:      id,
		Date:    request.Date,
		Title:   request.Title,
		Comment: request.Comment,
		Repeat:  request.Repeat,
	}

	if task.Title == "" {
		a.writeError(w, r, http.StatusBadRequest, "Не указан заголовок задачи")
		return
	}

	now := time.Now().Truncate(24 * time.Hour)
	if task.Date == "" {
		task.Date = now.Format(DateFormat)
	}

	parsedDate, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		a.writeError(w, r, http.StatusBadRequest, "Некорректный формат даты")
		return
	}

	if parsedDate.Before(now) && task.Repeat != "" {
		nextDate, err := dateutil.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			a.writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		task.Date = nextDate
	}

	if err := a.store.UpdateTask(&task); err != nil {
		a.writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	a.writeJSON(w, r, http.StatusOK, map[string]interface{}{})
}

// Обработчик DELETE /api/task
func (a *API) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		a.writeError(w, r, http.StatusBadRequest, "Не указан ID")
		return
	}

	if err := a.store.DeleteTask(id); err != nil {
		a.writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	a.writeJSON(w, r, http.StatusOK, map[string]interface{}{})
}

// Обработчик POST /api/task/done
func (a *API) handleTaskDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeError(w, r, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		a.writeError(w, r, http.StatusBadRequest, "Не указан ID")
		return
	}

	task, err := a.store.GetTask(id)
	if err != nil {
		a.writeError(w, r, http.StatusNotFound, err.Error())
		return
	}

	now := time.Now().Truncate(24 * time.Hour)
	if task.Repeat == "" {
		if err := a.store.DeleteTask(id); err != nil {
			a.writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		nextDate, err := dateutil.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			a.writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		if err := a.store.UpdateDate(nextDate, id); err != nil {
			a.writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	a.writeJSON(w, r, http.StatusOK, map[string]interface{}{})
}

// Обработчик POST /api/signin
func (a *API) handleSignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeError(w, r, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	var request struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.writeError(w, r, http.StatusBadRequest, "Ошибка формата запроса")
		return
	}

	envPassword := a.config.Password
	if envPassword == "" {
		a.writeJSON(w, r, http.StatusOK, map[string]string{"token": ""})
		return
	}

	if request.Password != a.config.Password {
		a.writeError(w, r, http.StatusUnauthorized, "Неверный пароль")
		return
	}

	hash := sha256.Sum256([]byte(envPassword))
	claims := jwt.MapClaims{
		"hash": hex.EncodeToString(hash[:]),
		"exp":  time.Now().Add(8 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(envPassword))
	if err != nil {
		a.writeError(w, r, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(8 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
	})

	a.writeJSON(w, r, http.StatusOK, map[string]string{"token": tokenString})
}

// Middleware для аутентификации
func (a *API) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.config.Password == "" { // Теперь берем из конфига
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("token")
		if err != nil {
			a.writeError(w, r, http.StatusUnauthorized, "Требуется аутентификация")
			return
		}

		token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			return []byte(a.config.Password), nil
		})
		if err != nil || !token.Valid {
			a.writeError(w, r, http.StatusUnauthorized, "Неверный токен")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			a.writeError(w, r, http.StatusUnauthorized, "Ошибка формата токена")
			return
		}

		currentHash := sha256.Sum256([]byte(a.config.Password))
		if hex.EncodeToString(currentHash[:]) != claims["hash"] {
			a.writeError(w, r, http.StatusUnauthorized, "Токен устарел")
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Вспомогательные методы
func (a *API) writeJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

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

func (a *API) writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	a.writeJSON(w, r, status, map[string]string{"error": message})
}
