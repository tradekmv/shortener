package config

import (
	"flag"
	"fmt"
	"os"
)

var flagsParsed bool

// Config хранит конфигурацию приложения
type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
}

// Load парсит конфигурацию с приоритетом: env > флаг > значение по умолчанию
func Load() (*Config, error) {
	cfg := &Config{}

	// Регистрируем флаги только один раз
	if !flagsParsed {
		flag.StringVar(&cfg.ServerAddress, "a", "", "server address")
		flag.StringVar(&cfg.BaseURL, "b", "", "base URL")
		flag.StringVar(&cfg.FileStoragePath, "f", "", "file storage path")
		flag.StringVar(&cfg.FileStoragePath, "file-storage-path", "", "file storage path")
		flagsParsed = true
	}

	// Значения по умолчанию
	defaultServerAddress := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"
	defaultFileStoragePath := "storage.json"

	// Устанавливаем собственный обработчик ошибок
	flag.CommandLine.Init(flag.CommandLine.Name(), flag.ContinueOnError)
	flag.CommandLine.SetOutput(&errorWriter{})

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга флагов: %w", err)
	}

	// Приоритет: env > флаг > умолчание
	if val := os.Getenv("SERVER_ADDRESS"); val != "" {
		cfg.ServerAddress = val
	} else if cfg.ServerAddress == "" {
		cfg.ServerAddress = defaultServerAddress
	}

	if val := os.Getenv("BASE_URL"); val != "" {
		cfg.BaseURL = val
	} else if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	if val := os.Getenv("FILE_STORAGE_PATH"); val != "" {
		cfg.FileStoragePath = val
	} else if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = defaultFileStoragePath
	}

	return cfg, nil
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
