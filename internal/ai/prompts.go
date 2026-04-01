package ai

import (
	"fmt"
	"strings"

	"daily-email-sender/internal/models"
)

// buildCombinedPrompt строит единый промпт для генерации тренировки, питания и мотивации.
func buildCombinedPrompt(user models.User, dayOfWeek, timeOfDay string, history []models.WorkoutHistory) []ChatMessage {
	system := `Ты — персональный фитнес-тренер и диетолог. Создай полный план на сегодня.

ВЕРНИ ТОЛЬКО JSON — без markdown-блоков, без пояснений, без лишнего текста:
{
  "workout": {
    "title": "название тренировки",
    "muscle_group": "одно из: legs, chest, back, shoulders, arms, core, cardio",
    "duration": "например: 40-50 минут",
    "description": "1-2 предложения о тренировке",
    "exercises": [
      {"name": "Жим лёжа", "sets": "4 подхода", "reps": "8-10 раз"}
    ]
  },
  "nutrition": {
    "breakfast": "блюдо с граммовками (например: Овсянка 150г + яйцо 2шт + кофе)",
    "lunch": "блюдо с граммовками",
    "dinner": "блюдо с граммовками",
    "snacks": ["перекус с граммовкой", "перекус с граммовкой"],
    "calories": "2400 ккал",
    "protein": "140 г",
    "fat": "70 г",
    "carbs": "280 г",
    "water_ml": "2800 мл"
  },
  "motivation": {
    "text": "мотивационное сообщение 2-3 предложения с юмором и эмодзи"
  }
}

Требования к тренировке:
- Ровно 4-6 упражнений
- Утром (morning) — бодрящие, динамичные; вечером (evening) — спокойнее, растяжка в конце
- НЕ повторяй muscle_group из истории последних 1-2 дней
- Если история тренировок есть — назначай тренировку на КОНКРЕТНУЮ группу мышц (не fullbody). В title и description укажи какие мышцы качаем сегодня
- Учитывай совместимость мышечных групп в одной тренировке:
  ХОРОШИЕ сочетания: грудь+трицепс, спина+бицепс, плечи+трапеция, бицепс+трицепс (руки), ноги отдельно
  ПЛОХИЕ сочетания (НЕ совмещать): грудь+бицепс, спина+трицепс (антагонисты забиваются при базовых упражнениях)
- Если история пуста (первая тренировка) — можно fullbody
- Разнообразие упражнений каждый день

Требования к питанию:
- Российские продукты, конкретные блюда с граммовками
- БЖУ и ккал соответствуют цели и параметрам пользователя
- 2-3 перекуса

Требования к мотивации:
- Обращение по имени, дружеский юмор, подколки, эмодзи`

	var historyStr string
	if len(history) > 0 {
		var parts []string
		for _, h := range history {
			parts = append(parts, fmt.Sprintf("%s — %s", h.Date.Format("02.01"), h.MuscleGroup))
		}
		historyStr = fmt.Sprintf("\nПоследние тренировки (не повторяй группу мышц): %s", strings.Join(parts, ", "))
	} else {
		historyStr = "\nИстория тренировок пуста — первая тренировка, выбери любую группу мышц."
	}

	userMsg := fmt.Sprintf(`Данные пользователя:
- Имя: %s
- Пол: %s
- Возраст: %d лет
- Рост: %d см, Вес: %.1f кг
- Цель: %s
- Уровень активности: %s
- День недели: %s
- Время суток: %s%s`,
		user.FirstName,
		genderRu(user.Gender), user.Age,
		user.HeightCm, user.WeightKg,
		goalRu(user.Goal), activityRu(user.ActivityLevel),
		dayOfWeek, timeOfDay, historyStr)

	return []ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: userMsg},
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
