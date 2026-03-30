package auth

import (
	"context"
	"net/http"
)

// contextKey — тип для ключей контекста (избегаем коллизий).
type contextKey string

const userIDKey contextKey = "userID"

// RequireAuth возвращает middleware, проверяющий авторизацию.
// Если сессия невалидна — 401 Unauthorized.
// Если валидна — userID помещается в context запроса.
func RequireAuth(sm *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
				return
			}

			session, err := sm.Validate(cookie.Value)
			if err != nil {
				ClearCookie(w)
				http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext извлекает userID из контекста запроса.
// Возвращает пустую строку если userID отсутствует.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// ContextWithUserID помещает userID в контекст (используется в кастомных middleware).
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
