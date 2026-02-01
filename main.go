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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/go-pdf/fpdf"
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
// Document Generation Helpers
// ---------------------------------------------------------------------------

// shortID generates a random alphanumeric ID of the specified length.
// Used for document reference numbers (Belegnummer).
func shortID(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	b := make([]byte, length)
	rand.Read(b)

	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}

	return string(b)
}

// formatDate formats a date as DD.MM.YYYY (German format).
func formatDate(year int, month time.Month, day int) string {
	return fmt.Sprintf("%02d.%02d.%d", day, month, year)
}

// ---------------------------------------------------------------------------
// Document Content Builders
// ---------------------------------------------------------------------------

// buildDocumentHeader creates the header section with date, reference number, and title.
func buildDocumentHeader(year int, month time.Month, dateString, title string) string {
	var b strings.Builder

	// Right-aligned date and reference number
	fmt.Fprintf(&b, "                                                               DATUM:   %s\n", dateString)
	fmt.Fprintf(&b, "                                                               BELEGNR: %s\n", shortID(6))
	b.WriteString("\n")

	// Document title
	fmt.Fprintf(&b, "Reisekosten %s %02d/%d\n", title, month, year)
	b.WriteString("===========================================\n\n")

	return b.String()
}

// buildCustomerHeader creates the trip info header for a customer.
func buildCustomerHeader(c Customer) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s)\n", c.ID)
	fmt.Fprintf(&b, "Von: %s\n", c.From)
	fmt.Fprintf(&b, "Nach: %s\n", c.To)
	fmt.Fprintf(&b, "Grund: %s\n\n", c.Reason)

	return b.String()
}

// buildKilometerEntry creates a single mileage reimbursement entry for a given date.
func buildKilometerEntry(dateString string, distanceKm int) string {
	var b strings.Builder

	amount := float64(distanceKm) * kmRatePerKm
	fmt.Fprintf(&b, "Anreise: %s\n", dateString)
	fmt.Fprintf(&b, "Abreise: %s\n", dateString)
	fmt.Fprintf(&b, "Fahrkosten (%dkm x 0,30 EUR):%s%.2f EUR\n\n",
		distanceKm, padding(distanceKm), amount)

	return b.String()
}

// padding returns spaces to align the amount column based on distance digits.
func padding(distanceKm int) string {
	// Base padding for single digit, reduce for each additional digit
	switch {
	case distanceKm >= 100:
		return "           "
	case distanceKm >= 10:
		return "            "
	default:
		return "             "
	}
}

// buildMealAllowanceEntry creates a single meal allowance entry for a given date.
func buildMealAllowanceEntry(dateString string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Anreise: %s, 07:00\n", dateString)
	fmt.Fprintf(&b, "Abreise: %s, 17:00\n", dateString)
	b.WriteString("Verpflegungsmehraufwand (8h < 24h):      14,-- EUR\n\n")

	return b.String()
}

// ---------------------------------------------------------------------------
// PDF Generation
// ---------------------------------------------------------------------------

// createPDF generates a PDF document with smart page breaks.
// Blocks are never split across pages - if a block doesn't fit, a new page is added.
func createPDF(header string, blocks []string, footer string, filename string) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Courier", "", pdfFontSize)
	pdf.AddPage()

	// Calculate available page height
	_, pageHeight := pdf.GetPageSize()
	_, _, _, marginBottom := pdf.GetMargins()
	maxY := pageHeight - marginBottom

	// Use large width to prevent line wrapping (text uses spaces for alignment)
	const cellWidth = 300

	// Write header (always fits on first page)
	pdf.MultiCell(cellWidth, pdfLineHeight, header, "", "", false)

	// Write each block, adding page break if block won't fit
	for _, block := range blocks {
		blockHeight := float64(strings.Count(block, "\n")+1) * pdfLineHeight

		if pdf.GetY()+blockHeight > maxY {
			pdf.AddPage()
		}
		pdf.MultiCell(cellWidth, pdfLineHeight, block, "", "", false)
	}

	// Write footer (total amount)
	footerHeight := float64(strings.Count(footer, "\n")+1) * pdfLineHeight
	if pdf.GetY()+footerHeight > maxY {
		pdf.AddPage()
	}
	pdf.MultiCell(cellWidth, pdfLineHeight, footer, "", "", false)

	if err := pdf.OutputFileAndClose(filename); err != nil {
		panic(err)
	}
}

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

// sendEmail sends the generated PDFs via SMTP.
func sendEmail(cfg *Config, subject string, filenames ...string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", cfg.Email.From)
	msg.SetHeader("To", cfg.Email.To)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", "Dokumente anbei.<br>")

	for _, f := range filenames {
		msg.Attach(f)
	}

	dialer := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)
	return dialer.DialAndSend(msg)
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
