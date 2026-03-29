package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"daily-email-sender/internal/auth"
	"daily-email-sender/internal/models"
	"daily-email-sender/internal/validation"
)

// --- SSR handlers ---

func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	token, _ := GenerateCSRFToken(w)
	s.render(w, "register", PageData{
		Title:     "Регистрация",
		CSRFToken: token,
		Success:   r.URL.Query().Get("success"),
	})
}

func (s *Server) handleRegisterForm(w http.ResponseWriter, r *http.Request) {
	if checkHoneypot(r) {
		w.WriteHeader(http.StatusOK)
		return
	}

	ip := clientIP(r)
	if !s.rateLimiter.Allow(ip, "register") {
		http.Error(w, "Слишком много запросов. Попробуйте позже.", http.StatusTooManyRequests)
		return
	}

	if err := ValidateCSRF(r); err != nil {
		http.Error(w, "Ошибка безопасности (CSRF)", http.StatusForbidden)
		return
	}

	user, password, form, err := parseRegisterForm(r)
	if err != nil {
		token, _ := GenerateCSRFToken(w)
		s.render(w, "register", PageData{
			Title:     "Регистрация",
			Error:     err.Error(),
			CSRFToken: token,
			Form:      form,
		})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("ошибка хеширования", "error", err)
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if err := s.store.CreateUserWithPassword(user, hash); err != nil {
		token, _ := GenerateCSRFToken(w)
		s.render(w, "register", PageData{
			Title:     "Регистрация",
			Error:     err.Error(),
			CSRFToken: token,
			Form:      form,
		})
		return
	}

	slog.Info("новый пользователь зарегистрирован", "email", user.Email, "id", user.ID)
	http.Redirect(w, r, "/login?success=Регистрация+успешна.+Войдите+в+аккаунт.", http.StatusSeeOther)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	token, _ := GenerateCSRFToken(w)
	s.render(w, "login", PageData{
		Title:     "Вход",
		CSRFToken: token,
		Success:   r.URL.Query().Get("success"),
	})
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	if checkHoneypot(r) {
		w.WriteHeader(http.StatusOK)
		return
	}

	ip := clientIP(r)
	if !s.rateLimiter.Allow(ip, "login") {
		http.Error(w, "Слишком много запросов. Попробуйте позже.", http.StatusTooManyRequests)
		return
	}

	if err := ValidateCSRF(r); err != nil {
		http.Error(w, "Ошибка безопасности (CSRF)", http.StatusForbidden)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	loginErr := "Неверный email или пароль"

	if s.loginBlocker.IsBlocked(email) {
		token, _ := GenerateCSRFToken(w)
		s.render(w, "login", PageData{
			Title:     "Вход",
			Error:     "Слишком много неудачных попыток. Попробуйте позже.",
			CSRFToken: token,
		})
		return
	}

	userID, hash, err := s.store.GetPasswordHashByEmail(email)
	if err != nil {
		s.loginBlocker.RecordFailure(email)
		token, _ := GenerateCSRFToken(w)
		s.render(w, "login", PageData{
			Title:     "Вход",
			Error:     loginErr,
			CSRFToken: token,
		})
		return
	}

	if hash == "" || auth.CheckPassword(hash, password) != nil {
		s.loginBlocker.RecordFailure(email)
		token, _ := GenerateCSRFToken(w)
		s.render(w, "login", PageData{
			Title:     "Вход",
			Error:     loginErr,
			CSRFToken: token,
		})
		return
	}

	s.loginBlocker.RecordSuccess(email)

	sessionToken, err := s.sessions.Create(userID)
	if err != nil {
		slog.Error("ошибка создания сессии", "error", err)
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	auth.SetCookie(w, sessionToken)
	slog.Info("пользователь вошёл", "email", email, "userID", userID)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		s.sessions.Delete(cookie.Value)
	}
	auth.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// --- API handlers ---

func (s *Server) handleAPIRegister(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.rateLimiter.Allow(ip, "register") {
		jsonError(w, "Слишком много запросов", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Email         string  `json:"email"`
		Password      string  `json:"password"`
		FirstName     string  `json:"first_name"`
		LastName      string  `json:"last_name"`
		Age           int     `json:"age"`
		Gender        string  `json:"gender"`
		HeightCm      int     `json:"height_cm"`
		WeightKg      float64 `json:"weight_kg"`
		Goal          string  `json:"goal"`
		ActivityLevel string  `json:"activity_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	user := &models.User{
		Email: req.Email, FirstName: req.FirstName, LastName: req.LastName,
		Age: req.Age, Gender: req.Gender, HeightCm: req.HeightCm,
		WeightKg: req.WeightKg, Goal: req.Goal, ActivityLevel: req.ActivityLevel,
	}

	if err := validateUserFields(user, req.Password); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if err := s.store.CreateUserWithPassword(user, hash); err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{
		"id":      user.ID,
		"message": "Регистрация успешна",
	})
}

func (s *Server) handleAPILogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.rateLimiter.Allow(ip, "login") {
		jsonError(w, "Слишком много запросов", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	loginErr := "Неверный email или пароль"

	if s.loginBlocker.IsBlocked(req.Email) {
		jsonError(w, "Слишком много неудачных попыток. Попробуйте позже.", http.StatusTooManyRequests)
		return
	}

	userID, hash, err := s.store.GetPasswordHashByEmail(req.Email)
	if err != nil || hash == "" || auth.CheckPassword(hash, req.Password) != nil {
		s.loginBlocker.RecordFailure(req.Email)
		jsonError(w, loginErr, http.StatusUnauthorized)
		return
	}

	s.loginBlocker.RecordSuccess(req.Email)

	sessionToken, err := s.sessions.Create(userID)
	if err != nil {
		jsonError(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	auth.SetCookie(w, sessionToken)
	jsonResponse(w, http.StatusOK, map[string]string{
		"token":   sessionToken,
		"user_id": userID,
	})
}

// --- Helpers ---

func parseRegisterForm(r *http.Request) (*models.User, string, map[string]string, error) {
	form := map[string]string{
		"email":          r.FormValue("email"),
		"first_name":     r.FormValue("first_name"),
		"last_name":      r.FormValue("last_name"),
		"age":            r.FormValue("age"),
		"gender":         r.FormValue("gender"),
		"height_cm":      r.FormValue("height_cm"),
		"weight_kg":      r.FormValue("weight_kg"),
		"goal":           r.FormValue("goal"),
		"activity_level": r.FormValue("activity_level"),
	}

	password := r.FormValue("password")

	age, _ := strconv.Atoi(form["age"])
	heightCm, _ := strconv.Atoi(form["height_cm"])
	weightKg, _ := strconv.ParseFloat(form["weight_kg"], 64)

	user := &models.User{
		Email:         form["email"],
		FirstName:     form["first_name"],
		LastName:      form["last_name"],
		Age:           age,
		Gender:        form["gender"],
		HeightCm:      heightCm,
		WeightKg:      weightKg,
		Goal:          form["goal"],
		ActivityLevel: form["activity_level"],
	}

	if err := validateUserFields(user, password); err != nil {
		return nil, "", form, err
	}

	return user, password, form, nil
}

func validateUserFields(user *models.User, password string) error {
	if err := validation.ValidateEmail(user.Email); err != nil {
		return err
	}
	if err := validation.ValidatePassword(password); err != nil {
		return err
	}
	if err := validation.ValidateAge(user.Age); err != nil {
		return err
	}
	if err := validation.ValidateHeightCm(user.HeightCm); err != nil {
		return err
	}
	if err := validation.ValidateWeightKg(user.WeightKg); err != nil {
		return err
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	jsonResponse(w, status, map[string]string{"error": message})
}
