package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Service struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type ServiceCut struct {
	Scode      Service
	Percentage string
}
type Provider struct {
	Provider    string `json:"providerName"`
	ServiceCuts map[string]ServiceCut
	Items       map[int]ServiceCut
	Employer    string `json:"employer"`
	ProviderId  string `json:"providerId"`
	Feedback    string `json:"feedback"`
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
	CsvLineStart int                          `json:"noneCsvLines"`
	CompanyName  string                       `json:"companyName"`
	CodeMap      []map[string][]string        `json:"codeMap"`
	PracMap      map[string]map[string]string `json:"pracMap"`
}

type PaymentFileResponse struct {
	Provider  string     `json:"provider"`
	Patient   string     `json:"patient"`
	InvoiceNo string     `json:"invoiceNo"`
	ItemNo    string     `json:"itemNo"`
	Service   ServiceCut `json:"service"`
	Payment   string     `json:"payment"`
	Billed    string     `json:"billed"`
}

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
// Required: 0 Location, 1 Provider, 8 ItemNo, 9 Description, 15 GST, 16 Payment, 17 Deposit
// Location,Provider,Billed To,Patient Name,Invoice No.,Service ID,Payment ID,Item No.,Description,Status,Transaction Date,Payment Method,Account Type,"GST
// ClinicName,Dr Phoebe Kho,Irrelevant,Patient Name,162307,174545,71756,80010,"Clinical psychologist consultation, >50 min, consulting rooms",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00

func processFileContent(content PaymentFile) ([]PaymentFileResponse, error) {

	location := 0
	provider := 1
	patient := 3
	invoiceNr := 4
	itemNo := 7
	//description := 8
	//GST := 13
	payment := 14
	//deposit := 15

	res := []PaymentFileResponse{}
	s, err := truncateCsv(content.FileContent, content.CsvLineStart)
	if err != nil {
		return res, err
	}

	reader := csv.NewReader(strings.NewReader(s))
	records, err := reader.ReadAll()
	if err != nil {
		s := fmt.Sprintf("Reading csv file failed with error: %v", err)
		logError.Printf(s)
		return res, fmt.Errorf(s)
	}
	itemMap := createItemMap(content.CodeMap)

	for i, record := range records {
		// Check if the company name is in the record
		// is the same as in the request, if not skip
		if !compareNames(record[location], content.CompanyName) {
			continue
		}
		serviceCode := ""
		ok := false
		if serviceCode, ok = itemMap[record[itemNo]]; !ok {
			return res,
				fmt.Errorf("item number: %v in line: %v is not in the item database", record[itemNo], i+content.CsvLineStart)
		}
		providerServiceCodes := map[string]string{}
		if providerServiceCodes, ok = content.PracMap[record[provider]]; !ok {
			return res,
				fmt.Errorf("provider: %v in line: %v has no service codes assigned", record[provider], i+content.CsvLineStart)
		}
		serviceCut := ""
		if serviceCut, ok = providerServiceCodes[serviceCode]; !ok {
			return res,
				fmt.Errorf("provider: %v in line: %v has no service cut assigned for service code: %v",
					record[provider], i+content.CsvLineStart, serviceCode)
		}
		billed, err := calcPayment(record[payment], serviceCut)
		if err != nil {
			return res, fmt.Errorf("error converting billed amount in line: %v: %v", i+content.CsvLineStart, err)
		}
		result := PaymentFileResponse{
			Provider:  record[provider],
			Patient:   record[patient],
			InvoiceNo: record[invoiceNr],
			ItemNo:    record[itemNo],
			Service: ServiceCut{
				Scode:      Service{Code: serviceCode, Description: ""},
				Percentage: serviceCut,
			},
			Payment: record[payment],
			Billed:  billed,
		}
		res = append(res, result)
	}
	return res, nil
}

func compareNames(name1, name2 string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(name1, " ", "")),
		strings.ToLower(strings.ReplaceAll(name2, " ", "")))
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

// input: CodeMap: []map[string][]string{{"code1": {"123", "456"}}, {"code2": {"789", "012"}}}
// output: map[string]string{"123": "code1", "456": "code1", "789": "code2", "012": "code2"}
func createItemMap(itemMap []map[string][]string) map[string]string {
	result := make(map[string]string)
	for _, mappings := range itemMap {
		for serviceCode, items := range mappings {
			for _, itemNr := range items {
				result[itemNr] = serviceCode
			}
		}
	}
	return result
}

func calcPayment(payment string, percentage string) (string, error) {
	p, err := convertToFloat(payment)
	if err != nil {
		return "", err
	}
	perc, err := convertPercToFloat(percentage)
	if err != nil {
		return "", err
	}
	partPay := p
	if perc <= 0 {
		return "", fmt.Errorf("error percantage is: %v", perc)
	}
	partPay = math.Round(p*perc) / 100
	return fmt.Sprintf("%.2f", partPay), nil
}

func convertToFloat(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	factor := 1
	if value[0] == '(' {
		factor = -1
		value = strings.ReplaceAll(value, "(", "")
		value = strings.ReplaceAll(value, ")", "")
	}
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting payment value: %v", err)
	}
	return num * float64(factor), nil
}

func convertPercToFloat(value string) (float64, error) {
	value = strings.ReplaceAll(value, "%", "")
	return convertToFloat(value)
}

func convertPayment(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, nil
	}
	factor := 1
	if value[0] == '(' {
		factor = -1
		value = strings.ReplaceAll(value, "(", "")
		value = strings.ReplaceAll(value, ")", "")
	}
	i := strings.Index(value, ".")
	decimal := 0
	if i > 0 {
		decimal = len(value) - i - 1
	}
	value = strings.ReplaceAll(value, ".", "")
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, 0, fmt.Errorf("error converting payment value: %v", err)
	}
	return num * factor, decimal, nil
}

func convertPerc(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	value = strings.ReplaceAll(value, "%", "")

	val, fact, err := convertPayment(value)
	if err != nil {
		return 0, fmt.Errorf("error converting percentage value: %v", err)
	}
	if val < 0 {
		return 0, fmt.Errorf("percentage value cannot be negative")
	}
	if val == 0 {
		return 0, nil
	}
	factor := 100
	if fact > 0 {
		factor = int(math.Pow10(fact)) * 100
	}
	return float64(val / factor), nil
}
