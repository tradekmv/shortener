package middleware

import (
	"compress/gzip"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var Log = zerolog.New(os.Stdout).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stderr})

// GzipMiddleware поддерживает сжатие gzip для запросов и ответов
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

func (grw *gzipResponseWriter) Write(p []byte) (int, error) {
	return grw.gz.Write(p)
}

func (grw *gzipResponseWriter) WriteHeader(code int) {
	grw.ResponseWriter.WriteHeader(code)
}

func (grw *gzipResponseWriter) Flush() {
	grw.gz.Flush()
}

func (grw *gzipResponseWriter) Header() http.Header {
	return grw.ResponseWriter.Header()
}

func (grw *gzipResponseWriter) close() {
	grw.gz.Close()
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		Log.Info().
			Str("URI", r.RequestURI).
			Str("method", r.Method).
			Dur("duration", time.Since(start)).
			Int("status", wrapped.statusCode).
			Int64("resp_size", wrapped.written)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}
