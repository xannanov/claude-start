package email

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"time"

	"gopkg.in/gomail.v2"

	"daily-email-sender/internal/config"
	"daily-email-sender/internal/models"
)

//go:embed templates/email.html
var emailTemplateRaw string

var emailTemplate = template.Must(template.New("email").Parse(emailTemplateRaw))

// exerciseItem используется в шаблоне для отображения упражнения с подходами и повторениями.
type exerciseItem struct {
	Name string
	Sets string
	Reps string
}

// emailData — данные для рендеринга HTML-шаблона письма.
type emailData struct {
	Greeting           string
	FirstName          string
	LastName           string
	DayName            string
	Motivation         string
	IsFallback         bool
	WorkoutTitle       string
	WorkoutDescription string
	WorkoutDuration    string
	Exercises          []exerciseItem
	CalorieTarget      string
	ProteinGoal        string
	WaterIntake        string
	Fat                string
	Carbs              string
	Breakfast          string
	Lunch              string
	Dinner             string
	Snacks             []string
	SentAt             string
}

// Sender отправляет письма через SMTP.
type Sender struct {
	smtp      config.SMTPConfig
	emailFrom string
}

// NewSender создаёт отправщик писем.
func NewSender(smtp config.SMTPConfig, emailFrom string) *Sender {
	return &Sender{smtp: smtp, emailFrom: emailFrom}
}

// CheckConnection проверяет доступность SMTP-сервера и правильность учётных данных.
// Вызывать при старте планировщика.
func (s *Sender) CheckConnection() error {
	d := gomail.NewDialer(s.smtp.Host, s.smtp.Port, s.smtp.User, s.smtp.Password)
	d.SSL = true
	d.TLSConfig = &tls.Config{ServerName: s.smtp.Host}

	closer, err := d.Dial()
	if err != nil {
		return fmt.Errorf("ошибка SMTP-аутентификации (%s:%d): %w", s.smtp.Host, s.smtp.Port, err)
	}
	closer.Close()
	return nil
}

// Send генерирует HTML-письмо из msg и отправляет на адрес toEmail.
func (s *Sender) Send(toEmail string, msg models.PersonalizedMessage) error {
	body, err := renderTemplate(msg)
	if err != nil {
		return fmt.Errorf("ошибка рендеринга шаблона: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.emailFrom)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", msg.Subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.smtp.Host, s.smtp.Port, s.smtp.User, s.smtp.Password)
	d.SSL = true
	d.TLSConfig = &tls.Config{ServerName: s.smtp.Host}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("ошибка отправки письма: %w", err)
	}

	return nil
}

// renderTemplate рендерит HTML-письмо по данным сообщения.
func renderTemplate(msg models.PersonalizedMessage) (string, error) {
	data := buildEmailData(msg)
	var buf bytes.Buffer
	if err := emailTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func buildEmailData(msg models.PersonalizedMessage) emailData {
	exercises := zipExercises(msg.Workout.Exercises, msg.Workout.Sets, msg.Workout.Reps)

	return emailData{
		Greeting:           msg.Subject,
		FirstName:          msg.User.FirstName,
		LastName:           msg.User.LastName,
		DayName:            msg.DayOfWeek,
		Motivation:         msg.Motivation,
		IsFallback:         msg.IsFallback,
		WorkoutTitle:       msg.Workout.Title,
		WorkoutDescription: msg.Workout.Description,
		WorkoutDuration:    msg.Workout.Duration,
		Exercises:          exercises,
		CalorieTarget:      msg.Nutrition.CalorieTarget,
		ProteinGoal:        msg.Nutrition.ProteinGoal,
		WaterIntake:        msg.Nutrition.WaterIntake,
		Fat:                msg.Nutrition.Fat,
		Carbs:              msg.Nutrition.Carbs,
		Breakfast:          msg.Nutrition.Breakfast,
		Lunch:              msg.Nutrition.Lunch,
		Dinner:             msg.Nutrition.Dinner,
		Snacks:             msg.Nutrition.Snacks,
		SentAt:             time.Now().In(mustLoadMoscow()).Format("15:04:05"),
	}
}

// zipExercises объединяет параллельные слайсы упражнений, подходов и повторений.
func zipExercises(names, sets, reps []string) []exerciseItem {
	n := len(names)
	if len(sets) < n {
		n = len(sets)
	}
	if len(reps) < n {
		n = len(reps)
	}
	items := make([]exerciseItem, n)
	for i := 0; i < n; i++ {
		items[i] = exerciseItem{Name: names[i], Sets: sets[i], Reps: reps[i]}
	}
	return items
}

func mustLoadMoscow() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(fmt.Sprintf("не удалось загрузить Europe/Moscow: %v", err))
	}
	return loc
}

// GeneratePersonalizedMessage генерирует персонализированное письмо на основе данных пользователя.
func GeneratePersonalizedMessage(user models.User, dayOfWeek int, emailType string) models.PersonalizedMessage {
	var greeting string
	switch emailType {
	case "morning":
		greeting = "Доброе утро! 🌅"
	case "afternoon":
		greeting = "Добрый день! ☀️"
	default:
		greeting = "Добрый вечер! 🌙"
	}

	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	dayName := days[dayOfWeek%7]

	workout := generateWorkoutPlan(user, emailType)
	// Тренировочный день — повышенная норма воды (35 мл/кг вместо 30)
	nutrition := generateNutritionPlan(user, true)

	return models.PersonalizedMessage{
		Subject:   greeting,
		Workout:   workout,
		Nutrition: nutrition,
		User:      user,
		DayOfWeek: dayName,
		TimeOfDay: emailType,
	}
}

func generateWorkoutPlan(user models.User, emailType string) models.WorkoutPlan {
	switch user.Goal {
	case "weight_loss":
		return generateWeightLossWorkout()
	case "muscle_gain":
		return generateMuscleGainWorkout()
	case "maintenance":
		return generateMaintenanceWorkout()
	default:
		return generateGeneralFitnessWorkout()
	}
}

func generateWeightLossWorkout() models.WorkoutPlan {
	return models.WorkoutPlan{
		Title:       "Кардио + силовые для потери веса",
		Description: "Акцент на кардио для сжигания калорий и базовые упражнения для сохранения мышц",
		Duration:    "30-40 минут",
		Exercises: []string{
			"Приседания с весом",
			"Отжимания",
			"Планка",
			"Бёрпи",
			"Велосипед/бег",
		},
		Sets: []string{"3 подхода", "3 подхода", "3 подхода", "2 подхода", "1 подход"},
		Reps: []string{"12-15 раз", "10-15 раз", "30-60 сек", "10 раз", "20-30 мин"},
	}
}

func generateMuscleGainWorkout() models.WorkoutPlan {
	return models.WorkoutPlan{
		Title:       "Тренировка с весом для набора мышц",
		Description: "Многосуставные упражнения для максимального роста мышц",
		Duration:    "45-60 минут",
		Exercises: []string{
			"Жим лёжа",
			"Становая тяга",
			"Приседания со штангой",
			"Тяга горизонтального блока",
			"Подтягивания",
		},
		Sets: []string{"4 подхода", "3 подхода", "4 подхода", "3 подхода", "3 подхода"},
		Reps: []string{"8-10 раз", "5-8 раз", "8-12 раз", "10-12 раз", "до отказа"},
	}
}

func generateMaintenanceWorkout() models.WorkoutPlan {
	return models.WorkoutPlan{
		Title:       "Балансирующая нагрузка",
		Description: "Хорошая нагрузка для поддержания формы",
		Duration:    "30-45 минут",
		Exercises: []string{
			"Комбинированные упражнения (присед+жим)",
			"Горизонтальная тяга",
			"Вертикальная тяга",
			"Кардио (бег/велосипед)",
			"Растяжка",
		},
		Sets: []string{"3 подхода", "3 подхода", "3 подхода", "1 подход", "1 подход"},
		Reps: []string{"10-12 раз", "10-12 раз", "10-12 раз", "15-20 мин", "10-15 мин"},
	}
}

func generateGeneralFitnessWorkout() models.WorkoutPlan {
	return models.WorkoutPlan{
		Title:       "Фитнес-тренировка для общего здоровья",
		Description: "Универсальная тренировка для всех уровней подготовки",
		Duration:    "30-40 минут",
		Exercises: []string{
			"Упражнения на мобильность и гибкость",
			"Комплексные упражнения для основных групп мышц",
			"Кардио-интервалы",
			"Силовые упражнения с весом тела",
			"Заминка",
		},
		Sets: []string{"3-4 серии", "3-4 серии", "3-4 серии", "3-4 серии", "1 подход"},
		Reps: []string{"10-15 раз", "10-15 раз", "3-4 мин", "10-15 раз", "10 мин"},
	}
}

func generateNutritionPlan(user models.User, isTrainingDay bool) models.NutritionPlan {
	var calorieTarget int
	var proteinPerKg float64

	switch user.ActivityLevel {
	case "sedentary":
		calorieTarget, proteinPerKg = 1800, 1.2
	case "light":
		calorieTarget, proteinPerKg = 2000, 1.3
	case "moderate":
		calorieTarget, proteinPerKg = 2200, 1.4
	case "active":
		calorieTarget, proteinPerKg = 2400, 1.6
	case "very_active":
		calorieTarget, proteinPerKg = 2600, 1.8
	default:
		calorieTarget, proteinPerKg = 2000, 1.4
	}

	if user.Age > 30 && user.Age < 50 {
		calorieTarget -= 200
	} else if user.Age >= 50 {
		calorieTarget -= 300
	}

	if user.Gender == "male" {
		calorieTarget += 300
		proteinPerKg += 0.5
	}

	proteinTarget := math.Round(proteinPerKg*user.WeightKg*10) / 10
	waterPerKg := 30.0
	if isTrainingDay {
		waterPerKg = 35.0
	}
	waterMl := int(user.WeightKg * waterPerKg)

	nutrition := models.NutritionPlan{
		ProteinGoal:   fmt.Sprintf("%.0f г (%.0f ккал)", proteinTarget, math.Round(proteinTarget*4)),
		CalorieTarget: fmt.Sprintf("%d ккал/день", calorieTarget),
		WaterIntake:   fmt.Sprintf("%d мл/день", waterMl),
	}

	switch user.Goal {
	case "weight_loss":
		nutrition.Breakfast = "Каша с ягодами + яйцо + кефир (250 мл)"
		nutrition.Lunch = "Куриная грудка с овощами + гречка (150 г)"
		nutrition.Dinner = "Белая рыба с салатом из огурцов и зелени"
		nutrition.Snacks = []string{"Яблоко", "Орехи (30 г)", "Творог 2% (150 г)"}
	case "muscle_gain":
		nutrition.Breakfast = "Яичница (2 яйца) + тост + банан + протеиновый коктейль"
		nutrition.Lunch = "Говядина + макароны + овощной салат"
		nutrition.Dinner = "Куриное филе + рис + овощи + творог"
		nutrition.Snacks = []string{"Протеиновый батончик", "Творог с мёдом", "Греческий йогурт"}
	default:
		nutrition.Breakfast = "Тост с авокадо + яйцо + кофе"
		nutrition.Lunch = "Салат с тунцом + макароны с курицей"
		nutrition.Dinner = "Рыба/мясо + овощи + гарнир"
		nutrition.Snacks = []string{"Фрукты", "Орехи", "Йогурт"}
	}

	return nutrition
}
