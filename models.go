package main

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// User represents a gym motivation user
type User struct {
	ID                 string      `json:"id"`
	Email              string      `json:"email"`
	FirstName          string      `json:"first_name"`
	LastName           string      `json:"last_name"`
	Age                int         `json:"age"`
	Gender             string      `json:"gender"`
	HeightCm           int         `json:"height_cm"`
	WeightKg           float64     `json:"weight_kg"`
	Goal               string      `json:"goal"`
	ActivityLevel      string      `json:"activity_level"`
	WorkoutPreferences JSONB       `json:"workout_preferences"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// UserSchedule represents when a user should receive emails
type UserSchedule struct {
	ID          int    `json:"id"`
	UserID      string `json:"user_id"`
	DayOfWeek   int    `json:"day_of_week"` // 0 = Monday, 6 = Sunday
	TimeHour    int    `json:"time_hour"`
	TimeMinute  int    `json:"time_minute"`
	EmailType   string `json:"email_type"` // morning, afternoon, evening
	IsActive    bool   `json:"is_active"`
}

// JSONB implements driver.Valuer and sql.Scanner for JSONB type
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

// WorkoutPlan represents a personalized workout
type WorkoutPlan struct {
	Title          string   `json:"title"`
	Exercises      []string `json:"exercises"`
	Duration       string   `json:"duration"`
	Description    string   `json:"description"`
	Sets           []string `json:"sets"`
	Reps           []string `json:"reps"`
}

// NutritionPlan represents a personalized nutrition plan
type NutritionPlan struct {
	Breakfast      string   `json:"breakfast"`
	Lunch          string   `json:"lunch"`
	Dinner         string   `json:"dinner"`
	Snacks         []string `json:"snacks"`
	ProteinGoal    string   `json:"protein_goal"`
	CalorieTarget  string   `json:"calorie_target"`
	WaterIntake    string   `json:"water_intake"`
}

// PersonalizedMessage represents a fully personalized email message
type PersonalizedMessage struct {
	Subject        string
	Body           string
	Workout        WorkoutPlan
	Nutrition      NutritionPlan
	User           User
	DayOfWeek      string
	TimeOfDay      string
}
