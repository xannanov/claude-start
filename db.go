package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

var db *sql.DB

// ConnectToDatabase establishes connection to PostgreSQL
func ConnectToDatabase() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	connStr := getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8")
	var err error

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL database")
	return nil
}

// CloseDatabase closes the database connection
func CloseDatabase() {
	if db != nil {
		db.Close()
		log.Println("Database connection closed")
	}
}

// GetUserByID retrieves a user by ID
func GetUserByID(id string) (*User, error) {
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, email, first_name, last_name, age, gender,
		       height_cm, weight_kg, goal, activity_level, workout_preferences
		FROM users
		WHERE id = $1 AND is_active = true
	`

	row := db.QueryRow(query, id)
	user := &User{}

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Age,
		&user.Gender,
		&user.HeightCm,
		&user.WeightKg,
		&user.Goal,
		&user.ActivityLevel,
		&user.WorkoutPreferences,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(email string) (*User, error) {
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, email, first_name, last_name, age, gender,
		       height_cm, weight_kg, goal, activity_level, workout_preferences
		FROM users
		WHERE email = $1 AND is_active = true
	`

	row := db.QueryRow(query, email)
	user := &User{}

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Age,
		&user.Gender,
		&user.HeightCm,
		&user.WeightKg,
		&user.Goal,
		&user.ActivityLevel,
		&user.WorkoutPreferences,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return user, nil
}

// GetActiveSchedulesForDay retrieves all active schedules for a specific day
func GetActiveSchedulesForDay(dayOfWeek int) ([]UserSchedule, error) {
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, user_id, day_of_week, time_hour, time_minute, email_type
		FROM user_schedules
		WHERE day_of_week = $1 AND is_active = true
		ORDER BY time_hour, time_minute
	`

	rows, err := db.Query(query, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	var schedules []UserSchedule
	for rows.Next() {
		var schedule UserSchedule
		err := rows.Scan(&schedule.ID, &schedule.UserID, &schedule.DayOfWeek,
			&schedule.TimeHour, &schedule.TimeMinute, &schedule.EmailType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

// CreateUser creates a new user in the database
func CreateUser(db *sql.DB, user *User) error {
	query := `
		INSERT INTO users (
			email, first_name, last_name, age, gender,
			height_cm, weight_kg, goal, activity_level, workout_preferences, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err := db.QueryRow(
		query,
		user.Email,
		user.FirstName,
		user.LastName,
		user.Age,
		user.Gender,
		user.HeightCm,
		user.WeightKg,
		user.Goal,
		user.ActivityLevel,
		user.WorkoutPreferences,
		true,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// CreateUserSchedule creates a schedule for a user
func CreateUserSchedule(db *sql.DB, schedule *UserSchedule) error {
	query := `
		INSERT INTO user_schedules (user_id, day_of_week, time_hour, time_minute, email_type, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (user_id, day_of_week, time_hour, time_minute)
		DO UPDATE SET is_active = true, updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	var id int
	err := db.QueryRow(
		query,
		schedule.UserID,
		schedule.DayOfWeek,
		schedule.TimeHour,
		schedule.TimeMinute,
		schedule.EmailType,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to create/update schedule: %w", err)
	}

	return nil
}

// GetAllUsers retrieves all users
func GetAllUsers() ([]User, error) {
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, email, first_name, last_name, age, gender,
		       height_cm, weight_kg, goal, activity_level, workout_preferences,
		       created_at, updated_at
		FROM users
		WHERE is_active = true
		ORDER BY email
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Age,
			&user.Gender,
			&user.HeightCm,
			&user.WeightKg,
			&user.Goal,
			&user.ActivityLevel,
			&user.WorkoutPreferences,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
