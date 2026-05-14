package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
)

const (
	cookieName   = "user_id"
	cookieMaxAge = 86400 * 30 // 30 дней
	separator    = "."
)

// ErrInvalidCookie ошибка при невалидной куке
var ErrInvalidCookie = errors.New("невалидная кука")

var (
	secretKey  []byte
	initOnce   sync.Once
	initKeyErr error
)

// initSecret инициализирует секретный ключ один раз
func initSecret() {
	secretKey = []byte("your-secret-key-change-in-production")
}

// getHMAC возвращает HMAC-SHA256 подпись для данных
func getHMAC(data string) string {
	initOnce.Do(initSecret)

	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// generateUserID генерирует уникальный ID пользователя
func generateUserID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SignCookie создаёт подписанную куку с user_id
func SignCookie(w http.ResponseWriter, r *http.Request, userID string) error {
	if userID == "" {
		return errors.New("userID не может быть пустым")
	}

	signature := getHMAC(userID)
	value := userID + separator + signature

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		MaxAge:   cookieMaxAge,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // для локальной разработки
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// GetUserIDFromCookie извлекает и проверяет user_id из куки
// Возвращает userID или ошибку ErrInvalidCookie
func GetUserIDFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", ErrInvalidCookie
	}

	if cookie.Value == "" {
		return "", ErrInvalidCookie
	}

	parts := strings.SplitN(cookie.Value, separator, 2)
	if len(parts) != 2 {
		return "", ErrInvalidCookie
	}

	userID := parts[0]
	signature := parts[1]

	expectedSignature := getHMAC(userID)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", ErrInvalidCookie
	}

	return userID, nil
}

// CreateUserIDIfNeeded создаёт новый userID если кука отсутствует или невалидна
// Устанавливает куку в ответе и возвращает userID
func CreateUserIDIfNeeded(w http.ResponseWriter, r *http.Request) (string, error) {
	userID, err := GetUserIDFromCookie(r)
	if err == nil && userID != "" {
		return userID, nil
	}

	// Создаём новый userID
	userID, err = generateUserID()
	if err != nil {
		return "", err
	}

	// Устанавливаем куку
	if err := SignCookie(w, r, userID); err != nil {
		return "", err
	}

	return userID, nil
}
