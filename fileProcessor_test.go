package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
	type PaymentFile struct {
		FileContent  string   `json:"fileContent"`
		CsvLineStart int      `json:"noneCsvLines"`
		CompanyName  string   `json:"companyName"`
		CodeMap []map[string][]string `json:"codeMap"`
		PracMap []map[string][]map[string]string `json:"pracMap"`
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
*/
func ReadTestFile(fileName string) (string, error) {
	b, err := os.ReadFile("PaymentsExport.csv")
	if err != nil {
		return "", fmt.Errorf("Error while reading the file: %v", err)
	}
	return string(b), nil
}

func TestProcessFile(t *testing.T) {
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	dr2 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,91182,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"

	paymentFile := PaymentFile{
		FileContent:  dr1 + "\n" + dr2,
		CsvLineStart: 0,
		CompanyName:  "A Practice",
		CodeMap:      []map[string][]string{{"code1": {"80010", "456"}}, {"code2": {"789", "012"}}},
		PracMap:      map[string]map[string]string{"Dr Aha": {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
	}
	res, err := processFileContent(paymentFile)
	if err != nil {
		t.Errorf("Error while processing the file: %v", err)
	}
	require.Equal(t, "Dr Aha", res[0].Provider)
	require.Equal(t, "Sick Patient", res[0].Patient)
	require.Equal(t, "80010", res[0].ItemNo)
	require.Equal(t, "(224.50)", res[0].Payment)
	require.Equal(t, "-67.35", res[0].Billed)
	require.NotEmpty(t, res)
}

func TestConvertPayment(t *testing.T) {
	tests := []struct {
		input  string
		output float64
	}{
		{"123.45", 123.45},
		{"123", 123},
		{"123.4", 123.4},
		{"123.456", 123.456},
		{"(123.45)", -123.45},
		{"(123", -123},
		{"(123.4)", -123.4},
		{"(123.456)", -123.456},
	}
	for _, test := range tests {
		res, err := convertToFloat(test.input)
		require.NoError(t, err)
		require.Equal(t, test.output, res)
	}
	_, _, err := convertPayment("test.input")
	require.Error(t, err)
	_, _, err = convertPayment("123)")
	require.Error(t, err)
	_, _, err = convertPayment("1 23)")
	require.Error(t, err)
}

func TestConvertPecentage(t *testing.T) {
	tests := []struct {
		input  string
		output float64
	}{
		{"10%", 10},
		{"80", 80},
		{"", 0},
		{"5.0%", 5},
	}
	for _, test := range tests {
		res, err := convertPercToFloat(test.input)
		require.NoError(t, err)
		require.Equal(t, test.output, res)
	}
	_, err := convertPerc("5 0")
	require.Error(t, err)
}

func TestCalcPayment(t *testing.T) {
	tests := []struct {
		payment    string
		percentage string
		output     string
	}{
		{"123.45", "10%", "12.35"},
		{"123", "80", "98.40"},
		{"123.4", "", ""},
		{"123.456", "5.0%", "6.17"},
		{"(123.45)", "10%", "-12.35"},
		{"(123", "80", "-98.40"},
		{"(123.4)", "", ""},
		{"(123.456)", "5.0%", "-6.17"},
	}
	for _, test := range tests {
		res, err := calcPayment(test.payment, test.percentage)
		if test.output == "" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.output, res)
		}
	}
}

func TestCompareNames(t *testing.T) {
	tests := []struct {
		name1  string
		name2  string
		output bool
	}{
		{"Vermont Medical Clinic", "Vermont Medical Clinic", true},
		{"Vermont Medical Clinic [no bulk-billing]", "Vermont Medical Clinic", true},
		{"Vermont Medical    Clinic [no bulk-billing]", "Vermont Medical Clinic", true},
		{"Vermont Medical Clinic [no bulk-billing]", "  Vermont    Medical Clinic", true},
		{"Vermont Medical Clinic [no bulk-billing]", "Vermont Medical Clinics", false},
	}
	for _, test := range tests {
		res := compareNames(test.name1, test.name2)
		require.Equal(t, test.output, res)
	}
}
