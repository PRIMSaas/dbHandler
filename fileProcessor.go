package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ServiceCut struct {
	Code       string `json:"code"`
	Percentage string `json:"percentage"`
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
}

type FileError struct {
	ErrorMsg string `json:"msg"`
}

type PaymentFileResponse struct {
	Provider     string     `json:"provider"`
	Patient      string     `json:"patient"`
	InvoiceNo    string     `json:"invoiceNo"`
	ItemNo       string     `json:"ItemNo"`
	Service      ServiceCut `json:"service"`
	Payment      string     `json:"payment"`
	Billed       string     `json:"billed"`
	ProcessError FileError  `json:"error"`
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

func processFileContent(content PaymentFile) ([]PaymentFileResponse, error) {

	//location := 0
	provider := 1
	patient := 3
	invoiceNum := 4
	itemNum := 7
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
		return processError(fmt.Sprintf("Reading csv file failed with error: %v", err), res), nil
	}
	itemMap := createItemMap(content.CodeMap)

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
		providerServiceCodes, ok := content.PracMap[prov]
		if !ok {
			res = processError(fmt.Sprintf("provider: %v in line: %v has not bee configured", record[provider], lineNum), res)
			continue
		}
		serviceCode, ok := itemMap[strings.TrimSpace(record[itemNum])]
		if !ok {
			res = processError(fmt.Sprintf("item number: %v in line: %v not associated with a Service Code", record[itemNum], lineNum), res)
			continue
		}
		serviceCut, ok := providerServiceCodes[serviceCode]
		if !ok {
			res = processError(fmt.Sprintf("provider: %v in line: %v has no service cut assigned for service code: %v",
				record[provider], lineNum, serviceCode), res)
			continue
		}
		billed, err := calcPayment(record[payment], serviceCut)
		if err != nil {
			if errors.Is(err, ErrAmount) {
				res = processError(fmt.Sprintf("provider: %v in line: %v value: %v. Cause: %v",
					record[provider], lineNum, record[payment], err.Error()), res)
				continue
			} else if errors.Is(err, ErrPercentage) {
				res = processError(fmt.Sprintf("provider: %v in line: %v value: %v. Cause: %v",
					record[provider], lineNum, serviceCut, err.Error()), res)
				continue
			}
			res = processError(fmt.Sprintf("provider: %v in line: %v with amount: %v and parcentage %v failed due to unknown error: %v",
				record[provider], lineNum, record[payment], serviceCut, err.Error()), res)
			continue
		}
		result := PaymentFileResponse{
			Provider:  record[provider],
			Patient:   record[patient],
			InvoiceNo: record[invoiceNum],
			ItemNo:    record[itemNum],
			Service: ServiceCut{
				Code:       serviceCode,
				Percentage: serviceCut,
			},
			Payment: record[payment],
			Billed:  billed,
		}
		res = append(res, result)
	}
	return res, nil
}

func processError(err string, resp []PaymentFileResponse) []PaymentFileResponse {
	res := PaymentFileResponse{}
	res.ProcessError.ErrorMsg = err
	logError.Printf(err)
	return append(resp, res)

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

func calcPayment(payment string, percentage string) (string, error) {
	p, err := convertToFloat(payment)
	if err != nil {
		return "", fmt.Errorf("%w %w", ErrAmount, err)
	}
	perc, err := convertPercToFloat(percentage)
	if err != nil {
		return "", fmt.Errorf("%w %w", ErrPercentage, err)
	}
	partPay := p
	if perc <= 0 {
		return "", fmt.Errorf("%w %w", ErrPercentage, fmt.Errorf("percentage value must be greater than 0"))
	}
	partPay = math.Round(p*perc) / 100
	return fmt.Sprintf("%.2f", partPay), nil
}

func convertToFloat(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("field is blank")
	}
	factor := 1
	if value[0] == '(' {
		factor = -1
		value = strings.ReplaceAll(value, "(", "")
		value = strings.ReplaceAll(value, ")", "")
	}
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting value")
	}
	return num * float64(factor), nil
}

func convertPercToFloat(value string) (float64, error) {
	value = strings.ReplaceAll(value, "%", "")
	return convertToFloat(value)
}
