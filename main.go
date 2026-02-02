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
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// Version
	version = "1.4.0"

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
	Name     string `json:"name"`
	From     string `json:"from"`
	To       string `json:"to"`
	Reason   string `json:"reason"`
	Distance int    `json:"distance"` // one-way distance in km
	Province string `json:"province"` // German state abbreviation (e.g., "BW", "BY")
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

// provinceHolidays maps German state abbreviations to their holiday slices.
var provinceHolidays = map[string][]*cal.Holiday{
	"BW": de.HolidaysBW, // Baden-Württemberg
	"BY": de.HolidaysBY, // Bayern (Bavaria)
	"BE": de.HolidaysBE, // Berlin
	"BB": de.HolidaysBB, // Brandenburg
	"HB": de.HolidaysHB, // Bremen
	"HH": de.HolidaysHH, // Hamburg
	"HE": de.HolidaysHE, // Hessen (Hesse)
	"MV": de.HolidaysMV, // Mecklenburg-Vorpommern
	"NI": de.HolidaysNI, // Niedersachsen (Lower Saxony)
	"NW": de.HolidaysNW, // Nordrhein-Westfalen (North Rhine-Westphalia)
	"RP": de.HolidaysRP, // Rheinland-Pfalz (Rhineland-Palatinate)
	"SL": de.HolidaysSL, // Saarland
	"SN": de.HolidaysSN, // Sachsen (Saxony)
	"ST": de.HolidaysST, // Sachsen-Anhalt (Saxony-Anhalt)
	"SH": de.HolidaysSH, // Schleswig-Holstein
	"TH": de.HolidaysTH, // Thüringen (Thuringia)
}

// newBusinessCalendar creates a calendar with German holidays for the given province.
func newBusinessCalendar(province string) *cal.BusinessCalendar {
	c := cal.NewBusinessCalendar()
	c.Name = "Rummeyer Consulting GmbH"
	c.Description = "Default company calendar"

	holidays, ok := provinceHolidays[province]
	if !ok {
		// Default to Baden-Württemberg if invalid province
		holidays = de.HolidaysBW
	}
	c.AddHoliday(holidays...)
	return c
}

// getCustomerCalendars creates a calendar for each customer based on their province.
func getCustomerCalendars(customers []Customer) []*cal.BusinessCalendar {
	calendars := make([]*cal.BusinessCalendar, len(customers))
	for i, c := range customers {
		calendars[i] = newBusinessCalendar(c.Province)
	}
	return calendars
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
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("reisekosten v%s\n", version)
		return
	}

	// Load configuration
	cfg, err := loadConfig("config.json")
	if err != nil {
		panic(err)
	}

	// Initialize calendars per customer and parse target month
	calendars := getCustomerCalendars(cfg.Customers)
	year, month := parseMonthYear()

	// Distribute workdays among customers (round-robin, respecting each customer's holidays)
	numDays := daysInMonth(year, month)
	customerDays := make(map[int][]string, len(cfg.Customers))
	customerIdx := 0
	var lastDateString string
	totalWorkdays := 0

	for day := 1; day <= numDays; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		// Check if workday for current customer's province
		if isWorkday(calendars[customerIdx], date) {
			dateString := formatDate(year, month, day)
			customerDays[customerIdx] = append(customerDays[customerIdx], dateString)
			lastDateString = dateString
			totalWorkdays++
			customerIdx = (customerIdx + 1) % len(cfg.Customers)
		}
	}

	// Build document blocks for each customer
	kmBlocks := make([]string, 0, totalWorkdays+len(cfg.Customers))
	verpBlocks := make([]string, 0, totalWorkdays+len(cfg.Customers))
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

	// Build document headers
	kmHeader := buildDocumentHeader(year, month, lastDateString, "Kilometergelderstattung")
	verpHeader := buildDocumentHeader(year, month, lastDateString, "Verpflegungsmehraufwand")

	// Build document footers
	kmFooter := buildDocumentFooter(totalKmCost)
	verpFooter := buildDocumentFooter(verpflegungRate * float64(totalWorkdays))

	// Generate PDFs in memory
	kmFilename := fmt.Sprintf("%02d_%d_Reisekosten_Kilometergelderstattung.pdf", month, year)
	verpFilename := fmt.Sprintf("%02d_%d_Reisekosten_Verpflegungsmehraufwand.pdf", month, year)

	kmData, err := createPDF(kmHeader, kmBlocks, kmFooter)
	if err != nil {
		panic(err)
	}
	verpData, err := createPDF(verpHeader, verpBlocks, verpFooter)
	if err != nil {
		panic(err)
	}

	// Send via email
	subject := fmt.Sprintf("Reisekostenabrechnung %02d/%d", month, year)
	if err := sendEmail(cfg, subject,
		Attachment{Filename: kmFilename, Data: kmData},
		Attachment{Filename: verpFilename, Data: verpData},
	); err != nil {
		panic(err)
	}
}
