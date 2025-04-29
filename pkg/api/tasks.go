package api

import (
	"go1f/pkg/db"
	"net/http"
	"strconv"
)

// Промежуточная структура для сериализации ID в строку
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

func tasksHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")
	limit := 50

	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			writeError(w, r, http.StatusBadRequest, "некорректный параметр limit")
			return
		}
	}

	tasks, err := db.Tasks(limit, search)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "ошибка получения задач")
		return
	}

	// Преобразование задач в JSON-формат с ID как строкой
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

	writeJSON(w, r, http.StatusOK, TasksResp{Tasks: jsonTasks})
}
