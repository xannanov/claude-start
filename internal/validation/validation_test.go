package validation

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid email with dots", "user.name@example.com", false},
		{"valid email with plus", "user+tag@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"empty email", "", true},
		{"no at sign", "userexample.com", true},
		{"no domain", "user@", true},
		{"no user", "@example.com", true},
		{"spaces", "user @example.com", true},
		{"double at", "user@@example.com", true},
		{"just text", "не-email", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAge(t *testing.T) {
	tests := []struct {
		name    string
		age     int
		wantErr bool
	}{
		{"minimum valid", 13, false},
		{"maximum valid", 120, false},
		{"mid range", 30, false},
		{"too young", 12, true},
		{"too old", 121, true},
		{"zero", 0, true},
		{"negative", -5, true},
		{"very large", 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAge(tt.age)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAge(%d) error = %v, wantErr %v", tt.age, err, tt.wantErr)
			}
		})
	}
}

func TestValidateHeightCm(t *testing.T) {
	tests := []struct {
		name    string
		height  int
		wantErr bool
	}{
		{"minimum valid", 100, false},
		{"maximum valid", 250, false},
		{"average", 175, false},
		{"too short", 99, true},
		{"too tall", 251, true},
		{"zero", 0, true},
		{"negative", -10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHeightCm(tt.height)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHeightCm(%d) error = %v, wantErr %v", tt.height, err, tt.wantErr)
			}
		})
	}
}

func TestValidateWeightKg(t *testing.T) {
	tests := []struct {
		name    string
		weight  float64
		wantErr bool
	}{
		{"minimum valid", 30.0, false},
		{"maximum valid", 300.0, false},
		{"average", 75.5, false},
		{"too light", 29.9, true},
		{"too heavy", 300.1, true},
		{"zero", 0, true},
		{"negative", -10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWeightKg(tt.weight)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWeightKg(%f) error = %v, wantErr %v", tt.weight, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDayOfWeek(t *testing.T) {
	tests := []struct {
		name    string
		day     int
		wantErr bool
	}{
		{"monday (0)", 0, false},
		{"sunday (6)", 6, false},
		{"wednesday (2)", 2, false},
		{"negative", -1, true},
		{"too large", 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDayOfWeek(tt.day)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDayOfWeek(%d) error = %v, wantErr %v", tt.day, err, tt.wantErr)
			}
		})
	}
}

func TestValidateHour(t *testing.T) {
	tests := []struct {
		name    string
		hour    int
		wantErr bool
	}{
		{"midnight", 0, false},
		{"max", 23, false},
		{"noon", 12, false},
		{"negative", -1, true},
		{"too large", 24, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHour(tt.hour)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHour(%d) error = %v, wantErr %v", tt.hour, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMinute(t *testing.T) {
	tests := []struct {
		name    string
		minute  int
		wantErr bool
	}{
		{"zero", 0, false},
		{"max", 59, false},
		{"mid", 30, false},
		{"negative", -1, true},
		{"too large", 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinute(tt.minute)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMinute(%d) error = %v, wantErr %v", tt.minute, err, tt.wantErr)
			}
		})
	}
}
