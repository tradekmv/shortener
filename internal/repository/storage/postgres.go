package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// ErrURLAlreadyExists - ошибка, возвращаемая при попытке сохранить уже существующий URL
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
	// Создаём таблицу
	createTable := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_url VARCHAR(32) NOT NULL UNIQUE,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(createTable)
	if err != nil {
		log.Error().Err(err).Msg("Ошибка создания таблицы БД")
		return err
	}

	// Добавляем колонку user_id если её нет
	addUserIDColumn := `
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'urls' AND column_name = 'user_id'
		) THEN
			ALTER TABLE urls ADD COLUMN user_id VARCHAR(64) NOT NULL DEFAULT '';
		END IF;
	END $$;
	`
	_, err = s.db.Exec(addUserIDColumn)
	if err != nil {
		log.Warn().Err(err).Msg("Ошибка добавления колонки user_id (может уже существовать)")
		// Не возвращаем ошибку, т.к. колонка может уже существовать
	}

	// Добавляем колонку is_deleted если её нет
	addDeletedColumn := `
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'urls' AND column_name = 'is_deleted'
		) THEN
			ALTER TABLE urls ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
		END IF;
	END $$;
	`
	_, err = s.db.Exec(addDeletedColumn)
	if err != nil {
		log.Warn().Err(err).Msg("Ошибка добавления колонки is_deleted (может уже существовать)")
	}

	// Создаём индексы
	indexes := `
	CREATE INDEX IF NOT EXISTS idx_short_url ON urls(short_url);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);
	CREATE INDEX IF NOT EXISTS idx_user_id ON urls(user_id);
	`
	_, err = s.db.Exec(indexes)
	if err != nil {
		log.Warn().Err(err).Msg("Ошибка создания индексов")
	}

	return nil
}

// Save сохраняет пару shortURL → originalURL с user_id
// Возвращает ErrURLAlreadyExists если original_url уже существует в базе
func (s *PostgresStorage) Save(ctx context.Context, shortID, originalURL string) error {
	return s.SaveWithUserID(ctx, shortID, originalURL, "")
}

// SaveWithUserID сохраняет пару shortURL → originalURL с указанным user_id
func (s *PostgresStorage) SaveWithUserID(ctx context.Context, shortID, originalURL, userID string) error {
	query := `INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3)`
	_, err := s.db.ExecContext(ctx, query, shortID, originalURL, userID)
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
func (s *PostgresStorage) Get(ctx context.Context, shortID string) (string, error) {
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRow("SELECT original_url, is_deleted FROM urls WHERE short_url = $1", shortID).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		log.Error().Err(err).Msg("Ошибка чтения из БД")
		return "", err
	}
	if isDeleted {
		return "", ErrDeletedGone
	}
	return originalURL, nil
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
func (s *PostgresStorage) SaveBatch(ctx context.Context, urls []URLRecord) ([]URLRecord, error) {
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

// GetUserURLs возвращает все URLs для указанного userID
func (s *PostgresStorage) GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error) {
	query := `SELECT short_url, original_url FROM urls WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Error().Err(err).Msg("Ошибка получения URLs пользователя")
		return nil, err
	}
	defer rows.Close()

	var results []URLRecord
	for rows.Next() {
		var rec URLRecord
		if err := rows.Scan(&rec.ShortURL, &rec.OriginalURL); err != nil {
			log.Error().Err(err).Msg("Ошибка сканирования строки")
			return nil, err
		}
		results = append(results, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// DeleteUserURLs помечает URLs как удалённые для указанного пользователя (async, batch update)
func (s *PostgresStorage) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}

	// Используем множественное обновление с ANY для эффективности
	query := `UPDATE urls SET is_deleted = TRUE WHERE short_url = ANY($1) AND user_id = $2`
	_, err := s.db.ExecContext(ctx, query, shortIDs, userID)
	if err != nil {
		log.Error().Err(err).Msg("Ошибка удаления URLs")
		return fmt.Errorf("ошибка удаления URLs: %w", err)
	}
	return nil
}
