package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/tradekmv/shortener.git/internal/auth"
)

func init() {
	// Требуется для всех примеров пакета.
	if len(os.Getenv("AUTH_SECRET_KEY")) < 32 {
		os.Setenv("AUTH_SECRET_KEY", "this-is-a-very-long-test-secret-key-1234567890")
	}
}

// ExampleSignCookie демонстрирует установку подписанной куки.
func ExampleSignCookie() {
	w := httptest.NewRecorder()
	if err := auth.SignCookie(w, "user-abc-123"); err != nil {
		fmt.Println("error:", err)
		return
	}
	res := w.Result()
	defer res.Body.Close()
	c := res.Cookies()
	fmt.Println(len(c) > 0)
	fmt.Println(c[0].Name)
	fmt.Println(c[0].HttpOnly)
	// Output:
	// true
	// user_id
	// true
}

// ExampleGetUserIDFromCookie демонстрирует чтение и проверку куки.
func ExampleGetUserIDFromCookie() {
	// Подготавливаем запрос с уже подписанной кукой.
	w := httptest.NewRecorder()
	if err := auth.SignCookie(w, "user-xyz-789"); err != nil {
		fmt.Println("error:", err)
		return
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := w.Result()
	defer res.Body.Close()
	req.AddCookie(res.Cookies()[0])

	userID, err := auth.GetUserIDFromCookie(req)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(userID)
	// Output:
	// user-xyz-789
}
