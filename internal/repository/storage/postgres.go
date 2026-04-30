package storage

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// PostgresStorage хранит данные в PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgres создаёт новое PostgreSQL хранилище
func NewPostgres(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения с БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с БД: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	s := &PostgresStorage{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("ошибка миграции: %w", err)
	}

	return s, nil
}

// migrate создаёт таблицы и индексы
func (s *PostgresStorage) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_url VARCHAR(32) NOT NULL UNIQUE,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_short_url ON urls(short_url);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		log.Error().Err(err).Msg("Ошибка миграции БД")
		return err
	}
	return nil
}

// Save сохраняет пару shortURL → originalURL
func (s *PostgresStorage) Save(shortID, originalURL string) error {
	query := `INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING`
	result, err := s.db.Exec(query, shortID, originalURL)
	if err != nil {
		return fmt.Errorf("ошибка сохранения в БД: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrAlreadyExists
	}

	return nil
}

// Get возвращает originalURL по shortID
func (s *PostgresStorage) Get(shortID string) (string, bool) {
	var originalURL string
	err := s.db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", shortID).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false
		}
		log.Error().Err(err).Msg("Ошибка чтения из БД")
		return "", false
	}
	return originalURL, true
}

// Close закрывает соединение с БД
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// Ping проверяет соединение с БД
func (s *PostgresStorage) Ping() error {
	return s.db.Ping()
}
