// Package main generates monthly travel expense reports (Reisekosten) for business trips.
// It creates two PDF documents per month:
//   - Kilometergelderstattung (mileage reimbursement)
//   - Verpflegungsmehraufwand (meal allowance)
//
// Workdays are distributed equally among configured customers.
// The documents are automatically emailed and then deleted locally.
//
// Usage: reisekosten [--config path] [M/YYYY]
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/de"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// Version
	version = "1.10.0"

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
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type EmailConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// Customer represents a client with trip details.
type Customer struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	From     string `yaml:"from"`
	To       string `yaml:"to"`
	Reason   string `yaml:"reason"`
	Distance int    `yaml:"distance"` // one-way distance in km
	Province string `yaml:"province"` // German state abbreviation (e.g., "BW", "BY")
}

type Config struct {
	SMTP             SMTPConfig  `yaml:"smtp"`
	Email            EmailConfig `yaml:"email"`
	Customers        []Customer  `yaml:"customers"`
	ChristmasWeekOff *bool       `yaml:"christmasWeekOff,omitempty"` // exclude Dec 24, 27-31 (default: true)
}

// ChristmasWeekOffEnabled returns whether the Christmas/New Year week off is enabled.
// Defaults to true if not specified.
func (c *Config) ChristmasWeekOffEnabled() bool {
	return c.ChristmasWeekOff == nil || *c.ChristmasWeekOff
}

// findConfigFile searches for the config file in the current directory first,
// then in the directory of the running executable.
func findConfigFile(filename string) (string, error) {
	// Try current directory first
	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	// Try executable directory
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		exeConfigPath := filepath.Join(exeDir, filename)
		if _, err := os.Stat(exeConfigPath); err == nil {
			return exeConfigPath, nil
		}
	}

	return "", fmt.Errorf("config file %q not found in current directory or executable directory", filename)
}

// loadConfig reads and parses the YAML configuration file.
// If configPath is non-empty, it uses that path directly.
// Otherwise, it searches for the file in the current directory and executable directory.
func loadConfig(filename, configPath string) (*Config, error) {
	var path string
	var err error

	if configPath != "" {
		path = configPath
	} else {
		path, err = findConfigFile(filename)
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
// Excludes weekends, holidays, and optionally Christmas/New Year week off (Dec 24, 27-31).
func isWorkday(c *cal.BusinessCalendar, date time.Time, christmasWeekOff bool) bool {
	if !c.IsWorkday(date) {
		return false
	}

	// Exclude Christmas/New Year week off (Dec 24 + Dec 27-31)
	if christmasWeekOff && date.Month() == 12 {
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

// parseArgs parses command line arguments and returns config path, year, and month.
func parseArgs() (configPath string, year int, month time.Month) {
	args := os.Args[1:]

	// Parse --config flag
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			// Remove --config and its value from args
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	// Parse month/year from remaining args
	for _, arg := range args {
		if monthArgRegex.MatchString(arg) {
			parts := strings.Split(arg, "/")
			year, _ = strconv.Atoi(parts[1])
			m, _ := strconv.Atoi(parts[0])
			month = time.Month(m)
			return
		}
	}

	// Default to current date
	year, month, _ = time.Now().Date()
	return
}

// daysInMonth returns the number of days in the given month.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func main() {
	// Handle --version flag
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("reisekosten v%s\n", version)
			return
		}
	}

	// Parse command line arguments
	configPath, year, month := parseArgs()

	// Load configuration
	cfg, err := loadConfig("config.yaml", configPath)
	if err != nil {
		panic(err)
	}

	// Initialize calendars per customer
	calendars := getCustomerCalendars(cfg.Customers)

	// Distribute workdays among customers (round-robin, respecting each customer's holidays)
	numDays := daysInMonth(year, month)
	customerDays := make(map[int][]string, len(cfg.Customers))
	customerIdx := 0
	var firstDateString, lastDateString string
	totalWorkdays := 0

	for day := 1; day <= numDays; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		// Check if workday for current customer's province
		if isWorkday(calendars[customerIdx], date, cfg.ChristmasWeekOffEnabled()) {
			dateString := formatDate(year, month, day)
			customerDays[customerIdx] = append(customerDays[customerIdx], dateString)
			if firstDateString == "" {
				firstDateString = dateString
			}
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
	kmHeader := buildDocumentHeader(year, month, lastDateString, firstDateString, lastDateString, "Kilometergelderstattung")
	verpHeader := buildDocumentHeader(year, month, lastDateString, firstDateString, lastDateString, "Verpflegungsmehraufwand")

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
	subject := fmt.Sprintf("Deine Reisekostenabrechnung %02d/%d", month, year)
	if err := sendEmail(cfg, subject,
		Attachment{Filename: kmFilename, Data: kmData},
		Attachment{Filename: verpFilename, Data: verpData},
	); err != nil {
		panic(err)
	}
}
