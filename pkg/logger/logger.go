package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger — type alias для удобства (позволяет использовать без импорта zerolog)
type Logger = zerolog.Logger

// zerologLogger — адаптер для совместимости (можно использовать если нужен интерфейс)
//type zerologLogger zerolog.Logger

// New создаёт новый логгер с timestamp и консольным выводом
func New() *zerolog.Logger {
	l := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Output(zerolog.ConsoleWriter{Out: os.Stderr})
	return &l
}
