-- Add unique index on original_url to prevent duplicate URLs
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);
