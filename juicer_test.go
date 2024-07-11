package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/jung-kurt/gofpdf"
)

type Font struct {
	face  string
	style string  // B, I, U, S
	size  float64 // 8, 10, 12, 14, 16, 20
}
type TableText struct {
	text  string
	font  Font   // "Courier" for fixed-width, "Helvetica" or "Arial" for sans serif, "Times" for serif, "Symbol" or "ZapfDingbats"
	align string // "L", "C" or "R" (left, center, right) in alignStr.
	// Vertical alignment is controlled by including "T", "M", "B" or "A" (top, middle, bottom, baseline) in alignStr.
	// The default alignment is left middle.
	border string // "0" for no border, "1" for border around cell, "L", "T", "R", "B" for left, top, right, bottom.
}

var (
	Arial12  = Font{"Arial", "", 12}
	Arial12B = Font{"Arial", "B", 12}
)
var blankCell = TableText{}

func TestInvoice(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(10, 10, 30)
	//drawGrid(pdf)

	pdf.SetTitle("TAX INVOICE", false)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(10, 20, "TAX INVOICE")
	pdf.ImageOptions("logo.png", 150, 20, 35, 35, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	pdf.SetXY(10, 29)
	addAddress(pdf, Address{"Vermont Medical Clinic", "123 Main St", "Vermont SOUTH VIC 3133", "123456789"})
	pdf.Ln(10)
	addAddressDate(pdf, Address{"Dr Jim Glaspole", "37A Park Crescent", "Fairfield VIC 3078", "123456789"}, "05-05-24")
	pdf.Ln(10)
	addInvoiceDetails(pdf, "Dr Jim Glaspole", "jim@yeatpole.com", "29/04/2024", "05/05/2024", "JG20240505")
	pdf.Ln(10)
	addTotal(pdf, 974327, 97433)

	pdf.AddPage()
	pdf.SetMargins(10, 10, 30)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(10, 20, "Service Fee Calculations")
	pdf.SetXY(10, 29)
	//
	// remember `` will honour newlines
	//
	//_, lineHt := pdf.GetFontSize()
	/* 	pdf.MultiCell(0, lineHt*1.5, "This is a longer text that should be broken down into multiple lines.\n"+
	"The text is long enough to wrap around the page and should be broken down into multiple lines. "+
	"The text is long enough to wrap around the page and should be broken down into multiple lines. "+
	"The text is long enough to wrap around the page and should be broken down into multiple lines. "+
	"The text is long enough to wrap around the page and should be broken down into multiple lines. "+
	"The text is long enough to wrap around the page and should be broken down into multiple lines.",
	gofpdf.BorderFull, gofpdf.AlignRight, false)
	*/
	err := pdf.OutputFileAndClose("hello.pdf")
	if err != nil {
		log.Fatal(err)
	}
}

func addAddress(pdf *gofpdf.Fpdf, address Address) {
	tableData := [][]TableText{
		{TableText{text: "From:", font: Arial12B}, TableText{text: address.CompanyName}},
		{blankCell, TableText{text: address.StreetAddress}},
		{blankCell, TableText{text: address.City}},
		{TableText{text: "ABN", font: Arial12B}, TableText{text: address.ABN}},
	}
	addTable(pdf, tableData, []float64{40, 150}, 5)
}
func addAddressDate(pdf *gofpdf.Fpdf, address Address, date string) {
	tableData := [][]TableText{
		{TableText{text: "From:", font: Arial12B}, TableText{text: address.CompanyName}, TableText{text: date, align: "R"}},
		{blankCell, TableText{text: address.StreetAddress}, blankCell},
		{blankCell, TableText{text: address.City}, blankCell},
		{TableText{text: "ABN", font: Arial12B}, TableText{text: address.ABN}, blankCell},
	}
	addTable(pdf, tableData, []float64{40, 150, 0}, 5)
}
func addInvoiceDetails(pdf *gofpdf.Fpdf, prac string, email string, periodStart string, periodEnd string, invoiceNo string) {
	tableData := [][]TableText{
		{TableText{text: "Practitioner:", font: Arial12B}, TableText{text: prac}, TableText{text: "Invoice Number", font: Arial12B, align: "R"}},
		{TableText{text: "Period:", font: Arial12B}, TableText{text: periodStart + " - " + periodEnd}, TableText{text: invoiceNo, align: "R"}},
		{TableText{text: "Email", font: Arial12B}, TableText{text: email}, blankCell},
	}
	addTable(pdf, tableData, []float64{40, 150, 0}, 5)

}
func addTotal(pdf *gofpdf.Fpdf, serviceFeeTotal int, gst int) {
	tableData := [][]TableText{
		{blankCell, TableText{text: "Service Fee (see calculation sheet)"}, TableText{text: pct(serviceFeeTotal), align: "R"}},
		{blankCell, TableText{text: "GST on Service Fee", border: "B"}, TableText{text: pct(gst), align: "R", border: "B"}},
		{blankCell, TableText{text: "Total Service Fee", border: "B"}, TableText{text: pct(serviceFeeTotal+gst), align: "R", border: "B"}},
	}
	addTable(pdf, tableData, []float64{40, 150, 0}, 5)
}
func pct(v int) string {
	return fmt.Sprintf("%d.%d", v/100, v%100)
}
func addTable(pdf *gofpdf.Fpdf, data [][]TableText, colWidths []float64, height float64) {
	for _, row := range data {
		for i, value := range row {
			//
			// set default font if there is some text
			//
			var font = value.font
			if font.face == "" || value.text != "" {
				font = Arial12
			}
			if font.face != "" {
				pdf.SetFont(value.font.face, value.font.style, value.font.size)
			}
			pdf.CellFormat(colWidths[i], height, value.text, value.border, 0, value.align, false, 0, "")
		}
		pdf.Ln(-1)
	}
}

func drawGrid(pdf *gofpdf.Fpdf) {
	w, h := pdf.GetPageSize()
	pdf.SetFont("courier", "", 12)
	pdf.SetTextColor(80, 80, 80)
	pdf.SetDrawColor(200, 200, 200)
	_, lineHt := pdf.GetFontSize()

	for x := 0.0; x <= w; x += w / 20 {
		pdf.Line(x, 0, x, h)
		pdf.Text(x, lineHt, fmt.Sprintf("%d", int(x)))
	}

	for y := 0.0; y <= h; y += h / 20 {
		pdf.Line(0, y, w, y)
		pdf.Text(0, y, fmt.Sprintf("%d", int(y)))
	}
}

func columns(pdf *gofpdf.Fpdf) {
	setCol := func(col int) {
		// Set position at a given column
		//crrntCol = col
		x := float64(col) * 65.0
		pdf.SetLeftMargin(x)
		pdf.SetX(x)
	}
	pdf.SetY(30)
	setCol(0)
	setCol(1)
}
