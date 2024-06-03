package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func ReadTestFile(fileName string) (string, error) {
	b, err := os.ReadFile("PaymentsExport.csv")
	if err != nil {
		return "", fmt.Errorf("Error while reading the file: %v", err)
	}
	return string(b), nil
}

func TestCalculate1(t *testing.T) {
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	//dr2 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,91182,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"

	paymentFile := PaymentFile{
		FileContent:  dr1, //+ "\n" + dr2,
		CsvLineStart: 0,
		CompanyName:  "A Practice",
		CodeMap:      map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:      map[string]map[string]string{"Dr Aha": {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
	}
	var res FileProcessingResponse
	res, err := processFileContent(paymentFile)
	require.NoError(t, err)
	require.Equal(t, "Dr Aha", res.ChargeDetail[0].Provider)
	require.Equal(t, "Sick Patient", res.ChargeDetail[0].Patient)
	require.Equal(t, "80010", res.ChargeDetail[0].ItemNo)
	require.Equal(t, "(224.50)", res.ChargeDetail[0].Payment)
	require.Equal(t, "-67.35", res.ChargeDetail[0].ServiceFee)
	require.NotEmpty(t, res)
}

func TestCalcErrorItemMappingServiceCode(t *testing.T) {
	configureLogging()
	goodCode := "code1"
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	//dr2 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,91182,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"

	drName := "Dr Aha"
	paymentFile := PaymentFile{
		FileContent:  dr1, // + "\n" + dr2,
		CsvLineStart: 0,
		CompanyName:  "A Practice",
		CodeMap:      map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:      map[string]map[string]string{drName: {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
	}
	var res FileProcessingResponse
	//
	// invalid item number
	//
	delete(paymentFile.CodeMap, goodCode)
	paymentFile.CodeMap[goodCode] = []string{"80011", "456"}
	res, err := processFileContent(paymentFile)
	require.NoError(t, err)
	_, ok := res.MissingItemNrs["80010"]
	require.True(t, ok)
	// restore
	delete(paymentFile.CodeMap, goodCode)
	paymentFile.CodeMap[goodCode] = []string{"80010", "456"}
	//
	// Dr missing service code mapping
	//
	delete(paymentFile.PracMap, drName)
	res, err = processFileContent(paymentFile)
	require.NoError(t, err)
	_, ok = res.MissingProviders[drName]
	require.True(t, ok)
	// restore
	paymentFile.PracMap[drName] = map[string]string{"code1": "30", "code2": "20"}
	//
	// Missing service code for Dr
	//
	delete(paymentFile.PracMap[drName], goodCode)
	res, err = processFileContent(paymentFile)
	require.NoError(t, err)
	_, ok = res.MissingServiceCodes[drName]
	require.True(t, ok)
	// restore the good code
	paymentFile.PracMap[drName][goodCode] = "30"
}

func TestCalcErrorBadNumbers(t *testing.T) {

	configureLogging()
	drName := "Dr Aha"
	goodCode := "code1"
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	//dr2 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,91182,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"

	paymentFile := PaymentFile{
		FileContent:  dr1, // + "\n" + dr2,
		CsvLineStart: 0,
		CompanyName:  "A Practice",
		CodeMap:      map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:      map[string]map[string]string{drName: {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
	}
	var res FileProcessingResponse //[]PaymentFileResponse
	//
	// Missing service code for Dr
	//
	delete(paymentFile.PracMap[drName], goodCode)
	paymentFile.PracMap[drName][goodCode] = "hello"
	res, err := processFileContent(paymentFile)
	require.NoError(t, err)
	require.NotEmpty(t, res.ChargeDetail[0].ErrorMsg)
	// blank percentage
	delete(paymentFile.PracMap[drName], goodCode)
	paymentFile.PracMap[drName][goodCode] = ""
	res, err = processFileContent(paymentFile)
	require.NoError(t, err)
	require.NotEmpty(t, res.ChargeDetail[0].ErrorMsg)
	// restore the good code
	paymentFile.PracMap[drName][goodCode] = "30"
	// bad payment
	paymentFile.FileContent = "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50.50),0.00"
	res, err = processFileContent(paymentFile)
	require.NoError(t, err)
	require.NotEmpty(t, res.ChargeDetail[0].ErrorMsg)
	// blank payment
	paymentFile.FileContent = "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,,0.00"
	res, err = processFileContent(paymentFile)
	require.NoError(t, err)
	require.NotEmpty(t, res.ChargeDetail[0].ErrorMsg)
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
	_, err := convertToFloat("test.input")
	require.Error(t, err)
	_, err = convertToFloat("123)")
	require.Error(t, err)
	_, err = convertToFloat("1 23)")
	require.Error(t, err)
}

func TestConvertPecentage(t *testing.T) {
	tests := []struct {
		input  string
		output float64
	}{
		{"10%", 10},
		{"80", 80},
		{"5.0%", 5},
	}
	for _, test := range tests {
		res, err := convertPercToFloat(test.input)
		require.NoError(t, err)
		require.Equal(t, test.output, res)
	}
	_, err := convertPercToFloat("5 0")
	require.Error(t, err)
	_, err = convertPercToFloat("")
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

func TestStandardString(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{"Vermont Medical Clinic", "vermont medical clinic"},
		{"Vermont Medical Clinic [no bulk-billing]", "vermont medical clinic [no bulk-billing]"},
		{"Vermont Medical    Clinic [no bulk-billing]", "vermont medical clinic [no bulk-billing]"},
		{"Vermont Medical \t Clinic [no bulk-billing]", "vermont medical clinic [no bulk-billing]"},
		{"  Vermont    Medical Clinic", "vermont medical clinic"},
	}
	for _, test := range tests {
		res := standardString(test.input)
		require.Equal(t, test.output, res)
	}
}
