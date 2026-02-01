// Package main generates monthly travel expense reports (Reisekosten) for business trips.
// It creates two PDF documents per month:
//   - Kilometergelderstattung (mileage reimbursement)
//   - Verpflegungsmehraufwand (meal allowance)
//
// Workdays are distributed equally among configured customers.
// The documents are automatically emailed and then deleted locally.
//
// Usage: reisekosten [M/YYYY]
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/de"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// Reimbursement rates
	kmRatePerKm     = 0.30 // EUR per kilometer
	verpflegungRate = 14.0 // 8h < 24h meal allowance

	// PDF settings
	pdfLineHeight = 5.0
	pdfFontSize   = 11
)

// monthArgRegex validates command line argument format: M/YYYY or MM/YYYY
var monthArgRegex = regexp.MustCompile(`^(0?[1-9]|1[0-2])/(20[0-9]{2})$`)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type EmailConfig struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Customer represents a client with trip details.
type Customer struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Reason   string `json:"reason"`
	Distance int    `json:"distance"` // one-way distance in km
}

type Config struct {
	SMTP      SMTPConfig  `json:"smtp"`
	Email     EmailConfig `json:"email"`
	Customers []Customer  `json:"customers"`
}

// loadConfig reads and parses the JSON configuration file.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(cfg.Customers) == 0 {
		return nil, fmt.Errorf("no customers configured")
	}

	return &cfg, nil
}

// ---------------------------------------------------------------------------
// Business Calendar
// ---------------------------------------------------------------------------

// newBusinessCalendar creates a calendar with German (Baden-Württemberg) holidays.
func newBusinessCalendar() *cal.BusinessCalendar {
	c := cal.NewBusinessCalendar()
	c.Name = "Rummeyer Consulting GmbH"
	c.Description = "Default company calendar"
	c.AddHoliday(de.HolidaysBW...)
	return c
}

// isWorkday checks if a date is a valid workday for expense reporting.
// Excludes weekends, holidays, and special December dates (24th, 27th-31st).
func isWorkday(c *cal.BusinessCalendar, date time.Time) bool {
	if !c.IsWorkday(date) {
		return false
	}

	// Exclude special December dates
	if date.Month() == 12 {
		day := date.Day()
		if day == 24 || (day >= 27 && day <= 31) {
			return false
		}
	}

	return true
}

// ---------------------------------------------------------------------------
// Day Distribution
// ---------------------------------------------------------------------------

// distributeWorkdays distributes workday dates equally among customers (round-robin).
// Returns a map of customer index to their assigned date strings.
func distributeWorkdays(workdays []string, numCustomers int) map[int][]string {
	result := make(map[int][]string, numCustomers)

	for i, date := range workdays {
		customerIdx := i % numCustomers
		result[customerIdx] = append(result[customerIdx], date)
	}

	return result
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

// parseMonthYear extracts month and year from command line args or uses current date.
func parseMonthYear() (int, time.Month) {
	if len(os.Args) > 1 && monthArgRegex.MatchString(os.Args[1]) {
		parts := strings.Split(os.Args[1], "/")
		year, _ := strconv.Atoi(parts[1])
		month, _ := strconv.Atoi(parts[0])
		return year, time.Month(month)
	}

	year, month, _ := time.Now().Date()
	return year, month
}

// daysInMonth returns the number of days in the given month.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func main() {
	// Load configuration
	cfg, err := loadConfig("config.json")
	if err != nil {
		panic(err)
	}

	// Initialize calendar and parse target month
	calendar := newBusinessCalendar()
	year, month := parseMonthYear()

	// Collect all workdays in the month
	numDays := daysInMonth(year, month)
	workdays := make([]string, 0, numDays)

	var lastDateString string
	for day := 1; day <= numDays; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		if isWorkday(calendar, date) {
			lastDateString = formatDate(year, month, day)
			workdays = append(workdays, lastDateString)
		}
	}

	// Distribute workdays equally among customers
	customerDays := distributeWorkdays(workdays, len(cfg.Customers))

	// Build document blocks for each customer
	kmBlocks := make([]string, 0, len(workdays)+len(cfg.Customers))
	verpBlocks := make([]string, 0, len(workdays)+len(cfg.Customers))
	var totalKmCost float64

	for i, customer := range cfg.Customers {
		days := customerDays[i]
		if len(days) == 0 {
			continue
		}

		// Add customer header as a block
		kmBlocks = append(kmBlocks, buildCustomerHeader(customer))
		verpBlocks = append(verpBlocks, buildCustomerHeader(customer))

		// Add entries for each assigned day
		for _, dateString := range days {
			kmBlocks = append(kmBlocks, buildKilometerEntry(dateString, customer.Distance))
			verpBlocks = append(verpBlocks, buildMealAllowanceEntry(dateString))
		}

		// Accumulate km cost for this customer
		totalKmCost += float64(len(days)) * float64(customer.Distance) * kmRatePerKm
	}

	totalWorkdays := len(workdays)

	// Build document headers
	kmHeader := buildDocumentHeader(year, month, lastDateString, "Kilometergelderstattung")
	verpHeader := buildDocumentHeader(year, month, lastDateString, "Verpflegungsmehraufwand")

	// Build document footers (totals in German number format)
	printer := message.NewPrinter(language.German)
	kmFooter := printer.Sprintf("GESAMTBETRAG: %.2f EUR\n", totalKmCost)
	verpFooter := printer.Sprintf("GESAMTBETRAG: %.2f EUR\n", verpflegungRate*float64(totalWorkdays))

	// Generate PDFs
	kmFilename := fmt.Sprintf("%02d_%d_Reisekosten_Kilometergelderstattung.pdf", month, year)
	verpFilename := fmt.Sprintf("%02d_%d_Reisekosten_Verpflegungsmehraufwand.pdf", month, year)

	createPDF(kmHeader, kmBlocks, kmFooter, kmFilename)
	createPDF(verpHeader, verpBlocks, verpFooter, verpFilename)

	// Send via email
	subject := fmt.Sprintf("Reisekostenabrechnung %02d/%d", month, year)
	if err := sendEmail(cfg, subject, kmFilename, verpFilename); err != nil {
		panic(err)
	}

	// Clean up local files
	os.Remove(kmFilename)
	os.Remove(verpFilename)
}
