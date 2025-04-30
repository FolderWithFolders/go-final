package db

import (
	"database/sql"
	"fmt"
	"time"
)

const DateFormat = "20060102"

type Task struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func (s *Store) AddTask(task *Task) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
		task.Date, task.Title, task.Comment, task.Repeat,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) Tasks(limit int, search string) ([]*Task, error) {
	query := "SELECT id, date, title, comment, repeat FROM scheduler"
	args := []interface{}{}

	parsedDate, err := time.Parse("02.01.2006", search)
	if err == nil {
		search = parsedDate.Format(DateFormat)
		query += " WHERE date = ?"
		args = append(args, search)
	} else if search != "" {
		searchTerm := "%" + search + "%"
		query += " WHERE (title LIKE ? OR comment LIKE ?)"
		args = append(args, searchTerm, searchTerm)
	}

	query += " ORDER BY date LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %v", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка чтения данных: %v", err)
		}
		tasks = append(tasks, &t)
	}

	// Проверка на ошибки после итерации
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов: %v", err)
	}

	if tasks == nil {
		tasks = make([]*Task, 0)
	}

	return tasks, nil
}

func (s *Store) GetTask(id string) (*Task, error) {
	var task Task
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?"
	row := s.db.QueryRow(query, id)
	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("задача не найдена")
		}
		return nil, err
	}
	return &task, nil
}

func (s *Store) UpdateTask(task *Task) error {
	query := `UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?`
	res, err := s.db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления: %v", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки обновления: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}
	return nil
}

func (s *Store) DeleteTask(id string) error {
	query := `DELETE FROM scheduler WHERE id = ?`
	res, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки удаления: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}
	return nil
}

func (s *Store) UpdateDate(next string, id string) error {
	query := `UPDATE scheduler SET date = ? WHERE id = ?`
	res, err := s.db.Exec(query, next, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления даты: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки обновления: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}
	return nil
}
