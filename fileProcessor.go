package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
)

type ServiceCut struct {
	Code       string `json:"code"`
	Percentage string `json:"percentage"`
}
type Address struct {
	Name          string `json:"name"`
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	ABN           string `json:"abn"`
	Email         string `json:"email"`
}

/*
json payload

	{
	    "FileContent": "your_file_content",
	    "CsvLineStart": 16,
	    "CompanyDetails": Address{...},
	    "CodeMap": [
	        {
	            "code1": ["123", "456"]
	        },
	        {
	            "code2": ["789", "012"]
	        }
	    ],
	    "PracMap":
	      {
	        "Doctor1": {"code1":"50", "code2":"20"},
	        "DOctor2": {"code1":"40", "code2":"30"}
	      }
	}
*/
type PaymentFile struct {
	FileContent    string                       `json:"fileContent"`
	CsvLineStart   int                          `json:"csvLineStart"`
	CompanyDetails Address                      `json:"companyDetails"`
	CodeMap        map[string][]string          `json:"codeMap"`
	PracMap        map[string]map[string]string `json:"pracMap"`
	PracDetails    map[string]Address           `json:"pracDetails"`
	AdjustMap      map[string][]Adjustments     `json:"adjustMap"` // maps providers to adjustments
}

type FileProcessingResponse struct {
	MissingProviders    map[string]string            `json:"missingProviders"`
	MissingItemNrs      map[string]string            `json:"missingItemNrs"`
	NoItemNrs           map[string]string            `json:"noItemNrs"`
	MissingServiceCodes map[string]map[string]string `json:"missingServiceCodes"`
	ChargeDetail        map[string]PaymentTotals     `json:"chargeDetail"`
}

type PaymentFileResponse struct {
	Provider     string     `json:"provider"`
	Patient      string     `json:"patient"`
	TransDate    string     `json:"transDate"`
	InvoiceNo    string     `json:"invoiceNo"`
	ItemNo       string     `json:"ItemNo"`
	Service      ServiceCut `json:"service"`
	Payment      string     `json:"payment"`
	GST          int        `json:"gst"`
	TotalPayment string     `json:"totalPayment"`
	ServiceFee   string     `json:"serviceFee"`
}

type Adjustments struct {
	Description string `json:"description"`
	Amount      int    `json:"amount"`
}

type PaymentTotals struct {
	Provider            string                   `json:"provider"`
	PaymentDetails      []PaymentFileResponse    `json:"paymentDetails"`
	PaymentTotalWithGST int                      `json:"paymentTotal"`
	PaymentTotalNoGST   int                      `json:"PaymentTotalWithGST"`
	ServiceCutTotal     int                      `json:"serviceCutTotal"`
	GSTTotal            int                      `json:"gstTotal"`
	AdjustmentTotal     int                      `json:"adjustmentTotal"`
	PdfFile             []byte                   `json:"invoice"`
	ServiceCodeSplit    map[string]ServiceTotals `json:"serviceCodeSplit"`
}

func (p *PaymentTotals) TotalPayments(gst int, payment int, serviceFee int) {
	if gst > 0 {
		p.PaymentTotalWithGST += payment
	} else {
		p.PaymentTotalNoGST += payment
	}
	p.ServiceCutTotal += serviceFee
	p.GSTTotal += gst
}

type ServiceTotals struct {
	ExGstFees   int    `json:"exgstfees"`
	ServiceFees int    `json:"serviceFees"`
	Rate        string `json:"rate"`
}

func (p *ServiceTotals) TotalServiceCodes(rate string, serviceFee int, payment int) {
	p.ExGstFees += payment
	p.ServiceFees += serviceFee
	p.Rate = rate
}

var (
	ErrAmount     = errors.New("invalid amount")
	ErrPercentage = errors.New("invalid percentage")
)

// IF:
// Providers present in report but not in provider database
// No option to create tax invoices
// Click button to update provider database
// IF:
// Items present in report but not in either item database
// No option to create tax invoices
// Click button to update item databases
// IF all clear calculate service fees [e.g. add service fee code (by item), percentage (by service
// fee code and provider), and service fee (by percentage x amount received {column O-N})
// ---> not doing this ---> appended to each line {5 to n-3} in Payments Export]
//
// Location, Provider, Billed, To, Patient Name, Invoice No., Service ID, Payment ID, Item No., Description,
// Status, Transaction, Date, Payment Method, Account Type, GST ($ incl GST), Payment ($ incl GST), Deposit ($ incl GST)
//
// Required: 0 Location, 1 Provider, 8 itemNum, 9 Description, 15 GST, 16 Payment, 17 Deposit
// Location,Provider,Billed To,Patient Name,Invoice No.,Service ID,Payment ID,Item No.,Description,Status,Transaction Date,Payment Method,Account Type,"GST
// ClinicName,Dr Phoebe Kho,Irrelevant,Patient Name,162307,174545,71756,80010,"Clinical psychologist consultation, >50 min, consulting rooms",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00

func processFileContent(content PaymentFile) (FileProcessingResponse, error) {

	//location := 0
	providerField := 1
	patientField := 3
	invoiceNumField := 4
	itemNumField := 7
	descriptionField := 8
	transDateField := 10
	GSTField := 13
	paymentField := 14

	fileRes := FileProcessingResponse{}
	fileRes.MissingProviders = map[string]string{}
	fileRes.MissingItemNrs = map[string]string{}
	fileRes.NoItemNrs = map[string]string{}
	fileRes.MissingServiceCodes = make(map[string]map[string]string)

	providerTotalsMap := map[string]PaymentTotals{}
	providerWithErrors := map[string]string{}

	reader := csv.NewReader(strings.NewReader(content.FileContent))
	records, err := reader.ReadAll()
	if err != nil {
		return fileRes, processError(fmt.Sprintf("Reading csv file failed with error: %v", err))
	}

	lineNum, records, reportPeriod, companyName, err := getHeaderDetails(records)
	if err != nil {
		logError.Printf("Reading csv file failed with error: %v", err)
	}

	itemMap := createItemMap(content.CodeMap)
	providerMap := createProviderMap(content.PracMap)

	for _, record := range records {
		lineNum++
		//
		// Skip blank lines
		//
		provider := strings.TrimSpace(record[providerField])
		if provider == "" {
			continue
		}
		itemNr := strings.TrimSpace(record[itemNumField])
		itemDesc := strings.TrimSpace(record[descriptionField])

		providerServiceCodes, ok := providerMap[standardString(provider)]
		if !ok {
			fileRes.MissingProviders[provider] = standardString(provider)
			continue
		}
		//
		// if there is no item number we use the description to map to the service code
		//
		if itemNr == "" {
			itemNr = itemDesc
		}
		serviceCode, itemFound := itemMap[itemNr]
		if !itemFound {
			if itemNr == "" {
				fileRes.NoItemNrs[itemNr] = itemNr
			} else {
				fileRes.MissingItemNrs[itemNr] = itemNr
			}
			providerWithErrors[provider] = provider
		}
		//
		// Once we have a service code, get the percentage per provider for that service code
		//
		serviceCut, ok := providerServiceCodes[serviceCode]
		if itemFound && !ok {
			errStr := fmt.Sprintf("provider: %v in line: %v has no service cut assigned for service code: %v",
				provider, lineNum, serviceCode)
			if fileRes.MissingServiceCodes[provider] == nil {
				fileRes.MissingServiceCodes[provider] = make(map[string]string)
			}
			fileRes.MissingServiceCodes[provider][serviceCode] = errStr
			providerWithErrors[provider] = provider
		}
		//
		// If there was any errors detected for that provider we will not produce an invoice
		//
		if _, exists := providerWithErrors[provider]; exists {
			continue
		}
		// Make the calculations for the service fee and exGst
		exGst, feeCents, paymentCents, gstCents, err := calcPayment(record[paymentField], record[GSTField], serviceCut)
		if err != nil {
			if errors.Is(err, ErrAmount) {
				return fileRes, processError(fmt.Sprintf("provider: %v in line: %v value: %v. Cause: %v",
					provider, lineNum, record[paymentField], err.Error()))
			} else if errors.Is(err, ErrPercentage) {
				return fileRes, processError(fmt.Sprintf("provider: %v in line: %v value: %v. Cause: %v",
					provider, lineNum, serviceCut, err.Error()))
			} else {
				return fileRes, processError(fmt.Sprintf("provider: %v in line: %v with amount: %v and percentage %v failed due to unknown error: %v",
					provider, lineNum, record[paymentField], serviceCut, err.Error()))
			}
		}
		// Create the maps storing totals and individual payments
		providerPaymentMap, exists := providerTotalsMap[provider]
		if !exists {
			providerPaymentMap = PaymentTotals{Provider: provider}
			providerPaymentMap.ServiceCodeSplit = make(map[string]ServiceTotals)
		}
		serviceTotals, exists := providerPaymentMap.ServiceCodeSplit[serviceCode]
		if !exists {
			serviceTotals = ServiceTotals{}
		}
		// Add the totals and the payment details
		providerPaymentMap.TotalPayments(gstCents, paymentCents, feeCents)
		serviceTotals.TotalServiceCodes(serviceCut, exGst, feeCents)

		result := PaymentFileResponse{
			Provider:  provider,
			Patient:   record[patientField],
			TransDate: record[transDateField],
			InvoiceNo: record[invoiceNumField],
			ItemNo:    itemNr,
			Service: ServiceCut{
				Code:       serviceCode,
				Percentage: serviceCut,
			},
			Payment:      record[paymentField],
			GST:          gstCents,
			TotalPayment: cents2DStr(paymentCents),
			ServiceFee:   cents2DStr(feeCents),
		}
		providerPaymentMap.PaymentDetails = append(providerPaymentMap.PaymentDetails, result)
		providerPaymentMap.ServiceCodeSplit[serviceCode] = serviceTotals
		providerTotalsMap[provider] = providerPaymentMap
	}
	//
	// Create PDFs, but only if that provider had no errors
	// If there are adjustments for that provider, add them to the PDF
	//
	for provider, details := range providerTotalsMap {
		if _, exists := providerWithErrors[provider]; !exists {
			if content.AdjustMap[provider] != nil {
				details.AdjustmentTotal = 0
				for _, adj := range content.AdjustMap[provider] {
					details.AdjustmentTotal += adj.Amount
				}
			}
			pdfBytes, err := makePdf(reportPeriod, companyName, provider, details, content.AdjustMap[provider],
				content.CompanyDetails, content.PracDetails[provider])

			if err != nil {
				logError.Printf("Error creating PDF for provider: %v. Cause: %v", provider, err)
			}
			details.PdfFile = pdfBytes
			details.Provider = provider
			providerTotalsMap[provider] = details
		}
	}
	fileRes.ChargeDetail = providerTotalsMap
	return fileRes, nil
}

func processError(err string) error {
	logError.Print(err)
	return fmt.Errorf("%s", err)

}
func compareNames(name1, name2 string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(name1, " ", "")),
		strings.ToLower(strings.ReplaceAll(name2, " ", "")))
}
func standardString(s string) string {
	ns := strings.Join(strings.Fields(s), " ")
	return strings.ToLower(ns)
}

func getHeaderDetails(records [][]string) (int, [][]string, string, string, error) {
	reportPeriod := ""
	companyName := ""
	for i, record := range records {
		trimmedRecord := strings.ToLower(strings.TrimSpace(record[0]))
		if strings.Contains(trimmedRecord, "report period:") {
			startIndex := strings.Index(trimmedRecord, "report period:") + len("report period:")
			reportPeriod = strings.TrimSpace(trimmedRecord[startIndex:])
			companyName = strings.TrimSpace(record[15])
		}

		if strings.Contains(trimmedRecord, "location") {
			return i + 1, records[i+1:], reportPeriod, companyName, nil
		}
	}
	return 0, records, reportPeriod, companyName, fmt.Errorf("no header found")
}

// input: CodeMap: map[string][]string{"code1": {"123", "456"}}, {"code2": {"789", "012"}}
// output: map[string]string{"123": "code1", "456": "code1", "789": "code2", "012": "code2"}
func createItemMap(itemMap map[string][]string) map[string]string {
	result := make(map[string]string)
	for serviceCode, items := range itemMap {
		for _, itemNr := range items {
			result[itemNr] = serviceCode
		}
	}
	return result
}
func createProviderMap(pracMap map[string]map[string]string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for provider, serviceCodes := range pracMap {
		result[standardString(provider)] = serviceCodes
	}
	return result
}
