// Package logger provides a thin wrapper around the zerolog library,
// configured with timestamps and a human-readable console writer.
//
// The package exposes a single constructor [New] which returns a pointer
// to a pre-configured [zerolog.Logger] suitable for application-wide use.
// The [Logger] type alias allows callers to refer to the underlying
// zerolog type without importing the third-party package directly.
package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger — type alias для удобства (позволяет использовать без импорта zerolog)
type Logger = zerolog.Logger

// New создаёт новый логгер с timestamp и консольным выводом.
//
// The returned logger writes structured log records to the process's
// standard output, while human-readable console-formatted output is
// mirrored to standard error via [zerolog.ConsoleWriter]. Each record
// is enriched with a Timestamp field for easier debugging.
//
// The function is safe to call once at program startup; the resulting
// logger is safe for concurrent use across goroutines, as required by
// zerolog.
//
// Example:
//
//	logger := logger.New()
//	logger.Info().Msg("service started")
func New() *zerolog.Logger {
	l := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Output(zerolog.ConsoleWriter{Out: os.Stderr})
	return &l
}
