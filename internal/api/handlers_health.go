package api

import (
	"log/slog"
	"net/http"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(); err != nil {
		slog.Error("health check failed", "error", err)
		http.Error(w, "db down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleUnsubscribePage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		s.render(w, "unsubscribe", PageData{
			Title: "Отписка",
			Error: "Токен отписки отсутствует",
		})
		return
	}

	_, scheduleID, err := ValidateUnsubscribeToken(s.secretKey, token)
	if err != nil {
		slog.Warn("невалидный токен отписки", "error", err)
		s.render(w, "unsubscribe", PageData{
			Title: "Отписка",
			Error: "Ссылка недействительна или истекла",
		})
		return
	}

	if err := s.store.DeactivateScheduleByID(scheduleID); err != nil {
		slog.Error("ошибка деактивации расписания", "error", err)
		s.render(w, "unsubscribe", PageData{
			Title: "Отписка",
			Error: "Ошибка отписки. Попробуйте позже.",
		})
		return
	}

	s.render(w, "unsubscribe", PageData{
		Title:   "Отписка",
		Success: "Вы успешно отписались от рассылки.",
	})
}

func (s *Server) handleAPIUnsubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		jsonError(w, "Токен отписки отсутствует", http.StatusBadRequest)
		return
	}

	_, scheduleID, err := ValidateUnsubscribeToken(s.secretKey, token)
	if err != nil {
		jsonError(w, "Невалидный токен отписки", http.StatusBadRequest)
		return
	}

	if err := s.store.DeactivateScheduleByID(scheduleID); err != nil {
		jsonError(w, "Ошибка отписки", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Отписка выполнена"})
}
