package main

import (
	"fmt"
	"math"
	"time"
)

// GeneratePersonalizedMessage generates a personalized email based on user data
func GeneratePersonalizedMessage(user User, dayOfWeek int, emailType string) PersonalizedMessage {
	var timeOfDay string
	if emailType == "morning" {
		timeOfDay = "Доброе утро! 🌅"
	} else if emailType == "afternoon" {
		timeOfDay = "Добрый день! ☀️"
	} else {
		timeOfDay = "Добрый вечер! 🌙"
	}

	// Get day of week name
	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	dayName := days[dayOfWeek%7]

	// Generate personalized workout plan
	workout := GenerateWorkoutPlan(user, emailType)

	// Generate personalized nutrition plan
	nutrition := GenerateNutritionPlan(user)

	// Generate email body
	body := generateEmailHTML(user, dayName, timeOfDay, workout, nutrition)

	return PersonalizedMessage{
		Subject:       timeOfDay,
		Body:          body,
		Workout:       workout,
		Nutrition:     nutrition,
		User:          user,
		DayOfWeek:     dayName,
		TimeOfDay:     emailType,
	}
}

// GenerateWorkoutPlan generates workout based on user goals and activity level
func GenerateWorkoutPlan(user User, emailType string) WorkoutPlan {
	workout := WorkoutPlan{}

	switch user.Goal {
	case "weight_loss":
		workout = generateWeightLossWorkout(user, emailType)
	case "muscle_gain":
		workout = generateMuscleGainWorkout(user, emailType)
	case "maintenance":
		workout = generateMaintenanceWorkout(user, emailType)
	default:
		workout = generateGeneralFitnessWorkout(user, emailType)
	}

	return workout
}

// GenerateNutritionPlan generates nutrition based on user parameters
func GenerateNutritionPlan(user User) NutritionPlan {
	var calorieTarget int
	var proteinPerKg float64

	// Calculate calorie target based on goal and activity
	switch user.ActivityLevel {
	case "sedentary":
		calorieTarget = 1800
		proteinPerKg = 1.2
	case "light":
		calorieTarget = 2000
		proteinPerKg = 1.3
	case "moderate":
		calorieTarget = 2200
		proteinPerKg = 1.4
	case "active":
		calorieTarget = 2400
		proteinPerKg = 1.6
	case "very_active":
		calorieTarget = 2600
		proteinPerKg = 1.8
	default:
		calorieTarget = 2000
		proteinPerKg = 1.4
	}

	// Adjust for age and gender
	if user.Age > 30 && user.Age < 50 {
		calorieTarget -= 200
	}
	if user.Age >= 50 {
		calorieTarget -= 300
	}

	if user.Gender == "male" {
		calorieTarget += 300
		proteinPerKg += 0.5
	}

	// Calculate protein target based on weight
	proteinTarget := math.Round(proteinPerKg * user.WeightKg * 10) / 10

	// Generate meal plans
	nutrition := NutritionPlan{
		ProteinGoal:   fmt.Sprintf("%.0f грамм (%.0f ккал)", proteinTarget, math.Round(proteinTarget*4)),
		CalorieTarget: fmt.Sprintf("%d ккал/день", calorieTarget),
		WaterIntake:   fmt.Sprintf("%d-2000 мл/день", calorieTarget/5),
	}

	switch user.Goal {
	case "weight_loss":
		nutrition.Breakfast = "Каша с ягодами + яйцо + кефир (250мл)"
		nutrition.Lunch = "Куриная грудка с овощами + гречка (150г)"
		nutrition.Dinner = "Белая рыба с салатом из огурцов и зелени"
		nutrition.Snacks = []string{
			"Яблоко",
			"Орехи (30г)",
			"Творог 2% (150г)",
		}
	case "muscle_gain":
		nutrition.Breakfast = "Яичница (2 яйца) + тост + банан + протеиновый коктейль"
		nutrition.Lunch = "Говядина + макароны + овощной салат + гречка"
		nutrition.Dinner = "Куриное филе + рис + овощи + творог"
		nutrition.Snacks = []string{
			"Протеиновый батончик",
			"Творог с медом",
			"Греческий йогурт",
		}
	default:
		nutrition.Breakfast = "Тост с авокадо + яйцо + кофе"
		nutrition.Lunch = "Салат с тунцом + макароны с курицей"
		nutrition.Dinner = "Рыба/мясо + овощи + гарнир"
		nutrition.Snacks = []string{
			"Фрукты",
			"Орехи",
			"Йогурт",
		}
	}

	return nutrition
}

func generateWeightLossWorkout(user User, emailType string) WorkoutPlan {
	return WorkoutPlan{
		Title: "Кардио + силовые для потери веса",
		Exercises: []string{
			"Приседания с весом (3 подхода по 12-15 раз)",
			"Отжимания (3 подхода по 10-15 раз)",
			"Планка (3 подхода по 30-60 сек)",
			"Бёрпи (2 подхода по 10 раз)",
			"Велосипед/бег (20-30 мин)",
		},
		Duration: "30-40 минут",
		Description: "Акцент на кардио для сжигания калорий и базовые упражнения для сохранения мышц",
		Sets: []string{"3 подхода", "3 подхода", "3 подхода", "2-3 подхода", "25-30 мин"},
		Reps: []string{"12-15 раз", "10-15 раз", "30-60 сек", "8-12 раз", "20-30 мин"},
	}
}

func generateMuscleGainWorkout(user User, emailType string) WorkoutPlan {
	return WorkoutPlan{
		Title: "Тренировка с весом для набора мышц",
		Exercises: []string{
			"Жим лежа (4 подхода по 8-10 раз)",
			"Становая тяга (3 подхода по 5-8 раз)",
			"Приседания со штангой (4 подхода по 8-12 раз)",
			"Тяга горизонтального блока (3 подхода по 10-12 раз)",
			"Подтягивания/машине (3 подхода до отказа)",
		},
		Duration: "45-60 минут",
		Description: "Многосуставные упражнения для максимального роста мышц",
		Sets: []string{"4 подхода", "3 подхода", "4 подхода", "3 подхода", "3 подхода"},
		Reps: []string{"8-10 раз", "5-8 раз", "8-12 раз", "10-12 раз", "до отказа"},
	}
}

func generateMaintenanceWorkout(user User, emailType string) WorkoutPlan {
	return WorkoutPlan{
		Title: "Балансирование нагрузка",
		Exercises: []string{
			"Комбинированные упражнения (присед+жим) - 3 подхода по 10-12 раз",
			"Горизонтальная тяга - 3 подхода по 10-12 раз",
			"Вертикальная тяга - 3 подхода по 10-12 раз",
			"Кардио (бег/велосипед) - 15-20 минут",
			"Растяжка - 10-15 минут",
		},
		Duration: "30-45 минут",
		Description: "Хорошая нагрузка для поддержания формы",
		Sets: []string{"3 подхода", "3 подхода", "3 подхода", "15-20 мин", "10-15 мин"},
		Reps: []string{"10-12 раз", "10-12 раз", "10-12 раз", "15-20 мин", "10-15 мин"},
	}
}

func generateGeneralFitnessWorkout(user User, emailType string) WorkoutPlan {
	return WorkoutPlan{
		Title: "Фитнес-тренировка для общего здоровья",
		Exercises: []string{
			"Упражнения на мобильность и гибкость",
			"Комплексные упражнения для основных групп мышц",
			"Кардио-интервалы",
			"Силовые упражнения с весом тела",
			"Заминка",
		},
		Duration: "30-40 минут",
		Description: "Универсальная тренировка для всех уровней подготовки",
		Sets: []string{"3-4 серии", "3-4 серии", "3-4 серии", "3-4 серии", "10 минут"},
		Reps: []string{"10-15 повторений", "10-15 повторений", "3-4 минуты", "10-15 повторений", "10 минут"},
	}
}

func generateEmailHTML(user User, dayName, timeOfDay string, workout WorkoutPlan, nutrition NutritionPlan) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
		</head>
		<body style="font-family: Arial, sans-serif; background-color: #f0f0f0; padding: 20px;">
			<div style="background-color: #fff; border-radius: 10px; padding: 30px; max-width: 600px; margin: 0 auto; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
				<h1 style="color: #4CAF50;">%s</h1>
				<p>Приветствую, %s %s!</p>
				<p><strong>Сегодня %s</strong></p>

				<h3 style="color: #2196F3;">🔥 Тренировка на сегодня:</h3>
				<h4 style="color: #1976D2;">%s</h4>
				<p><em>%s</em></p>
				<ul style="list-style-type: circle; margin-left: 20px;">
					%s</ul>
				<p><strong>Длительность:</strong> %s</p>
				<p><strong>Сеты/Повторения:</strong> %s / %s</p>

				<h3 style="color: #FF9800; margin-top: 30px;">🍎 Питание на сегодня:</h3>
				<p><strong>Твоя цель:</strong> %s ккал/день</p>
				<p><strong>Белок:</strong> %s</p>
				<p><strong>Вода:</strong> %s</p>

				<h4>Завтрак:</h4>
				<p style="margin-left: 20px;">%s</p>

				<h4>Обед:</h4>
				<p style="margin-left: 20px;">%s</p>

				<h4>Ужин:</h4>
				<p style="margin-left: 20px;">%s</p>

				<h4>Перекусы:</h4>
				<ul style="list-style-type: circle; margin-left: 20px;">
					%s</ul>

				<p style="margin-top: 30px; color: #888; font-size: 12px;">Время: %s</p>
			</div>
		</body>
		</html>
	`,
		timeOfDay,
		user.FirstName,
		user.LastName,
		dayName,
		workout.Title,
		workout.Description,
		generateExerciseList(workout.Exercises),
		workout.Duration,
		generateSetsReps(workout.Sets, workout.Reps),
		nutrition.CalorieTarget,
		nutrition.ProteinGoal,
		nutrition.WaterIntake,
		nutrition.Breakfast,
		nutrition.Lunch,
		nutrition.Dinner,
		generateSnackList(nutrition.Snacks),
		time.Now().Format("15:04:05"),
	)
}

func generateExerciseList(exercises []string) string {
	var result string
	for _, ex := range exercises {
		result += fmt.Sprintf("<li>%s</li>", ex)
	}
	return result
}

func generateSetsReps(sets, reps []string) string {
	var result string
	for i := 0; i < len(sets); i++ {
		if i > 0 {
			result += "<br>"
		}
		result += fmt.Sprintf("%s / %s", sets[i], reps[i])
	}
	return result
}

func generateSnackList(snacks []string) string {
	var result string
	for _, snack := range snacks {
		result += fmt.Sprintf("<li>%s</li>", snack)
	}
	return result
}
