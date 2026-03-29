package api

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"daily-email-sender/internal/auth"
	"daily-email-sender/internal/database"
)

//go:embed templates/*.html
var templatesFS embed.FS

// DayNames — названия дней недели на русском (0=пн, 6=вс).
var DayNames = map[int]string{
	0: "Понедельник",
	1: "Вторник",
	2: "Среда",
	3: "Четверг",
	4: "Пятница",
	5: "Суббота",
	6: "Воскресенье",
}

// GoalNames — названия целей на русском.
var GoalNames = map[string]string{
	"weight_loss":     "Похудение",
	"muscle_gain":     "Набор мышечной массы",
	"maintenance":     "Поддержание формы",
	"general_fitness": "Общий фитнес",
}

// ActivityNames — названия уровней активности на русском.
var ActivityNames = map[string]string{
	"sedentary":   "Сидячий",
	"light":       "Лёгкая активность",
	"moderate":    "Умеренная активность",
	"active":      "Активный",
	"very_active": "Очень активный",
}

// PageData — данные для рендеринга HTML-шаблонов.
type PageData struct {
	Title          string
	Error          string
	Success        string
	CSRFToken      string
	User           interface{}
	Schedules      interface{}
	DayNames       map[int]string
	GoalNames      map[string]string
	ActivityNames  map[string]string
	Form           map[string]string
}

// Server — HTTP-сервер приложения.
type Server struct {
	store        *database.Store
	sessions     *auth.SessionManager
	rateLimiter  *RateLimiter
	loginBlocker *LoginBlocker
	secretKey    []byte
	templates    map[string]*template.Template
	httpServer   *http.Server
}

// NewServer создаёт HTTP-сервер с настроенным роутингом.
func NewServer(store *database.Store, sessions *auth.SessionManager, secretKey []byte, port string) (*Server, error) {
	s := &Server{
		store:        store,
		sessions:     sessions,
		rateLimiter:  NewRateLimiter(),
		loginBlocker: NewLoginBlocker(),
		secretKey:    secretKey,
	}

	if err := s.loadTemplates(); err != nil {
		return nil, fmt.Errorf("ошибка загрузки шаблонов: %w", err)
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)

	handler := SecurityHeaders(RequestLogger(mux))

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
	authMW := auth.RequireAuth(s.sessions)

	// --- Страницы (SSR) ---
	mux.HandleFunc("GET /register", s.handleRegisterPage)
	mux.HandleFunc("POST /register", s.handleRegisterForm)
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLoginForm)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.Handle("GET /dashboard", authMW(http.HandlerFunc(s.handleDashboard)))
	mux.Handle("GET /profile/edit", authMW(http.HandlerFunc(s.handleProfileEditPage)))
	mux.Handle("POST /profile/edit", authMW(http.HandlerFunc(s.handleProfileEditForm)))
	mux.Handle("GET /schedules", authMW(http.HandlerFunc(s.handleSchedulesPage)))
	mux.Handle("POST /schedules/add", authMW(http.HandlerFunc(s.handleScheduleAdd)))
	mux.Handle("POST /schedules/{id}/delete", authMW(http.HandlerFunc(s.handleScheduleDelete)))
	mux.HandleFunc("GET /unsubscribe", s.handleUnsubscribePage)

	// --- API (JSON) ---
	mux.HandleFunc("POST /api/register", s.handleAPIRegister)
	mux.HandleFunc("POST /api/login", s.handleAPILogin)
	mux.Handle("GET /api/profile", authMW(http.HandlerFunc(s.handleAPIProfile)))
	mux.Handle("PUT /api/profile", authMW(http.HandlerFunc(s.handleAPIProfileUpdate)))
	mux.Handle("GET /api/schedules", authMW(http.HandlerFunc(s.handleAPISchedules)))
	mux.Handle("POST /api/schedules", authMW(http.HandlerFunc(s.handleAPIScheduleCreate)))
	mux.Handle("PUT /api/schedules/{id}", authMW(http.HandlerFunc(s.handleAPIScheduleUpdate)))
	mux.Handle("DELETE /api/schedules/{id}", authMW(http.HandlerFunc(s.handleAPIScheduleDelete)))
	mux.HandleFunc("POST /api/unsubscribe", s.handleAPIUnsubscribe)

	// --- Сервисные ---
	mux.HandleFunc("GET /health", s.handleHealth)

	// Корень → редирект
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})
}

// Start запускает HTTP-сервер (блокирующий вызов).
func (s *Server) Start() error {
	slog.Info("HTTP-сервер запущен", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown корректно останавливает сервер.
func (s *Server) Shutdown(ctx context.Context) error {
	s.rateLimiter.Stop()
	s.loginBlocker.Stop()
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) loadTemplates() error {
	layout, err := template.New("").Funcs(template.FuncMap{
		"dayName": func(d int) string { return DayNames[d] },
	}).ParseFS(templatesFS, "templates/layout.html")
	if err != nil {
		return fmt.Errorf("ошибка парсинга layout: %w", err)
	}

	pages := []string{
		"register", "login", "dashboard",
		"profile_edit", "schedules", "unsubscribe",
	}

	s.templates = make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		tmpl, err := layout.Clone()
		if err != nil {
			return fmt.Errorf("ошибка клонирования layout для %s: %w", page, err)
		}
		tmpl, err = tmpl.ParseFS(templatesFS, "templates/"+page+".html")
		if err != nil {
			return fmt.Errorf("ошибка парсинга шаблона %s: %w", page, err)
		}
		s.templates[page] = tmpl
	}

	return nil
}

func (s *Server) render(w http.ResponseWriter, page string, data PageData) {
	tmpl, ok := s.templates[page]
	if !ok {
		http.Error(w, "Шаблон не найден", http.StatusInternalServerError)
		return
	}

	data.DayNames = DayNames
	data.GoalNames = GoalNames
	data.ActivityNames = ActivityNames

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("ошибка рендеринга шаблона", "page", page, "error", err)
	}
}

// checkHoneypot проверяет honeypot-поле. Возвращает true если бот.
func checkHoneypot(r *http.Request) bool {
	return r.FormValue("website") != ""
}

// rateLimitCategory определяет категорию rate limit по пути.
func rateLimitCategory(path string) string {
	switch path {
	case "/register", "/api/register":
		return "register"
	case "/login", "/api/login":
		return "login"
	default:
		return "default"
	}
}
