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

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	user, err := s.store.GetUserByID(userID)
	if err != nil {
		slog.Error("ошибка получения пользователя", "error", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	schedules, err := s.store.GetSchedulesByUserID(userID)
	if err != nil {
		slog.Error("ошибка получения расписаний", "error", err)
	}

	s.render(w, "dashboard", PageData{
		Title:     "Личный кабинет",
		User:      user,
		Schedules: schedules,
	})
}

func (s *Server) handleProfileEditPage(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	user, err := s.store.GetUserByID(userID)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	token, _ := GenerateCSRFToken(w)
	s.render(w, "profile_edit", PageData{
		Title:     "Редактирование профиля",
		User:      user,
		CSRFToken: token,
		Success:   r.URL.Query().Get("success"),
	})
}

func (s *Server) handleProfileEditForm(w http.ResponseWriter, r *http.Request) {
	if err := ValidateCSRF(r); err != nil {
		http.Error(w, "Ошибка безопасности (CSRF)", http.StatusForbidden)
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	age, _ := strconv.Atoi(r.FormValue("age"))
	heightCm, _ := strconv.Atoi(r.FormValue("height_cm"))
	weightKg, _ := strconv.ParseFloat(r.FormValue("weight_kg"), 64)

	user := &models.User{
		ID:            userID,
		FirstName:     r.FormValue("first_name"),
		LastName:      r.FormValue("last_name"),
		Age:           age,
		Gender:        r.FormValue("gender"),
		HeightCm:      heightCm,
		WeightKg:      weightKg,
		Goal:          r.FormValue("goal"),
		ActivityLevel: r.FormValue("activity_level"),
	}

	if err := validation.ValidateAge(user.Age); err != nil {
		s.renderProfileEditError(w, user, err.Error())
		return
	}
	if err := validation.ValidateHeightCm(user.HeightCm); err != nil {
		s.renderProfileEditError(w, user, err.Error())
		return
	}
	if err := validation.ValidateWeightKg(user.WeightKg); err != nil {
		s.renderProfileEditError(w, user, err.Error())
		return
	}

	if err := s.store.UpdateUser(user); err != nil {
		slog.Error("ошибка обновления профиля", "error", err)
		s.renderProfileEditError(w, user, "Ошибка сохранения")
		return
	}

	http.Redirect(w, r, "/profile/edit?success=Профиль+обновлён", http.StatusSeeOther)
}

func (s *Server) renderProfileEditError(w http.ResponseWriter, user *models.User, errMsg string) {
	token, _ := GenerateCSRFToken(w)
	s.render(w, "profile_edit", PageData{
		Title:     "Редактирование профиля",
		User:      user,
		Error:     errMsg,
		CSRFToken: token,
	})
}

// --- API handlers ---

func (s *Server) handleAPIProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	user, err := s.store.GetUserByID(userID)
	if err != nil {
		jsonError(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, user)
}

func (s *Server) handleAPIProfileUpdate(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
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
		ID: userID, FirstName: req.FirstName, LastName: req.LastName,
		Age: req.Age, Gender: req.Gender, HeightCm: req.HeightCm,
		WeightKg: req.WeightKg, Goal: req.Goal, ActivityLevel: req.ActivityLevel,
	}

	if err := validation.ValidateAge(user.Age); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateHeightCm(user.HeightCm); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateWeightKg(user.WeightKg); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateUser(user); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Профиль обновлён"})
}
