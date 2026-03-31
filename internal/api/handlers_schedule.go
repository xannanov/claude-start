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

func (s *Server) handleSchedulesPage(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	schedules, err := s.store.GetSchedulesByUserID(userID)
	if err != nil {
		slog.Error("ошибка получения расписаний", "error", err)
	}

	token, _ := GenerateCSRFToken(w)
	s.render(w, "schedules", PageData{
		Title:     "Управление расписанием",
		Schedules: schedules,
		CSRFToken: token,
		Success:   r.URL.Query().Get("success"),
		Error:     r.URL.Query().Get("error"),
	})
}

func (s *Server) handleScheduleAdd(w http.ResponseWriter, r *http.Request) {
	if err := ValidateCSRF(r); err != nil {
		http.Error(w, "Ошибка безопасности (CSRF)", http.StatusForbidden)
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	dayOfWeek, _ := strconv.Atoi(r.FormValue("day_of_week"))
	timeHour, _ := strconv.Atoi(r.FormValue("time_hour"))
	timeMinute, _ := strconv.Atoi(r.FormValue("time_minute"))

	if err := validation.ValidateDayOfWeek(dayOfWeek); err != nil {
		http.Redirect(w, r, "/schedules?error="+err.Error(), http.StatusSeeOther)
		return
	}
	if err := validation.ValidateHour(timeHour); err != nil {
		http.Redirect(w, r, "/schedules?error="+err.Error(), http.StatusSeeOther)
		return
	}
	if err := validation.ValidateMinute(timeMinute); err != nil {
		http.Redirect(w, r, "/schedules?error="+err.Error(), http.StatusSeeOther)
		return
	}

	count, err := s.store.CountActiveSchedulesByUserID(userID)
	if err != nil {
		slog.Error("ошибка подсчёта расписаний", "error", err)
		http.Redirect(w, r, "/schedules?error=Ошибка+проверки+лимита", http.StatusSeeOther)
		return
	}
	if count >= 14 {
		http.Redirect(w, r, "/schedules?error=Достигнут+лимит+расписаний+(максимум+14)", http.StatusSeeOther)
		return
	}

	emailType := emailTypeFromHour(timeHour)

	schedule := &models.UserSchedule{
		UserID:     userID,
		DayOfWeek:  dayOfWeek,
		TimeHour:   timeHour,
		TimeMinute: timeMinute,
		EmailType:  emailType,
	}

	if err := s.store.CreateUserSchedule(schedule); err != nil {
		slog.Error("ошибка создания расписания", "error", err)
		msg := "Ошибка создания расписания"
		if err.Error() == "Расписание на этот день и время уже существует" {
			msg = err.Error()
		}
		schedules, _ := s.store.GetSchedulesByUserID(userID)
		token, _ := GenerateCSRFToken(w)
		s.render(w, "schedules", PageData{
			Title:     "Управление расписанием",
			Error:     msg,
			Schedules: schedules,
			CSRFToken: token,
		})
		return
	}

	http.Redirect(w, r, "/schedules?success=Расписание+добавлено", http.StatusSeeOther)
}

func (s *Server) handleScheduleDelete(w http.ResponseWriter, r *http.Request) {
	if err := ValidateCSRF(r); err != nil {
		http.Error(w, "Ошибка безопасности (CSRF)", http.StatusForbidden)
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	scheduleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Redirect(w, r, "/schedules?error=Некорректный+ID", http.StatusSeeOther)
		return
	}

	if err := s.store.DeleteSchedule(scheduleID, userID); err != nil {
		slog.Error("ошибка удаления расписания", "error", err)
		http.Redirect(w, r, "/schedules?error=Ошибка+удаления", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/schedules?success=Расписание+удалено", http.StatusSeeOther)
}

// --- API handlers ---

func (s *Server) handleAPISchedules(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	schedules, err := s.store.GetSchedulesByUserID(userID)
	if err != nil {
		jsonError(w, "Ошибка получения расписаний", http.StatusInternalServerError)
		return
	}
	if schedules == nil {
		schedules = []models.UserSchedule{}
	}

	jsonResponse(w, http.StatusOK, schedules)
}

func (s *Server) handleAPIScheduleCreate(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		DayOfWeek  int `json:"day_of_week"`
		TimeHour   int `json:"time_hour"`
		TimeMinute int `json:"time_minute"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	if err := validation.ValidateDayOfWeek(req.DayOfWeek); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateHour(req.TimeHour); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateMinute(req.TimeMinute); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	count, err := s.store.CountActiveSchedulesByUserID(userID)
	if err != nil {
		jsonError(w, "Ошибка проверки лимита расписаний", http.StatusInternalServerError)
		return
	}
	if count >= 14 {
		jsonError(w, "Достигнут лимит расписаний (максимум 14)", http.StatusUnprocessableEntity)
		return
	}

	schedule := &models.UserSchedule{
		UserID:     userID,
		DayOfWeek:  req.DayOfWeek,
		TimeHour:   req.TimeHour,
		TimeMinute: req.TimeMinute,
		EmailType:  emailTypeFromHour(req.TimeHour),
	}

	if err := s.store.CreateUserSchedule(schedule); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusCreated, schedule)
}

func (s *Server) handleAPIScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	scheduleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		jsonError(w, "Некорректный ID расписания", http.StatusBadRequest)
		return
	}

	var req struct {
		DayOfWeek  int `json:"day_of_week"`
		TimeHour   int `json:"time_hour"`
		TimeMinute int `json:"time_minute"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	if err := validation.ValidateDayOfWeek(req.DayOfWeek); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateHour(req.TimeHour); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateMinute(req.TimeMinute); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	schedule := &models.UserSchedule{
		ID:         scheduleID,
		UserID:     userID,
		DayOfWeek:  req.DayOfWeek,
		TimeHour:   req.TimeHour,
		TimeMinute: req.TimeMinute,
		EmailType:  emailTypeFromHour(req.TimeHour),
	}

	if err := s.store.UpdateSchedule(schedule); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Расписание обновлено"})
}

func (s *Server) handleAPIScheduleDelete(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	scheduleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		jsonError(w, "Некорректный ID расписания", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteSchedule(scheduleID, userID); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Расписание удалено"})
}

func emailTypeFromHour(hour int) string {
	switch {
	case hour < 12:
		return "morning"
	case hour < 17:
		return "afternoon"
	default:
		return "evening"
	}
}
