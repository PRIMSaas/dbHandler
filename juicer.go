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

func makePdf(reportPeriod string, companyName string, provider string, details PaymentTotals, adjustments []Adjustments,
	companyDetails Address, providerAddr Address) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(10, 10, 30)

	pdf.SetTitle("TAX INVOICE", false)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(10, 20, "TAX INVOICE")
	pdf.ImageOptions("logo.jpg", 150, 20, 35, 35, false, gofpdf.ImageOptions{ImageType: "JPEG", ReadDpi: true}, 0, "")
	pdf.SetXY(10, 29)
	addAddress(pdf, companyName, companyDetails)
	pdf.Ln(10)
	addAddressDate(pdf, providerAddr, time.Now().Format("02-01-06"))
	pdf.Ln(10)
	addInvoiceDetails(pdf, provider, providerAddr.Email, reportPeriod, "JG20240505")
	pdf.Ln(10)
	addTotal(pdf, details.ServiceCutTotal, details.AdjustmentTotal)
	pdf.Ln(10)
	addAdjustments(pdf, adjustments, details.AdjustmentTotal)
	pdf.Ln(10)
	addPaymentSummary(pdf, details.PaymentTotalWithGST, details.PaymentTotalNoGST)

	pdf.AddPage()
	pdf.SetMargins(10, 10, 30)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(10, 20, "Service Fee Calculations")
	pdf.SetXY(10, 29)
	addServiceFeeCalculation(pdf, details)
	pdf.Ln(3)
	addTotalCalc(pdf, details)

	pdf.AddPage()
	pdf.SetMargins(20, 10, 30)
	//drawGrid(pdf)
	pdf.SetFont("Arial", "B", 16)
	pdf.Text(20, 20, "Service Fee Breakdown")
	pdf.SetXY(20, 29)
	addServiceFeeBreakdown(pdf, details.ServiceCodeSplit)

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
	type ServiceTotals struct {
		ServiceCode string `json:"serviceCode"`
		ExGstFees   int    `json:"exgstfees"`
		ServiceFees int    `json:"serviceFees"`
		Rate		int    `json:"rate"`
	}
*/
func addServiceFeeBreakdown(pdf *gofpdf.Fpdf, serviceTotals map[string]ServiceTotals) {
	columns := []float64{50, 30, 30, 50}
	tableData := [][]TableText{
		{blankCell,
			{text: "Payment ex GST", align: "R", font: Arial12B},
			{text: "Rate %", align: "R", font: Arial12B},
			{text: "Service Fees ex GST", align: "R", font: Arial12B}},
	}
	addTable(pdf, tableData, columns, 4)
	tableData = [][]TableText{}
	serviceFeeTotal := 0
	exGstTotal := 0
	for code, service := range serviceTotals {
		tableData = append(tableData, []TableText{
			{text: code},
			{text: cents2DStr(service.ExGstFees), align: "R"},
			{text: service.Rate, align: "R"},
			{text: cents2DStr(service.ServiceFees), align: "R"},
		})
		serviceFeeTotal += service.ServiceFees
		exGstTotal += service.ExGstFees
	}
	addTable(pdf, tableData, columns, 5)

	tableData = [][]TableText{{blankCell, blankCell, blankCell}}
	addTable(pdf, tableData, columns, 1)

	tableData = [][]TableText{{{text: "Total"}, {text: cents2DStr(serviceFeeTotal), align: "R", border: "T"},
		blankCell, {text: cents2DStr(exGstTotal), align: "R", border: "T"}}}
	addTable(pdf, tableData, columns, 7)
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
	columns := []float64{25, 30, 17, 30, 30, 17, 10, 30}
	tableData := [][]TableText{
		{
			TableText{text: "Date", font: Arial12B},
			TableText{text: "Patient", font: Arial12B},
			TableText{text: "ItemNo", font: Arial12B},
			TableText{text: "Payment", align: "R", font: Arial12B},
			TableText{text: "with GST", align: "R", font: Arial12B},
			TableText{text: "Code", align: "R", font: Arial12B},
			TableText{text: "%", align: "R", font: Arial12B},
			TableText{text: "Service Fee", align: "R", font: Arial12B}},
	}
	addTable(pdf, tableData, columns, 7)
	tableData = [][]TableText{}
	for _, payments := range details.PaymentDetails {
		itemNo := fmt.Sprintf("%.8s", payments.ItemNo)
		name := fmt.Sprintf("%.12s", payments.Patient)
		lineData := []TableText{{text: payments.TransDate}, {text: name}, {text: itemNo}}
		if payments.GST == 0 {
			lineData = append(lineData, TableText{text: payments.Payment, align: "R"}, blankCell)
		} else {
			lineData = append(lineData, blankCell, TableText{text: payments.Payment, align: "R"})
		}
		lineData = append(lineData, TableText{text: payments.Service.Code, align: "R"},
			TableText{text: payments.Service.Percentage, align: "R"},
			TableText{text: payments.ServiceFee, align: "R"})
		tableData = append(tableData, lineData)
	}
	addTable(pdf, tableData, columns, 5)
}

func addTotalCalc(pdf *gofpdf.Fpdf, details PaymentTotals) {
	columns := []float64{25, 30, 17, 30, 30, 17, 10, 30}

	tableData := [][]TableText{
		{TableText{text: "Totals:", font: Arial12B},
			blankCell,
			blankCell,
			TableText{text: cents2DStr(details.PaymentTotalNoGST), align: "R", font: Arial12B, border: "T"},
			TableText{text: cents2DStr(details.PaymentTotalWithGST), align: "R", font: Arial12B, border: "T"},
			blankCell,
			blankCell,
			TableText{text: cents2DStr(details.ServiceCutTotal), align: "R", font: Arial12B, border: "T"}},
	}
	addTable(pdf, tableData, columns, 7)
}

func addAddress(pdf *gofpdf.Fpdf, companyName string, address Address) {
	if companyName == "" {
		companyName = address.Name
	}
	tableData := [][]TableText{
		{TableText{text: "From:", font: Arial12B}, TableText{text: companyName}},
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
func addInvoiceDetails(pdf *gofpdf.Fpdf, prac string, email string, invoicePeriod string, invoiceNo string) {
	tableData := [][]TableText{
		{TableText{text: "Practitioner:", font: Arial12B}, TableText{text: prac}, TableText{text: "Invoice Number", font: Arial12B, align: "R"}},
		{TableText{text: "Period:", font: Arial12B}, TableText{text: invoicePeriod}, TableText{text: invoiceNo, align: "R"}},
		{TableText{text: "Email", font: Arial12B}, TableText{text: email}, blankCell},
	}
	addTable(pdf, tableData, []float64{40, 150, 0}, 5)
}

func addTotal(pdf *gofpdf.Fpdf, serviceFeeTotal int, adjustments int) {
	tableData := [][]TableText{
		{blankCell, TableText{text: "Service Fee (see calculation sheet)"}, TableText{text: cents2DStr(serviceFeeTotal), align: "R"}}}
	if adjustments != 0 {
		tableData = append(tableData, []TableText{blankCell, {text: "Adjustments"},
			{text: cents2DStr(adjustments), align: "R"},
		})
	}
	tableData = append(tableData, []TableText{blankCell, {text: "Subtotal"},
		{text: cents2DStr(serviceFeeTotal + adjustments), align: "R", border: "T"},
	})

	gst := calcGST(serviceFeeTotal, adjustments)
	tableData = append(tableData, []TableText{blankCell, {text: "GST"},
		{text: cents2DStr(gst), align: "R", border: "B"},
	})

	tableData = append(tableData, []TableText{blankCell, {text: "Total"},
		{text: cents2DStr(serviceFeeTotal + adjustments + gst), align: "R", border: "B"}})
	addTable(pdf, tableData, []float64{40, 110, 0}, 5)
}

func addAdjustments(pdf *gofpdf.Fpdf, adjustments []Adjustments, total int) {

	if len(adjustments) == 0 {
		return
	}
	tableData := [][]TableText{
		{TableText{text: "Adjustments"}, blankCell, blankCell}}
	for _, payments := range adjustments {
		amount := cents2DStr(payments.Amount)
		tableData = append(tableData, []TableText{
			blankCell,
			{text: payments.Description},
			{text: amount, align: "R"},
		})
	}
	tableData = append(tableData, []TableText{blankCell, {text: "Total"},
		{text: cents2DStr(total), align: "R", border: "T"}})
	addTable(pdf, tableData, []float64{40, 110, 0}, 5)
}

func addPaymentSummary(pdf *gofpdf.Fpdf, paymentTotalWithGST int, paymentTotalNoGST int) {
	tableData := [][]TableText{
		{TableText{text: "Tax Statement"}, blankCell, blankCell},
		{blankCell, TableText{text: "Services without GST"}, TableText{text: cents2DStr(paymentTotalNoGST), align: "R"}},
		{blankCell, TableText{text: "Services with GST"}, TableText{text: cents2DStr(paymentTotalWithGST), align: "R"}},
		{blankCell, TableText{text: "Total"}, TableText{text: cents2DStr(paymentTotalWithGST + paymentTotalNoGST), align: "R", border: "T"}}}
	addTable(pdf, tableData, []float64{40, 110, 0}, 5)
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
