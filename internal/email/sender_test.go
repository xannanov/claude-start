package email

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"daily-email-sender/internal/models"
)

// testUser возвращает типичного пользователя для тестов.
func testUser(goal, activityLevel, gender string, age int, weightKg float64) models.User {
	return models.User{
		ID:            "test-uuid",
		Email:         "test@example.com",
		FirstName:     "Иван",
		LastName:      "Иванов",
		Age:           age,
		Gender:        gender,
		HeightCm:      180,
		WeightKg:      weightKg,
		Goal:          goal,
		ActivityLevel: activityLevel,
	}
}

// --- zipExercises ---

func TestZipExercises(t *testing.T) {
	t.Run("equal length slices", func(t *testing.T) {
		names := []string{"Приседания", "Отжимания"}
		sets := []string{"3 подхода", "4 подхода"}
		reps := []string{"10 раз", "12 раз"}

		result := zipExercises(names, sets, reps)
		if len(result) != 2 {
			t.Fatalf("expected 2 exercises, got %d", len(result))
		}
		if result[0].Name != "Приседания" || result[0].Sets != "3 подхода" || result[0].Reps != "10 раз" {
			t.Errorf("unexpected first exercise: %+v", result[0])
		}
		if result[1].Name != "Отжимания" || result[1].Sets != "4 подхода" || result[1].Reps != "12 раз" {
			t.Errorf("unexpected second exercise: %+v", result[1])
		}
	})

	t.Run("names shorter", func(t *testing.T) {
		names := []string{"A"}
		sets := []string{"1", "2", "3"}
		reps := []string{"10", "20", "30"}
		result := zipExercises(names, sets, reps)
		if len(result) != 1 {
			t.Fatalf("expected 1 exercise, got %d", len(result))
		}
	})

	t.Run("sets shorter", func(t *testing.T) {
		names := []string{"A", "B", "C"}
		sets := []string{"1"}
		reps := []string{"10", "20", "30"}
		result := zipExercises(names, sets, reps)
		if len(result) != 1 {
			t.Fatalf("expected 1 exercise, got %d", len(result))
		}
	})

	t.Run("reps shorter", func(t *testing.T) {
		names := []string{"A", "B", "C"}
		sets := []string{"1", "2", "3"}
		reps := []string{"10", "20"}
		result := zipExercises(names, sets, reps)
		if len(result) != 2 {
			t.Fatalf("expected 2 exercises, got %d", len(result))
		}
	})

	t.Run("empty slices", func(t *testing.T) {
		result := zipExercises(nil, nil, nil)
		if len(result) != 0 {
			t.Fatalf("expected 0 exercises, got %d", len(result))
		}
	})
}

// --- generateWorkoutPlan ---

func TestGenerateWorkoutPlan(t *testing.T) {
	goals := []struct {
		goal          string
		expectedTitle string
	}{
		{"weight_loss", "Кардио + силовые для потери веса"},
		{"muscle_gain", "Тренировка с весом для набора мышц"},
		{"maintenance", "Балансирующая нагрузка"},
		{"unknown_goal", "Фитнес-тренировка для общего здоровья"},
		{"", "Фитнес-тренировка для общего здоровья"},
	}

	for _, tt := range goals {
		t.Run(tt.goal, func(t *testing.T) {
			user := testUser(tt.goal, "moderate", "male", 25, 80)
			plan := generateWorkoutPlan(user, "morning")

			if plan.Title != tt.expectedTitle {
				t.Errorf("goal=%q: expected title %q, got %q", tt.goal, tt.expectedTitle, plan.Title)
			}
			if len(plan.Exercises) == 0 {
				t.Error("expected exercises, got empty")
			}
			if len(plan.Exercises) != len(plan.Sets) {
				t.Errorf("exercises/sets length mismatch: %d vs %d", len(plan.Exercises), len(plan.Sets))
			}
			if len(plan.Exercises) != len(plan.Reps) {
				t.Errorf("exercises/reps length mismatch: %d vs %d", len(plan.Exercises), len(plan.Reps))
			}
			if plan.Duration == "" {
				t.Error("expected duration, got empty")
			}
		})
	}
}

// --- generateNutritionPlan ---

func TestGenerateNutritionPlan_WaterFormula(t *testing.T) {
	tests := []struct {
		name           string
		weightKg       float64
		isTrainingDay  bool
		expectedWaterMl int
	}{
		{"80kg training day", 80.0, true, 2800},   // 80 * 35
		{"80kg rest day", 80.0, false, 2400},       // 80 * 30
		{"60kg training day", 60.0, true, 2100},    // 60 * 35
		{"60kg rest day", 60.0, false, 1800},       // 60 * 30
		{"100kg training day", 100.0, true, 3500},  // 100 * 35
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := testUser("maintenance", "moderate", "female", 25, tt.weightKg)
			plan := generateNutritionPlan(user, tt.isTrainingDay)

			expected := fmt.Sprintf("%d мл/день", tt.expectedWaterMl)
			if plan.WaterIntake != expected {
				t.Errorf("WaterIntake=%q, expected %q", plan.WaterIntake, expected)
			}
		})
	}
}

func TestGenerateNutritionPlan_CalorieAdjustments(t *testing.T) {
	// Базовый moderate = 2200, male +300 = 2500
	youngMale := testUser("maintenance", "moderate", "male", 25, 80)
	planYoung := generateNutritionPlan(youngMale, false)

	// 31-49: -200 → 2300
	midMale := testUser("maintenance", "moderate", "male", 35, 80)
	planMid := generateNutritionPlan(midMale, false)

	// 50+: -300 → 2200
	oldMale := testUser("maintenance", "moderate", "male", 55, 80)
	planOld := generateNutritionPlan(oldMale, false)

	if !strings.Contains(planYoung.CalorieTarget, "2500") {
		t.Errorf("young male moderate: expected 2500, got %s", planYoung.CalorieTarget)
	}
	if !strings.Contains(planMid.CalorieTarget, "2300") {
		t.Errorf("mid male moderate: expected 2300, got %s", planMid.CalorieTarget)
	}
	if !strings.Contains(planOld.CalorieTarget, "2200") {
		t.Errorf("old male moderate: expected 2200, got %s", planOld.CalorieTarget)
	}

	// Female sedentary, age 25: 1800, no male bonus = 1800
	youngFemale := testUser("maintenance", "sedentary", "female", 25, 60)
	planFemale := generateNutritionPlan(youngFemale, false)
	if !strings.Contains(planFemale.CalorieTarget, "1800") {
		t.Errorf("young female sedentary: expected 1800, got %s", planFemale.CalorieTarget)
	}
}

func TestGenerateNutritionPlan_ProteinCalculation(t *testing.T) {
	// moderate female 25y, 60kg: proteinPerKg=1.4, protein=1.4*60=84
	user := testUser("maintenance", "moderate", "female", 25, 60)
	plan := generateNutritionPlan(user, false)

	expectedProtein := math.Round(1.4*60*10) / 10 // 84.0
	if !strings.Contains(plan.ProteinGoal, "84") {
		t.Errorf("expected protein ~84g, got %s", plan.ProteinGoal)
	}
	_ = expectedProtein

	// male adds +0.5: moderate male 25y, 80kg → proteinPerKg=1.9, protein=1.9*80=152
	userMale := testUser("maintenance", "moderate", "male", 25, 80)
	planMale := generateNutritionPlan(userMale, false)
	expectedProteinMale := math.Round(1.9*80*10) / 10 // 152.0
	if !strings.Contains(planMale.ProteinGoal, "152") {
		t.Errorf("expected protein ~152g, got %s", planMale.ProteinGoal)
	}
	_ = expectedProteinMale
}

func TestGenerateNutritionPlan_ActivityLevels(t *testing.T) {
	levels := []struct {
		level    string
		baseCal  int
	}{
		{"sedentary", 1800},
		{"light", 2000},
		{"moderate", 2200},
		{"active", 2400},
		{"very_active", 2600},
		{"unknown", 2000},
	}

	for _, tt := range levels {
		t.Run(tt.level, func(t *testing.T) {
			// Female, age 25 — без корректировок пола и возраста
			user := testUser("maintenance", tt.level, "female", 25, 70)
			plan := generateNutritionPlan(user, false)
			expected := fmt.Sprintf("%d", tt.baseCal)
			if !strings.Contains(plan.CalorieTarget, expected) {
				t.Errorf("level=%s: expected %s cal, got %s", tt.level, expected, plan.CalorieTarget)
			}
		})
	}
}

func TestGenerateNutritionPlan_MealsByGoal(t *testing.T) {
	tests := []struct {
		goal             string
		breakfastContains string
	}{
		{"weight_loss", "Каша с ягодами"},
		{"muscle_gain", "Яичница"},
		{"maintenance", "Тост с авокадо"},
		{"anything_else", "Тост с авокадо"},
	}

	for _, tt := range tests {
		t.Run(tt.goal, func(t *testing.T) {
			user := testUser(tt.goal, "moderate", "male", 25, 80)
			plan := generateNutritionPlan(user, false)
			if !strings.Contains(plan.Breakfast, tt.breakfastContains) {
				t.Errorf("goal=%s: breakfast=%q does not contain %q", tt.goal, plan.Breakfast, tt.breakfastContains)
			}
			if plan.Lunch == "" || plan.Dinner == "" {
				t.Error("lunch or dinner is empty")
			}
			if len(plan.Snacks) == 0 {
				t.Error("snacks is empty")
			}
		})
	}
}

// --- GeneratePersonalizedMessage ---

func TestGeneratePersonalizedMessage(t *testing.T) {
	user := testUser("weight_loss", "moderate", "male", 25, 80)

	t.Run("morning greeting", func(t *testing.T) {
		msg := GeneratePersonalizedMessage(user, 0, "morning")
		if !strings.Contains(msg.Subject, "утро") {
			t.Errorf("morning subject=%q does not contain 'утро'", msg.Subject)
		}
		if msg.DayOfWeek != "Понедельник" {
			t.Errorf("day 0 expected Понедельник, got %s", msg.DayOfWeek)
		}
	})

	t.Run("afternoon greeting", func(t *testing.T) {
		msg := GeneratePersonalizedMessage(user, 2, "afternoon")
		if !strings.Contains(msg.Subject, "день") {
			t.Errorf("afternoon subject=%q does not contain 'день'", msg.Subject)
		}
		if msg.DayOfWeek != "Среда" {
			t.Errorf("day 2 expected Среда, got %s", msg.DayOfWeek)
		}
	})

	t.Run("evening greeting", func(t *testing.T) {
		msg := GeneratePersonalizedMessage(user, 4, "evening")
		if !strings.Contains(msg.Subject, "вечер") {
			t.Errorf("evening subject=%q does not contain 'вечер'", msg.Subject)
		}
		if msg.DayOfWeek != "Пятница" {
			t.Errorf("day 4 expected Пятница, got %s", msg.DayOfWeek)
		}
	})

	t.Run("unknown email type defaults to evening", func(t *testing.T) {
		msg := GeneratePersonalizedMessage(user, 6, "unknown")
		if !strings.Contains(msg.Subject, "вечер") {
			t.Errorf("unknown type subject=%q does not contain 'вечер'", msg.Subject)
		}
		if msg.DayOfWeek != "Воскресенье" {
			t.Errorf("day 6 expected Воскресенье, got %s", msg.DayOfWeek)
		}
	})

	t.Run("all days of week", func(t *testing.T) {
		expectedDays := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
		for i, expected := range expectedDays {
			msg := GeneratePersonalizedMessage(user, i, "morning")
			if msg.DayOfWeek != expected {
				t.Errorf("day %d: expected %s, got %s", i, expected, msg.DayOfWeek)
			}
		}
	})

	t.Run("message has all fields", func(t *testing.T) {
		msg := GeneratePersonalizedMessage(user, 0, "morning")
		if msg.Workout.Title == "" {
			t.Error("workout title is empty")
		}
		if msg.Nutrition.CalorieTarget == "" {
			t.Error("calorie target is empty")
		}
		if msg.User.ID != user.ID {
			t.Error("user not set in message")
		}
		if msg.TimeOfDay != "morning" {
			t.Errorf("TimeOfDay=%q, expected 'morning'", msg.TimeOfDay)
		}
	})
}

// --- renderTemplate ---

func TestRenderTemplate(t *testing.T) {
	user := testUser("weight_loss", "moderate", "male", 25, 80)
	msg := GeneratePersonalizedMessage(user, 0, "morning")

	html, err := renderTemplate(msg)
	if err != nil {
		t.Fatalf("renderTemplate error: %v", err)
	}

	checks := []struct {
		name     string
		contains string
	}{
		{"has user name", "Иван"},
		{"has last name", "Иванов"},
		{"has workout title", "силовые для потери веса"},
		{"has calorie target", msg.Nutrition.CalorieTarget},
		{"has water intake", msg.Nutrition.WaterIntake},
		{"has day name", "Понедельник"},
		{"is HTML", "<!DOCTYPE html>"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(html, c.contains) {
				t.Errorf("rendered HTML does not contain %q", c.contains)
			}
		})
	}
}

func TestRenderTemplate_HTMLEscaping(t *testing.T) {
	user := testUser("weight_loss", "moderate", "male", 25, 80)
	user.FirstName = "<script>alert('xss')</script>"
	msg := GeneratePersonalizedMessage(user, 0, "morning")

	html, err := renderTemplate(msg)
	if err != nil {
		t.Fatalf("renderTemplate error: %v", err)
	}

	if strings.Contains(html, "<script>") {
		t.Error("HTML injection: <script> tag not escaped")
	}
}

