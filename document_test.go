package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    time.Month
		day      int
		expected string
	}{
		{"single digit day and month", 2026, 1, 5, "05.01.2026"},
		{"double digit day and month", 2026, 12, 25, "25.12.2026"},
		{"first day of year", 2026, 1, 1, "01.01.2026"},
		{"last day of year", 2026, 12, 31, "31.12.2026"},
		{"leap year date", 2024, 2, 29, "29.02.2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDate(tt.year, tt.month, tt.day)
			if got != tt.expected {
				t.Errorf("formatDate(%d, %d, %d) = %q, want %q", tt.year, tt.month, tt.day, got, tt.expected)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		expected string
	}{
		{"zero", 0, "0,00"},
		{"integer amount", 14, "14,00"},
		{"decimal amount", 30.60, "30,60"},
		{"large amount", 1234.56, "1234,56"},
		{"small amount", 0.30, "0,30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAmount(tt.amount)
			if got != tt.expected {
				t.Errorf("formatAmount(%v) = %q, want %q", tt.amount, got, tt.expected)
			}
		})
	}
}

func TestRightAlign(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		width    int
		expected string
	}{
		{"shorter than width", "hello", 10, "     hello"},
		{"equal to width", "hello", 5, "hello"},
		{"longer than width", "hello world", 5, "hello world"},
		{"empty string", "", 5, "     "},
		{"width zero", "hello", 0, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rightAlign(tt.s, tt.width)
			if got != tt.expected {
				t.Errorf("rightAlign(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.expected)
			}
		})
	}
}

func TestDocumentID(t *testing.T) {
	id := documentID(2026, 2)

	// Check prefix
	if !strings.HasPrefix(id, "RK-2026-02-") {
		t.Errorf("documentID(2026, 2) = %q, want prefix RK-2026-02-", id)
	}

	// Check total length: "RK-2026-02-XXXX" = 15
	if len(id) != 15 {
		t.Errorf("documentID length = %d, want 15", len(id))
	}

	// Check suffix is alphanumeric
	suffix := id[11:]
	for _, c := range suffix {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			t.Errorf("documentID suffix %q contains invalid character %c", suffix, c)
		}
	}

	// Check uniqueness (two calls should differ)
	id2 := documentID(2026, 2)
	if id == id2 {
		t.Logf("Warning: two documentID calls returned same value %q (possible but unlikely)", id)
	}
}

func TestBuildCustomerHeader(t *testing.T) {
	c := Customer{
		ID:     "1",
		Name:   "Acme Corp",
		From:   "Stuttgart",
		To:     "München",
		Reason: "Projektarbeit",
	}

	got := buildCustomerHeader(c)

	checks := []string{
		"1) Acme Corp",
		"Von:    Stuttgart",
		"Nach:   München",
		"Grund:  Projektarbeit",
		lineSingle,
	}

	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("buildCustomerHeader missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildKilometerEntry(t *testing.T) {
	got := buildKilometerEntry("13.02.2026", 100)

	checks := []string{
		"13.02.2026",
		"Fahrkosten (100 km x 0,30 EUR)",
		"30,00 EUR",
	}

	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("buildKilometerEntry missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildKilometerEntryCalculation(t *testing.T) {
	tests := []struct {
		distance int
		amount   string
	}{
		{50, "15,00 EUR"},
		{1, "0,30 EUR"},
		{200, "60,00 EUR"},
	}

	for _, tt := range tests {
		t.Run(tt.amount, func(t *testing.T) {
			got := buildKilometerEntry("01.01.2026", tt.distance)
			if !strings.Contains(got, tt.amount) {
				t.Errorf("buildKilometerEntry with distance %d missing amount %q", tt.distance, tt.amount)
			}
		})
	}
}

func TestBuildMealAllowanceEntry(t *testing.T) {
	got := buildMealAllowanceEntry("13.02.2026")

	checks := []string{
		"13.02.2026",
		"07:00 - 17:00",
		"Verpflegungsmehraufwand (8h - 24h)",
		"14,00 EUR",
	}

	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("buildMealAllowanceEntry missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildDocumentFooter(t *testing.T) {
	got := buildDocumentFooter(150.00)

	checks := []string{
		"GESAMTBETRAG:",
		"150,00 EUR",
		lineSingle,
		lineDouble,
	}

	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("buildDocumentFooter missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildDocumentFooterZero(t *testing.T) {
	got := buildDocumentFooter(0)
	if !strings.Contains(got, "0,00 EUR") {
		t.Errorf("buildDocumentFooter(0) missing 0,00 EUR in:\n%s", got)
	}
}
