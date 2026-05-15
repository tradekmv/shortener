package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LoggingMiddleware создаёт middleware логирования с переданным логгером
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Обработка сжатого тела запроса
		if r.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to read gzip body", http.StatusBadRequest)
				return
			}
			defer reader.Close()
			r.Body = reader
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		shouldCompress := strings.Contains(acceptEncoding, "gzip") && (r.Method == http.MethodPost || r.Method == http.MethodPut)

		if shouldCompress {
			gz := gzip.NewWriter(w)
			wr := &gzipResponseWriter{
				ResponseWriter: w,
				gz:             gz,
			}
			w.Header().Set("Content-Encoding", "gzip")
			w = wr
			defer wr.close()
		}

		next.ServeHTTP(w, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz      *gzip.Writer
	flushed sync.Once
}

// Write переопределяет метод Write для записи через gzip
func (grw *gzipResponseWriter) Write(p []byte) (int, error) {
	return grw.gz.Write(p)
}

// WriteHeader переопределяет WriteHeader для ответа через gzip
func (grw *gzipResponseWriter) WriteHeader(code int) {
	grw.ResponseWriter.WriteHeader(code)
}

// Flush принудительно сбрасывает буфер gzip
func (grw *gzipResponseWriter) Flush() {
	grw.gz.Flush()
}

// Header возвращает заголовки ответа
func (grw *gzipResponseWriter) Header() http.Header {
	return grw.ResponseWriter.Header()
}

func (grw *gzipResponseWriter) close() {
	grw.gz.Close()
}

// LoggingMiddleware логирует HTTP-запросы и ответы
func LoggingMiddleware(next http.Handler, log *zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Info().
			Str("URI", r.RequestURI).
			Str("method", r.Method).
			Dur("duration", time.Since(start)).
			Int("status", wrapped.statusCode).
			Int64("resp_size", wrapped.written)
	})
}

// responseWriter оборачивает http.ResponseWriter для отслеживания статуса
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

// WriteHeader записывает код статуса и отслеживает его
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write записывает тело ответа и отслеживает размер
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}
