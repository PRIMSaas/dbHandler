package main

import (
	"bytes"
	"fmt"
	"time"

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

func makePdf(provider string, details PaymentTotals, adjustments []Adjustments, companyDetails Address, providerAddr Address) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(10, 10, 30)

	pdf.SetTitle("TAX INVOICE", false)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(10, 20, "TAX INVOICE")
	pdf.ImageOptions("logo.jpg", 150, 20, 35, 35, false, gofpdf.ImageOptions{ImageType: "JPEG", ReadDpi: true}, 0, "")
	pdf.SetXY(10, 29)
	addAddress(pdf, companyDetails)
	pdf.Ln(10)
	addAddressDate(pdf, providerAddr, time.Now().Format("02-01-06"))
	pdf.Ln(10)
	addInvoiceDetails(pdf, provider, providerAddr.Email, "01/01/2024", "05/05/2024", "JG20240505")
	pdf.Ln(10)
	addTotal(pdf, details.ServiceCutTotal, details.GSTTotal, details.AdjustmentTotal)

	pdf.AddPage()
	pdf.SetMargins(10, 10, 30)
	//drawGrid(pdf)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(10, 20, "Service Fee Calculations")
	pdf.SetXY(10, 29)
	addServiceFeeCalculation(pdf, details)
	pdf.Ln(5)
	addAdjustments(pdf, adjustments)
	addTotalCalc(pdf, details.ServiceCutTotal, details.GSTTotal)

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	/* 	err = os.WriteFile("invoice.pdf", buf.Bytes(), 0644)
	   	if err != nil {
	   		logError.Printf("Error writing PDF: %v", err)
	   	}
	*/
	return buf.Bytes(), nil
}

/*
Patient    string     `json:"patient"`
InvoiceNo  string     `json:"invoiceNo"`
ItemNo     string     `json:"ItemNo"`
Service    ServiceCut `json:"service"`
Payment    string     `json:"payment"`
GST        string     `json:"gst"`
ServiceFee string     `json:"serviceFee"`
*/
func addServiceFeeCalculation(pdf *gofpdf.Fpdf, details PaymentTotals) {
	columns := []float64{25, 20, 15, 15, 25, 25, 25}
	tableData := [][]TableText{
		{
			TableText{text: "InvoiceNo", font: Arial12B},
			TableText{text: "ItemNo", font: Arial12B},
			TableText{text: "Code", font: Arial12B},
			TableText{text: "%", font: Arial12B},
			TableText{text: "Payment", font: Arial12B},
			TableText{text: "GST", font: Arial12B},
			TableText{text: "Service Fee", font: Arial12B}},
	}
	addTable(pdf, tableData, columns, 7)
	tableData = [][]TableText{}
	for _, payments := range details.PaymentDetails {
		tableData = append(tableData, []TableText{
			{text: payments.InvoiceNo},
			{text: payments.ItemNo},
			{text: payments.Service.Code},
			{text: payments.Service.Percentage},
			{text: payments.Payment},
			{text: payments.GST},
			{text: payments.ServiceFee},
		})
	}
	addTable(pdf, tableData, columns, 5)
}

func addAdjustments(pdf *gofpdf.Fpdf, adjustments []Adjustments) {
	if len(adjustments) == 0 {
		return
	}
	columns := []float64{75, 25, 25, 25}
	tableData := [][]TableText{
		{
			TableText{text: "Adjustments", font: Arial12B},
			TableText{text: "Charge", font: Arial12B},
			TableText{text: "GST", font: Arial12B},
			TableText{text: "Fee", font: Arial12B}},
	}
	addTable(pdf, tableData, columns, 7)
	tableData = [][]TableText{}
	for _, payments := range adjustments {
		amount := pct(payments.Amount)
		tableData = append(tableData, []TableText{
			{text: payments.Description},
			{text: amount},
			{text: "0"},
			{text: amount},
		})
	}
	addTable(pdf, tableData, columns, 5)
}

func addTotalCalc(pdf *gofpdf.Fpdf, serviceFeeTotal int, gst int) {
	columns := []float64{100, 25, 25}

	tableData := [][]TableText{
		{blankCell,
			TableText{text: "", font: Font{"Arial", "", 8}, border: "T"},
			TableText{text: "", font: Font{"Arial", "", 8}, border: "T"}},

		{TableText{text: "Totals:", font: Arial12B},
			TableText{text: fmt.Sprintf("%d", gst), font: Arial12B},
			TableText{text: pct(serviceFeeTotal), font: Arial12B}},
	}
	addTable(pdf, tableData, columns, 3)
}

func addAddress(pdf *gofpdf.Fpdf, address Address) {
	tableData := [][]TableText{
		{TableText{text: "From:", font: Arial12B}, TableText{text: address.Name}},
		{blankCell, TableText{text: address.StreetAddress}},
		{blankCell, TableText{text: address.City}},
		{TableText{text: "ABN", font: Arial12B}, TableText{text: address.ABN}},
	}
	addTable(pdf, tableData, []float64{40, 150}, 5)
}
func addAddressDate(pdf *gofpdf.Fpdf, address Address, date string) {
	tableData := [][]TableText{
		{TableText{text: "To:", font: Arial12B}, TableText{text: address.Name}, TableText{text: date, align: "R"}},
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
func addTotal(pdf *gofpdf.Fpdf, serviceFeeTotal int, gst int, adjustments int) {
	tableData := [][]TableText{
		{blankCell, TableText{text: "Service Fee (see calculation sheet)"}, TableText{text: pct(serviceFeeTotal), align: "R"}},
		{blankCell, TableText{text: "GST on Service Fee"}, TableText{text: pct(gst), align: "R"}}}
	if adjustments != 0 {
		tableData = append(tableData, []TableText{blankCell, TableText{text: "Adjustments", border: "B"},
			TableText{text: pct(adjustments), align: "R"},
		})
	}
	tableData = append(tableData, []TableText{blankCell, TableText{text: "Total", border: "B"},
		TableText{text: pct(serviceFeeTotal + gst), align: "R", border: "B"}})
	addTable(pdf, tableData, []float64{40, 150, 0}, 5)
}
func pct(v int) string {
	if v < 0 {
		return fmt.Sprintf("(%d.%02d)", -v/100, -v%100)
	}
	return fmt.Sprintf("%d.%02d", v/100, v%100)
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
