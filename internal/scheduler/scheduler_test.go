package scheduler

import (
	"testing"
	"time"
)

func TestGoWeekdayToISO(t *testing.T) {
	tests := []struct {
		name       string
		goWeekday  time.Weekday
		wantISO    int
	}{
		{"Sunday (Go 0) → ISO 6", time.Sunday, 6},
		{"Monday (Go 1) → ISO 0", time.Monday, 0},
		{"Tuesday (Go 2) → ISO 1", time.Tuesday, 1},
		{"Wednesday (Go 3) → ISO 2", time.Wednesday, 2},
		{"Thursday (Go 4) → ISO 3", time.Thursday, 3},
		{"Friday (Go 5) → ISO 4", time.Friday, 4},
		{"Saturday (Go 6) → ISO 5", time.Saturday, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GoWeekdayToISO(tt.goWeekday)
			if got != tt.wantISO {
				t.Errorf("GoWeekdayToISO(%v) = %d, want %d", tt.goWeekday, got, tt.wantISO)
			}
		})
	}
}

func TestGoWeekdayToISO_AllDays(t *testing.T) {
	// Проверяем что все 7 дней уникальны и в диапазоне [0, 6]
	seen := make(map[int]bool)
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		iso := GoWeekdayToISO(wd)
		if iso < 0 || iso > 6 {
			t.Errorf("GoWeekdayToISO(%v) = %d, out of range [0,6]", wd, iso)
		}
		if seen[iso] {
			t.Errorf("GoWeekdayToISO(%v) = %d, duplicate ISO value", wd, iso)
		}
		seen[iso] = true
	}
	if len(seen) != 7 {
		t.Errorf("expected 7 unique ISO values, got %d", len(seen))
	}
}
