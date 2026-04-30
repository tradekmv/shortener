package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Pinger interface для проверки соединения с БД
type Pinger interface {
	Ping() error
}

// Database хранит соединение с базой данных
type Database struct {
	db *sql.DB
}

// New создаёт новое подключение к базе данных
func New(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения с БД: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с БД: %w", err)
	}

	return &Database{db: db}, nil
}

// Ping проверяет соединение с базой данных
func (d *Database) Ping() error {
	return d.db.Ping()
}

// Close закрывает соединение с базой данных
func (d *Database) Close() error {
	return d.db.Close()
}

// InitSchema инициализирует схему базы данных (создаёт таблицу urls)
func (d *Database) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_url VARCHAR(32) NOT NULL UNIQUE,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_short_url ON urls(short_url);
	`

	_, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("ошибка создания схемы БД: %w", err)
	}
	return nil
}

// MockDatabase - mock для тестирования
type MockDatabase struct {
	PingFunc func() error
}

func (m *MockDatabase) Ping() error {
	if m.PingFunc != nil {
		return m.PingFunc()
	}
	return nil
}
