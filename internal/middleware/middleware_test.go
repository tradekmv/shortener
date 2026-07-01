package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func newLogger() *zerolog.Logger {
	l := zerolog.New(io.Discard)
	return &l
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello"))
}

func TestGzipMiddleware_PassThrough_NoEncoding(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(okHandler))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("ожидался статус 200, получен %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("неожиданное тело: %s", w.Body.String())
	}
}

func TestGzipMiddleware_CompressResponse(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(okHandler))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("ожидался Content-Encoding: gzip")
	}
}

func TestGzipMiddleware_DecompressRequest(t *testing.T) {
	called := false
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" {
			t.Errorf("неожиданное тело: %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte("hello"))
	zw.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Encoding", "gzip")
	h.ServeHTTP(w, r)
	if !called {
		t.Error("хендлер не был вызван")
	}
}

func TestGzipMiddleware_DecompressRequest_BadGzip(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(okHandler))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-gzip"))
	r.Header.Set("Content-Encoding", "gzip")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус 400, получен %d", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	h := LoggingMiddleware(http.HandlerFunc(okHandler), newLogger())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("ожидался статус 200, получен %d", w.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("ожидался статус 404, получен %d", rw.statusCode)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	n, err := rw.Write([]byte("abc"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("ожидалось n=3, получено %d", n)
	}
	if rw.written != 3 {
		t.Errorf("ожидался written=3, получено %d", rw.written)
	}
}

func TestGzipResponseWriter_Header(t *testing.T) {
	w := httptest.NewRecorder()
	gz := gzip.NewWriter(io.Discard)
	grw := &gzipResponseWriter{ResponseWriter: w, gz: gz}
	if grw.Header() == nil {
		t.Error("Header не должен быть nil")
	}
	grw.gz.Close()
}
