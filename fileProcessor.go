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
	CompanyName   string `json:"companyName"`
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	ABN           string `json:"abn"`
}

/*
json payload

	{
	    "FileContent": "your_file_content",
	    "CsvLineStart": 16,
	    "CompanyName": "Vermont Medical Clinic",
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
	FileContent  string                       `json:"fileContent"`
	CsvLineStart int                          `json:"csvLineStart"`
	CompanyName  string                       `json:"companyName"`
	CodeMap      map[string][]string          `json:"codeMap"`
	PracMap      map[string]map[string]string `json:"pracMap"`
	DescMap      map[string]string            `json:"descMap"` // maps description to service code
}

type FileProcessingResponse struct {
	MissingProviders    map[string]string            `json:"missingProviders"`
	MissingItemNrs      map[string]string            `json:"missingItemNrs"`
	NoItemNrs           map[string]string            `json:"noItemNrs"`
	MissingServiceCodes map[string]map[string]string `json:"missingServiceCodes"`
	ChargeDetail        map[string]PaymentTotals     `json:"chargeDetail"`
	ErrorMsg            string                       `json:"errorMsg"`
}

type PaymentFileResponse struct {
	Provider         string     `json:"provider"`
	Patient          string     `json:"patient"`
	InvoiceNo        string     `json:"invoiceNo"`
	ItemNo           string     `json:"ItemNo"`
	Service          ServiceCut `json:"service"`
	Payment          string     `json:"payment"`
	GST              string     `json:"gst"`
	ServiceFee       string     `json:"serviceFee"`
	ProviderErrorMsg string     `json:"msg"`
}

type PaymentTotals struct {
	Provider        string                `json:"provider"`
	PaymentDetails  []PaymentFileResponse `json:"paymentDetails"`
	PaymentTotal    int                   `json:"paymentTotal"`
	ServiceCutTotal int                   `json:"serviceCutTotal"`
	GSTTotal        int                   `json:"gstTotal"`
	PdfFile         []byte                `json:"invoice"`
}

func (p *PaymentTotals) AddPaymentDetails(details PaymentFileResponse, serviceFee int) {
	p.PaymentDetails = append(p.PaymentDetails, details)
	payment, _ := dollarStringToCents(details.Payment)
	gst, _ := dollarStringToCents(details.GST)
	p.PaymentTotal += payment
	p.ServiceCutTotal += serviceFee
	p.GSTTotal += gst
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
	provider := 1
	patient := 3
	invoiceNum := 4
	itemNum := 7
	description := 8
	GST := 13
	payment := 14
	//deposit := 15

	fileRes := FileProcessingResponse{}
	fileRes.MissingProviders = map[string]string{}
	fileRes.MissingItemNrs = map[string]string{}
	fileRes.NoItemNrs = map[string]string{}
	fileRes.MissingServiceCodes = make(map[string]map[string]string)

	providerTotalsMap := map[string]PaymentTotals{}

	result := PaymentFileResponse{}
	s, err := truncateCsv(content.FileContent, content.CsvLineStart)
	if err != nil {
		return fileRes, err
	}
	reader := csv.NewReader(strings.NewReader(s))
	records, err := reader.ReadAll()
	if err != nil {
		fileRes.ErrorMsg = fmt.Sprintf("Reading csv file failed with error: %v", err)
		return fileRes, nil
	}
	itemMap := createItemMap(content.CodeMap)
	providerMap := createProviderMap(content.PracMap)

	lineNum := content.CsvLineStart
	for _, record := range records {
		lineNum++
		//
		// Skip blank lines
		//
		prov := strings.TrimSpace(record[provider])
		if prov == "" {
			continue
		}
		// Check if the company name is in the record
		// is the same as in the request, if not skip
		//	if !compareNames(record[location], content.CompanyName) {
		//		continue
		//	}
		// providerServiceCodes := map[string]string{}
		providerServiceCodes, ok := providerMap[standardString(prov)]
		if !ok {
			fileRes.MissingProviders[prov] = standardString(prov)
			continue
		}
		//
		// if there is no item number we use the description to map to the service code
		//
		serviceCode := ""
		if strings.TrimSpace(record[itemNum]) == "" {
			serviceCode, ok = itemMap[strings.TrimSpace(record[description])]
			if !ok {
				desc := strings.TrimSpace(record[description])
				fileRes.NoItemNrs[desc] = desc
				continue
			}
		} else {
			serviceCode, ok = itemMap[strings.TrimSpace(record[itemNum])]
			if !ok {
				fileRes.MissingItemNrs[record[itemNum]] = record[itemNum]
				continue
			}
		}
		//
		// Once we have a service code, get the percentage per provider for that service code
		//
		serviceCut, ok := providerServiceCodes[serviceCode]
		if !ok {
			errStr := fmt.Sprintf("provider: %v in line: %v has no service cut assigned for service code: %v",
				record[provider], lineNum, serviceCode)
			if fileRes.MissingServiceCodes[record[provider]] == nil {
				fileRes.MissingServiceCodes[record[provider]] = make(map[string]string)
			}
			fileRes.MissingServiceCodes[record[provider]][serviceCode] = errStr
			continue
		}
		//
		// Now we are ready to perform the calculations
		//
		billed, err := calcPayment(record[payment], serviceCut)
		if err != nil {
			if errors.Is(err, ErrAmount) {
				result = processError(fmt.Sprintf("provider: %v in line: %v value: %v. Cause: %v",
					record[provider], lineNum, record[payment], err.Error()))
			} else if errors.Is(err, ErrPercentage) {
				result = processError(fmt.Sprintf("provider: %v in line: %v value: %v. Cause: %v",
					record[provider], lineNum, serviceCut, err.Error()))
			} else {
				result = processError(fmt.Sprintf("provider: %v in line: %v with amount: %v and parcentage %v failed due to unknown error: %v",
					record[provider], lineNum, record[payment], serviceCut, err.Error()))
			}
		} else {
			//
			// Prepare the response to be sent back
			//
			result = PaymentFileResponse{
				Provider:  record[provider],
				Patient:   record[patient],
				InvoiceNo: record[invoiceNum],
				ItemNo:    record[itemNum],
				Service: ServiceCut{
					Code:       serviceCode,
					Percentage: serviceCut,
				},
				Payment:    record[payment],
				GST:        record[GST],
				ServiceFee: pct(billed),
			}
		}
		plist, exists := providerTotalsMap[record[provider]]
		if !exists {
			plist = PaymentTotals{Provider: record[provider]}
		}
		plist.AddPaymentDetails(result, billed)
		providerTotalsMap[record[provider]] = plist
	}
	//
	// Create PDFs
	//
	for provider, details := range providerTotalsMap {
		file, err := makePdf(provider, details)
		if err != nil {
			logError.Printf("Error creating PDF for provider: %v. Cause: %v", provider, err)
		}
		details.PdfFile = file
		details.Provider = provider
		//break;
	}
	fileRes.ChargeDetail = providerTotalsMap
	return fileRes, nil
}

func processError(err string) PaymentFileResponse {
	res := PaymentFileResponse{}
	res.ProviderErrorMsg = err
	logError.Printf(err)
	return res

}
func compareNames(name1, name2 string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(name1, " ", "")),
		strings.ToLower(strings.ReplaceAll(name2, " ", "")))
}
func standardString(s string) string {
	ns := strings.Join(strings.Fields(s), " ")
	return strings.ToLower(ns)
}

func truncateCsv(content string, noneCsvLines int) (string, error) {
	index := 0
	for i := 0; i < noneCsvLines-1; i++ {
		nextNewline := strings.Index(content[index:], "\n")
		if nextNewline == -1 {
			return "", fmt.Errorf("file content has less than %v lines", noneCsvLines)
		}
		index += nextNewline + 1
	}

	// Slice the string from the noneCsvLines'th newline character
	return content[index:], nil
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
