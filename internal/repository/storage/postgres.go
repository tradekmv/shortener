package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// ErrURLAlreadyExists - ошибка, возвращаемая при попытке сохранить уже существующий URL
var ErrURLAlreadyExists = errors.New("URL уже существует")

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
	CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		log.Error().Err(err).Msg("Ошибка миграции БД")
		return err
	}
	return nil
}

// Save сохраняет пару shortURL → originalURL
// Возвращает ErrURLAlreadyExists если original_url уже существует в базе
func (s *PostgresStorage) Save(shortID, originalURL string) error {
	query := `INSERT INTO urls (short_url, original_url) VALUES ($1, $2)`
	_, err := s.db.Exec(query, shortID, originalURL)
	if err != nil {
		// Проверяем, является ли ошибка нарушением уникального индекса (pq.Error используется с github.com/lib/pq)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // unique_violation
				return ErrURLAlreadyExists
			}
			log.Warn().Err(err).Str("code", string(pqErr.Code)).Msg("PostgreSQL error")
		}
		return fmt.Errorf("ошибка сохранения в БД: %w", err)
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

// GetByOriginalURL возвращает shortURL по originalURL
func (s *PostgresStorage) GetByOriginalURL(originalURL string) (string, bool) {
	var shortURL string
	err := s.db.QueryRow("SELECT short_url FROM urls WHERE original_url = $1", originalURL).Scan(&shortURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false
		}
		log.Error().Err(err).Msg("Ошибка чтения из БД")
		return "", false
	}
	return shortURL, true
}

// SaveBatch saves multiple URLs in one transaction for PostgreSQL
func (s *PostgresStorage) SaveBatch(urls []URLRecord) ([]URLRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	result := make([]URLRecord, 0, len(urls))
	for _, rec := range urls {
		_, err := tx.Exec(
			`INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO UPDATE SET short_url = EXCLUDED.short_url RETURNING id`,
			rec.ShortURL,
			rec.OriginalURL,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка вставки в БД: %w", err)
		}
		result = append(result, URLRecord{
			ShortURL:    rec.ShortURL,
			OriginalURL: rec.OriginalURL,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}
	return result, nil
}

// Close закрывает соединение с БД
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// Ping проверяет соединение с БД
func (s *PostgresStorage) Ping() error {
	return s.db.Ping()
}
