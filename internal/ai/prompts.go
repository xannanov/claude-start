package ai

import (
	"fmt"
	"strings"

	"daily-email-sender/internal/models"
)

// buildWorkoutPrompt строит промпт для генерации тренировки.
func buildWorkoutPrompt(user models.User, dayOfWeek, timeOfDay string, history []models.WorkoutHistory) []ChatMessage {
	system := `Ты — профессиональный фитнес-тренер. Составь тренировку на сегодня.
Ответ СТРОГО в формате JSON (без markdown-обёрток):
{
  "title": "название тренировки",
  "muscle_group": "основная группа мышц (одно из: legs, chest, back, shoulders, arms, core, cardio)",
  "duration": "длительность (например: 30-40 минут)",
  "description": "краткое описание тренировки (1-2 предложения)",
  "exercises": [
    {"name": "название упражнения", "sets": "N подходов", "reps": "N раз или N сек"}
  ]
}

Правила:
- Ровно 4-6 упражнений
- Утром (morning) — бодрящие, динамичные упражнения
- Днём (afternoon) — стандартная нагрузка
- Вечером (evening) — спокойнее, с акцентом на растяжку в конце
- НЕ повторяй группу мышц, которая была в последние 1-2 дня
- Разнообразие: не повторяй одинаковые упражнения из раза в раз
- Все тексты на русском языке`

	var historyStr string
	if len(history) > 0 {
		var parts []string
		for _, h := range history {
			parts = append(parts, fmt.Sprintf("%s — %s", h.Date.Format("02.01"), h.MuscleGroup))
		}
		historyStr = fmt.Sprintf("\nПоследние тренировки: %s", strings.Join(parts, ", "))
	} else {
		historyStr = "\nИстория тренировок пуста (первая тренировка)."
	}

	user_msg := fmt.Sprintf(`Данные пользователя:
- Пол: %s
- Возраст: %d лет
- Рост: %d см
- Вес: %.1f кг
- Цель: %s
- Уровень активности: %s
- День недели: %s
- Время суток: %s%s`,
		genderRu(user.Gender), user.Age, user.HeightCm, user.WeightKg,
		goalRu(user.Goal), activityRu(user.ActivityLevel),
		dayOfWeek, timeOfDay, historyStr)

	return []ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user_msg},
	}
}

// buildNutritionPrompt строит промпт для генерации плана питания.
func buildNutritionPrompt(user models.User) []ChatMessage {
	system := `Ты — диетолог. Составь план питания на день.
Ответ СТРОГО в формате JSON (без markdown-обёрток):
{
  "breakfast": "конкретное блюдо с граммовками",
  "lunch": "конкретное блюдо с граммовками",
  "dinner": "конкретное блюдо с граммовками",
  "snacks": ["перекус 1 с граммовкой", "перекус 2 с граммовкой"],
  "calories": "N ккал",
  "protein": "N г",
  "fat": "N г",
  "carbs": "N г",
  "water_ml": "N мл"
}

Правила:
- Продукты, доступные в России (реалистичные блюда)
- Конкретные граммовки (не "порция", а "150 г")
- Расчёт БЖУ и калорий на основе данных пользователя
- Разнообразие блюд (не повторяй одно и то же каждый день)
- 2-3 перекуса
- Все тексты на русском языке`

	user_msg := fmt.Sprintf(`Данные пользователя:
- Пол: %s
- Возраст: %d лет
- Рост: %d см
- Вес: %.1f кг
- Цель: %s
- Уровень активности: %s`,
		genderRu(user.Gender), user.Age, user.HeightCm, user.WeightKg,
		goalRu(user.Goal), activityRu(user.ActivityLevel))

	return []ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user_msg},
	}
}

// buildMotivationPrompt строит промпт для генерации мотивационного сообщения.
func buildMotivationPrompt(user models.User, dayOfWeek string) []ChatMessage {
	system := `Ты — лучший друг, который тащит в зал. Напиши мотивационное сообщение.
Ответ СТРОГО в формате JSON (без markdown-обёрток):
{"text": "мотивационное сообщение"}

Правила:
- 2-3 предложения
- Юмор, подколки, дружеский тон
- Используй эмодзи
- Обращайся по имени
- Каждое сообщение уникальное
- На русском языке`

	user_msg := fmt.Sprintf(`Имя: %s
День недели: %s
Цель: %s`, user.FirstName, dayOfWeek, goalRu(user.Goal))

	return []ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user_msg},
	}
}

// genderRu переводит пол на русский.
func genderRu(gender string) string {
	switch gender {
	case "male":
		return "мужской"
	case "female":
		return "женский"
	default:
		return "другой"
	}
}

// goalRu переводит цель на русский.
func goalRu(goal string) string {
	switch goal {
	case "weight_loss":
		return "похудение"
	case "muscle_gain":
		return "набор мышечной массы"
	case "maintenance":
		return "поддержание формы"
	case "general_fitness":
		return "общая физическая форма"
	default:
		return goal
	}
}

// activityRu переводит уровень активности на русский.
func activityRu(level string) string {
	switch level {
	case "sedentary":
		return "сидячий образ жизни"
	case "light":
		return "лёгкая активность"
	case "moderate":
		return "умеренная активность"
	case "active":
		return "активный образ жизни"
	case "very_active":
		return "очень активный"
	default:
		return level
	}
}
