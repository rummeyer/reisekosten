package main

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

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
