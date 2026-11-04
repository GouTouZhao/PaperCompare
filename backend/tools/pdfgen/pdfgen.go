package pdfgen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
)

type SamplePage struct {
	Heading string
	Lines   []string
}

func GenerateSamplePDF(outputPath string, title string, authors string, year int, doi string, pages []SamplePage) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.SetAuthor(authors, false)

	for idx, pg := range pages {
		pdf.AddPage()
		if idx == 0 {
			pdf.SetFont("Helvetica", "B", 15)
			pdf.MultiCell(190, 8, title, "", "L", false)
			pdf.Ln(2)
			pdf.SetFont("Helvetica", "I", 10)
			pdf.CellFormat(190, 6, fmt.Sprintf("Authors: %s   |   Year: %d   |   DOI: %s", authors, year, doi), "", 1, "L", false, 0, "")
			pdf.Ln(4)
		}

		if pg.Heading != "" {
			pdf.SetFont("Helvetica", "B", 13)
			pdf.CellFormat(190, 8, pg.Heading, "", 1, "L", false, 0, "")
			pdf.Ln(2)
		}

		pdf.SetFont("Helvetica", "", 10.5)
		for _, line := range pg.Lines {
			pdf.MultiCell(190, 6, line, "", "L", false)
			pdf.Ln(2)
		}
	}

	return pdf.OutputFileAndClose(outputPath)
}

