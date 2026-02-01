package main

import (
	"strings"

	"github.com/go-pdf/fpdf"
)

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
