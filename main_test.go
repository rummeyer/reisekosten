package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    time.Month
		expected int
	}{
		{"January", 2026, 1, 31},
		{"February non-leap", 2025, 2, 28},
		{"February leap", 2024, 2, 29},
		{"March", 2026, 3, 31},
		{"April", 2026, 4, 30},
		{"May", 2026, 5, 31},
		{"June", 2026, 6, 30},
		{"July", 2026, 7, 31},
		{"August", 2026, 8, 31},
		{"September", 2026, 9, 30},
		{"October", 2026, 10, 31},
		{"November", 2026, 11, 30},
		{"December", 2026, 12, 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := daysInMonth(tt.year, tt.month)
			if got != tt.expected {
				t.Errorf("daysInMonth(%d, %d) = %d, want %d", tt.year, tt.month, got, tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestChristmasWeekOffEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{"nil defaults to true", Config{ChristmasWeekOff: nil}, true},
		{"explicit true", Config{ChristmasWeekOff: boolPtr(true)}, true},
		{"explicit false", Config{ChristmasWeekOff: boolPtr(false)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ChristmasWeekOffEnabled()
			if got != tt.expected {
				t.Errorf("ChristmasWeekOffEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewBusinessCalendar(t *testing.T) {
	// Valid province
	cal := newBusinessCalendar("BY")
	if cal == nil {
		t.Fatal("newBusinessCalendar(BY) returned nil")
	}

	// Invalid province defaults to BW (should not panic)
	cal = newBusinessCalendar("INVALID")
	if cal == nil {
		t.Fatal("newBusinessCalendar(INVALID) returned nil")
	}

	// Empty province
	cal = newBusinessCalendar("")
	if cal == nil {
		t.Fatal("newBusinessCalendar('') returned nil")
	}
}

func TestGetCustomerCalendars(t *testing.T) {
	customers := []Customer{
		{Province: "BW"},
		{Province: "BY"},
		{Province: "BE"},
	}

	calendars := getCustomerCalendars(customers)
	if len(calendars) != len(customers) {
		t.Errorf("getCustomerCalendars returned %d calendars, want %d", len(calendars), len(customers))
	}

	for i, c := range calendars {
		if c == nil {
			t.Errorf("calendar[%d] is nil", i)
		}
	}
}

func TestIsWorkday(t *testing.T) {
	cal := newBusinessCalendar("BW")

	tests := []struct {
		name             string
		date             time.Time
		christmasWeekOff bool
		expected         bool
	}{
		{"regular weekday", time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), true, true},             // Tuesday
		{"Saturday", time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC), true, false},                    // Saturday
		{"Sunday", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), true, false},                      // Sunday
		{"New Years Day", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), true, false},                // Holiday
		{"Christmas Eve off", time.Date(2026, 12, 24, 0, 0, 0, 0, time.UTC), true, false},          // Dec 24 with flag
		{"Christmas Eve on", time.Date(2025, 12, 24, 0, 0, 0, 0, time.UTC), false, true},           // Dec 24 without flag (Wednesday)
		{"Dec 28 off", time.Date(2026, 12, 28, 0, 0, 0, 0, time.UTC), true, false},                 // Dec 28 with flag (Monday)
		{"Dec 28 on", time.Date(2026, 12, 28, 0, 0, 0, 0, time.UTC), false, true},                  // Dec 28 without flag
		{"Dec 31 off", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), true, false},                 // Dec 31 with flag (Wednesday)
		{"Dec 31 on", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), false, true},                  // Dec 31 without flag
		{"Dec 26 not in range", time.Date(2026, 12, 26, 0, 0, 0, 0, time.UTC), true, false},        // Dec 26 is Zweiter Weihnachtstag (Saturday in 2026)
		{"regular Dec day", time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), true, true},              // Dec 1 (Tuesday)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWorkday(cal, tt.date, tt.christmasWeekOff)
			if got != tt.expected {
				t.Errorf("isWorkday(%s, christmasWeekOff=%v) = %v, want %v",
					tt.date.Format("2006-01-02 Monday"), tt.christmasWeekOff, got, tt.expected)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.yaml")
		content := `smtp:
  host: smtp.example.com
  port: 587
  user: user@example.com
  pass: secret
email:
  from: user@example.com
  to: boss@example.com
customers:
  - id: "1"
    name: Acme Corp
    from: Stuttgart
    to: München
    reason: Projektarbeit
    distance: 100
    province: BW
`
		os.WriteFile(configFile, []byte(content), 0644)

		cfg, err := loadConfig("config.yaml", configFile)
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if len(cfg.Customers) != 1 {
			t.Errorf("expected 1 customer, got %d", len(cfg.Customers))
		}
		if cfg.Customers[0].Name != "Acme Corp" {
			t.Errorf("expected customer name 'Acme Corp', got %q", cfg.Customers[0].Name)
		}
		if cfg.Customers[0].Distance != 100 {
			t.Errorf("expected distance 100, got %d", cfg.Customers[0].Distance)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadConfig("config.yaml", "/nonexistent/config.yaml")
		if err == nil {
			t.Error("loadConfig() expected error for missing file")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.yaml")
		os.WriteFile(configFile, []byte("{{invalid yaml"), 0644)

		_, err := loadConfig("config.yaml", configFile)
		if err == nil {
			t.Error("loadConfig() expected error for invalid YAML")
		}
	})

	t.Run("no customers", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.yaml")
		content := `smtp:
  host: smtp.example.com
customers: []
`
		os.WriteFile(configFile, []byte(content), 0644)

		_, err := loadConfig("config.yaml", configFile)
		if err == nil {
			t.Error("loadConfig() expected error for no customers")
		}
	})

	t.Run("christmasWeekOff defaults to true", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.yaml")
		content := `customers:
  - id: "1"
    name: Test
    from: A
    to: B
    reason: Test
    distance: 10
    province: BW
`
		os.WriteFile(configFile, []byte(content), 0644)

		cfg, err := loadConfig("config.yaml", configFile)
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if !cfg.ChristmasWeekOffEnabled() {
			t.Error("expected ChristmasWeekOffEnabled() to be true by default")
		}
	})
}

func TestCreatePDF(t *testing.T) {
	header := "Test Header\n"
	blocks := []string{"Block 1\nLine 2\n", "Block 2\n"}
	footer := "Footer\n"

	data, err := createPDF(header, blocks, footer)
	if err != nil {
		t.Fatalf("createPDF() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("createPDF() returned empty data")
	}

	// Check PDF magic bytes
	if len(data) < 4 || string(data[:4]) != "%PDF" {
		t.Error("createPDF() output does not start with PDF magic bytes")
	}
}

func TestCreatePDFEmpty(t *testing.T) {
	data, err := createPDF("", nil, "")
	if err != nil {
		t.Fatalf("createPDF() with empty input error = %v", err)
	}
	if len(data) == 0 {
		t.Error("createPDF() with empty input returned empty data")
	}
}
