package config

import (
	"flag"
	"fmt"
	"os"
)

// Config хранит конфигурацию приложения
type Config struct {
	ServerAddress string
	BaseURL       string
}

// Load парсит флаги командной строки и возвращает конфигурацию
func Load() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base URL")

	// Устанавливаем собственный обработчик ошибок для обнаружения неизвестных флагов
	flag.CommandLine.Init(flag.CommandLine.Name(), flag.ContinueOnError)
	flag.CommandLine.SetOutput(&errorWriter{})

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга флагов: %w", err)
	}

	return cfg, nil
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
