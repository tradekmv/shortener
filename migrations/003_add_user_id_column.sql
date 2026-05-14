"-- Добавляем колонку user_id для поддержки аутентификации пользователей
-- Если колонка уже существует, миграция не выполняется

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'urls' AND column_name = 'user_id'
    ) THEN
        ALTER TABLE urls ADD COLUMN user_id VARCHAR(64) NOT NULL DEFAULT '';
        
        -- Создаём индекс для быстрого поиска по user_id
        CREATE INDEX IF NOT EXISTS idx_user_id ON urls(user_id);
    END IF;
END $$;"