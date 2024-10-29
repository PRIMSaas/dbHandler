package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var addr = Address{Name: "A Practice",
	StreetAddress: "22 Happy St",
	City:          "Melbourne, 3000, VIC",
	ABN:           "12345678901",
	Email:         "a@b.c"}

func ReadTestFile(fileName string) (string, error) {
	b, err := os.ReadFile("PaymentsExport.csv")
	if err != nil {
		return "", fmt.Errorf("Error while reading the file: %v", err)
	}
	return string(b), nil
}

// TODO: Add test for the service code totals. This seems too difficult.
// A solution would be to add a test file which has already been manually verified.
// Maybe use PaymentsExport.csv, but it needs payments with GST to cover more permutations.

func TestCalculate1(t *testing.T) {
	configureLogging()
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	//dr2 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,91182,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"
	drName := "Dr Aha"

	paymentFile := PaymentFile{
		FileContent:    dr1, //+ "\n" + dr2,
		CsvLineStart:   0,
		CompanyDetails: addr,
		CodeMap:        map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:        map[string]map[string]string{drName: {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
		PracDetails:    map[string]Address{drName: addr},
		AdjustMap:      map[string][]Adjustments{drName: {Adjustments{Description: "adjustment1", Amount: 10}, Adjustments{"adjustment1", 5}}},
	}
	var res FileProcessingResponse
	res, err := processFileContent(paymentFile)
	require.NoError(t, err)
	require.Equal(t, "Dr Aha", res.ChargeDetail[drName].Provider)
	require.Equal(t, "Sick Patient", res.ChargeDetail[drName].PaymentDetails[0].Patient)
	require.Equal(t, "80010", res.ChargeDetail[drName].PaymentDetails[0].ItemNo)
	require.Equal(t, "(224.50)", res.ChargeDetail[drName].PaymentDetails[0].Payment)
	require.Equal(t, "(67.35)", res.ChargeDetail[drName].PaymentDetails[0].ServiceFee)
	require.Equal(t, "30", res.ChargeDetail[drName].ServiceCodeSplit["code1"].Rate)
	require.NotEmpty(t, res)
}

func TestCalcErrorItemMappingServiceCode(t *testing.T) {
	configureLogging()
	goodCode := "code1"
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	
	drName := "Dr Aha"
	paymentFile := PaymentFile{
		FileContent:    dr1, // + "\n" + dr2,
		CsvLineStart:   0,
		CompanyDetails: addr,
		CodeMap:        map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:        map[string]map[string]string{drName: {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
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

func TestMissingItemNr(t *testing.T) {
	configureLogging()
	dr1 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"

	drName := "Dr Aha"
	paymentFile := PaymentFile{
		FileContent:    dr1,
		CsvLineStart:   0,
		CompanyDetails: addr,
		CodeMap:        map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:        map[string]map[string]string{drName: {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
	}
	var res FileProcessingResponse
	res, err := processFileContent(paymentFile)
	require.NoError(t, err)
	require.Empty(t, res.NoItemNrs)
	require.Empty(t, res.MissingServiceCodes)
	require.NotEmpty(t, res.MissingItemNrs)
	require.NotEmpty(t, res.MissingItemNrs["Psychological therapy health service provided by phone"])

}

func TestCalcErrorBadNumbers(t *testing.T) {

	configureLogging()
	drName := "Dr Aha"
	goodCode := "code1"
	dr1 := "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50),0.00"
	//dr2 := "B Practice,Dr Buhu,Irrelevant,Patient Name,162436,174678,72714,91182,\"Psychological therapy health service provided by phone\",Reversed payment,26/02/2024,Direct Credit,Private,0.00,(224.50),0.00"

	paymentFile := PaymentFile{
		FileContent:    dr1, // + "\n" + dr2,
		CsvLineStart:   0,
		CompanyDetails: addr,
		CodeMap:        map[string][]string{"code1": {"80010", "456"}, "code2": {"789", "012"}},
		PracMap:        map[string]map[string]string{drName: {"code1": "30", "code2": "20"}, "Dr Buhu": {"code1": "40", "code2": "30"}},
	}
	//
	// Missing service code for Dr
	//
	delete(paymentFile.PracMap[drName], goodCode)
	paymentFile.PracMap[drName][goodCode] = "hello"
	_, err := processFileContent(paymentFile)
	require.Error(t, err)
	// blank percentage
	delete(paymentFile.PracMap[drName], goodCode)
	paymentFile.PracMap[drName][goodCode] = ""
	_, err = processFileContent(paymentFile)
	require.Error(t, err)
	// restore the good code
	paymentFile.PracMap[drName][goodCode] = "30"
	// bad payment
	paymentFile.FileContent = "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,(224.50.50),0.00"
	_, err = processFileContent(paymentFile)
	require.Error(t, err)
	// blank payment
	paymentFile.FileContent = "A Practice [no bulk-billing],Dr Aha,Irrelevant,Sick Patient,162307,174545,71756,80010,\"Clinical psychologist consultation, >50 min, consulting rooms\",Reversed payment,01/03/2024,EFT,Private,0.00,,0.00"
	_, err = processFileContent(paymentFile)
	require.Error(t, err)
}

func TestConvertPayment(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{"123.45", "123.45"},
		{"123", "123.00"},
		{"123.4", "123.40"},
		{"123.456", "123.45"},
		{"(123.45)", "(123.45)"},
		{"(123", "(123.00)"},
		{"(123.4)", "(123.40)"},
		{"(123.456)", "(123.45)"},
	}
	for _, test := range tests {
		res, err := convertToInt(test.input)
		require.NoError(t, err)
		require.Equal(t, test.output, cents2DStr(res))
	}
	_, err := convertToInt("test.input")
	require.Error(t, err)
	_, err = convertToInt("123)")
	require.Error(t, err)
	_, err = convertToInt("1 23)")
	require.Error(t, err)
}

func TestConvertPecentage(t *testing.T) {
	tests := []struct {
		input  string
		output int
	}{
		{"10%", 1000},
		{"80", 8000},
		{"5.0%", 500},
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
		gst        string
		percentage string
		exGst      int
		billed     int
		totalP     int
		gstc       int
	}{
		// positive numbers, vary payemnt and precentage
		{"123.45", "0", "10%", 12345, 1235, 12345, 0},
		{"123", "0", "80", 12300, 9840, 12300, 0},
		{"123.4", "0", "", 12340, 0, 12340, 0},
		{"123.456", "0", "5.0%", 12345, 617, 12345, 0},
		// same as above but negative
		{"(123.45)", "0", "10%", -12345, -1235, -12345, 0},
		{"(123", "0", "80", -12300, -9840, -12300, 0},
		{"(123.4)", "0", "", -12340, 0, -12340, 0},
		{"(123.456)", "0", "5.0%", -12345, -617, -12345, 0},
		{"(80.1)", "0", "5.5%", -8010, -441, -8010, 0},
		// now with gst
		{"123.45", "10.0", "10%", 11345, 1135, 12345, 1000},
		{"123", "22.00", "80", 10100, 8080, 12300, 2200},
		{"(123.4)", "12.3456", "0.8", -13574, -109, -12340, 1234},
	}
	for idx, test := range tests {
		//exGst, fee, totalP, gst, err
		exGst, fee, totalP, gst, err := calcPayment(test.payment, test.gst, test.percentage)
		if test.billed == 0 {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.exGst, exGst, "Failed calcPayment exGst test %d", idx)
			require.Equal(t, test.billed, fee, "Failed calcPayment fee test %d", idx)
			require.Equal(t, test.totalP, totalP, "Failed calcPayment payment conversion test %d", idx)
			require.Equal(t, test.gstc, gst, "Failed calcPayment gst conversion test %d", idx)
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

func TestCalculator(t *testing.T) {
	testCases := []struct {
		dval   string
		perc   string
		pcents int
		cents  int
	}{
		{"10.99", "10.99", 121, 1099}, {"5.5", "5.5", 30, 550}, {"3.14159", "3.14159", 10, 314},
		{"20", "20", 400, 2000}, {"$15.758", "15.758", 248, 1575}}

	for _, tc := range testCases {
		cents, err := dollarStringToCents(tc.dval)
		require.NoError(t, err)
		require.Equal(t, tc.cents, cents)

		_, res, _, _, err := calcPayment(tc.dval, "0", tc.perc)
		require.NoError(t, err)
		require.Equal(t, tc.pcents, res)
	}
}
