package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("AUTH_SECRET_KEY", "test-secret-key-must-be-at-least-32-chars-long")
	os.Setenv("RUN_ENV", "test")
	m.Run()
}

func TestSignCookie_Success(t *testing.T) {
	w := httptest.NewRecorder()
	if err := SignCookie(w, "user-123"); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	result := w.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("ожидалась 1 кука, получено %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != cookieName {
		t.Errorf("ожидалось имя %s, получено %s", cookieName, c.Name)
	}
	if c.Value == "" {
		t.Error("кука не должна быть пустой")
	}
}

func TestSignCookie_EmptyUserID(t *testing.T) {
	w := httptest.NewRecorder()
	if err := SignCookie(w, ""); err == nil {
		t.Error("ожидалась ошибка для пустого userID")
	}
}

func TestGetUserIDFromCookie_NoCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := GetUserIDFromCookie(r); err != ErrInvalidCookie {
		t.Errorf("ожидалась ошибка ErrInvalidCookie, получена: %v", err)
	}
}

func TestGetUserIDFromCookie_EmptyValue(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: ""})
	if _, err := GetUserIDFromCookie(r); err != ErrInvalidCookie {
		t.Errorf("ожидалась ошибка ErrInvalidCookie, получена: %v", err)
	}
}

func TestGetUserIDFromCookie_InvalidFormat(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "no-separator"})
	if _, err := GetUserIDFromCookie(r); err != ErrInvalidCookie {
		t.Errorf("ожидалась ошибка ErrInvalidCookie, получена: %v", err)
	}
}

func TestGetUserIDFromCookie_BadSignature(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "user-1.invalidsignature"})
	if _, err := GetUserIDFromCookie(r); err != ErrInvalidCookie {
		t.Errorf("ожидалась ошибка ErrInvalidCookie, получена: %v", err)
	}
}

func TestCreateUserIDIfNeeded_NoCookie_Creates(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	uid, err := CreateUserIDIfNeeded(w, r)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if uid == "" {
		t.Error("ожидался непустой userID")
	}
	result := w.Result()
	defer result.Body.Close()
	if len(result.Cookies()) != 1 {
		t.Error("ожидалась установленная кука")
	}
}

func TestCreateUserIDIfNeeded_ValidCookie_Returns(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	uid1, _ := CreateUserIDIfNeeded(w, r)
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	result1 := w.Result()
	defer result1.Body.Close()
	for _, c := range result1.Cookies() {
		r2.AddCookie(c)
	}
	uid2, err := CreateUserIDIfNeeded(w2, r2)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if uid1 != uid2 {
		t.Errorf("ожидался тот же userID, получен другой: %s vs %s", uid1, uid2)
	}
}

func TestGetHMAC_Deterministic(t *testing.T) {
	a := getHMAC("user-1")
	b := getHMAC("user-1")
	if a != b {
		t.Error("HMAC должен быть детерминированным")
	}
	if a == "" {
		t.Error("HMAC не должен быть пустым")
	}
}
