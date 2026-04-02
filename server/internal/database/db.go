package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id          TEXT PRIMARY KEY,
			title       TEXT NOT NULL,
			description TEXT,
			status      TEXT NOT NULL,
			due_date    DATETIME NOT NULL,
			created_at  DATETIME NOT NULL,
			updated_at  DATETIME NOT NULL
		)
	`)
	return err
}

func (db *DB) CreateTask(input model.CreateTaskInput) (*model.Task, error) {
	task := &model.Task{
		ID:          uuid.New().String(),
		Title:       input.Title,
		Description: input.Description,
		Status:      input.Status,
		DueDate:     input.DueDate,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	_, err := db.conn.Exec(`
		INSERT INTO tasks (id, title, description, status, due_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Title, task.Description, string(task.Status),
		task.DueDate, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}

	return task, nil
}

func (db *DB) GetTaskByID(id string) (*model.Task, error) {
	row := db.conn.QueryRow(`
		SELECT id, title, description, status, due_date, created_at, updated_at
		FROM tasks WHERE id = ?`, id)

	return scanTask(row)
}

func (db *DB) GetAllTasks() ([]*model.Task, error) {
	rows, err := db.conn.Query(`
		SELECT id, title, description, status, due_date, created_at, updated_at
		FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (db *DB) UpdateTaskStatus(id string, status model.TaskStatus) (*model.Task, error) {
	now := time.Now().UTC()
	_, err := db.conn.Exec(`
		UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating task status: %w", err)
	}

	return db.GetTaskByID(id)
}

func (db *DB) DeleteTask(id string) error {
	result, err := db.conn.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found: %s", id)
	}
	return nil
}

// scanTask scans a single *sql.Row
func scanTask(row *sql.Row) (*model.Task, error) {
	var task model.Task
	var description sql.NullString
	var status string

	err := row.Scan(
		&task.ID, &task.Title, &description, &status,
		&task.DueDate, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scanning task: %w", err)
	}

	if description.Valid {
		task.Description = &description.String
	}
	task.Status = model.TaskStatus(status)
	return &task, nil
}

// scanTaskRow scans from *sql.Rows
func scanTaskRow(rows *sql.Rows) (*model.Task, error) {
	var task model.Task
	var description sql.NullString
	var status string

	err := rows.Scan(
		&task.ID, &task.Title, &description, &status,
		&task.DueDate, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning task row: %w", err)
	}

	if description.Valid {
		task.Description = &description.String
	}
	task.Status = model.TaskStatus(status)
	return &task, nil
}