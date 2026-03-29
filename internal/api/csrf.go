package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
)

const (
	csrfCookieName = "csrf_token"
	csrfFieldName  = "csrf_token"
	csrfTokenBytes = 32
)

// GenerateCSRFToken создаёт новый CSRF-токен и устанавливает cookie.
func GenerateCSRFToken(w http.ResponseWriter) (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("ошибка генерации CSRF-токена: %w", err)
	}
	token := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	return token, nil
}

// ValidateCSRF проверяет CSRF-токен из формы против cookie.
// Возвращает ошибку если токены не совпадают.
func ValidateCSRF(r *http.Request) error {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return fmt.Errorf("CSRF-токен отсутствует")
	}

	formToken := r.FormValue(csrfFieldName)
	if formToken == "" {
		return fmt.Errorf("CSRF-токен не передан в форме")
	}

	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) != 1 {
		return fmt.Errorf("CSRF-токен не совпадает")
	}

	return nil
}
