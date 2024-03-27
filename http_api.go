package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var userIdKey = "userId"

func getClinicsHttp(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusInternalServerError)
		return
	}
	vals := jsonToMap(string(body))
	userID := string(vals[userIdKey].(string))
	ctx, cancel := createContext()
	defer cancel()
	data, err := getCompanies(ctx, gClient, userID)
	if err != nil {
		http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	// Set the Content-Type header to application/json
	writer.Header().Set("Content-Type", "application/json")

	// Use json.NewEncoder to write the data to the response writer
	err = json.NewEncoder(writer).Encode(data)
	if err != nil {
		http.Error(writer, "Error encoding response body", http.StatusInternalServerError)
	}

}

func setCompaniesHttp(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusInternalServerError)
		return
	}
	vals := jsonToMap(string(body))
	userID := string(vals[userIdKey].(string))
	companies := jsonToCompanies("{}") //string(vals["companies"].(string)))
	ctx, cancel := createContext()
	defer cancel()
	err = setCompanies(ctx, gClient, userID, companies)
	if err != nil {
		http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
	}
}

func processFile(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		errs := fmt.Sprintf("Error reading request body: %v", err)
		http.Error(writer, errs, http.StatusInternalServerError)
		return
	}
	file := PaymentFile{}
	err = json.Unmarshal(body, &file)
	if err != nil {
		errs := fmt.Sprintf("Error parsing json body: %v", err)
		http.Error(writer, errs, http.StatusInternalServerError)
		return
	}
	resp, err := processFileContent(file)
	if err != nil {
		http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
	}
	// Set the Content-Type header to application/json
	writer.Header().Set("Content-Type", "application/json")

	// Use json.NewEncoder to write the data to the response writer
	err = json.NewEncoder(writer).Encode(resp)
	if err != nil {
		http.Error(writer, "Error encoding response body", http.StatusInternalServerError)
	}

}

func jsonToCompanies(jsonStr string) []companyDetails {
	//var result []companyDetails
	//json.Unmarshal([]byte(jsonStr), &result)
	initCompanies := []companyDetails{}
	initCompanies = append(initCompanies, companyDetails{
		Name:    "Test Clinic",
		Address: "123 Test St",
	})

	return initCompanies
}

func jsonToMap(jsonStr string) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}

func limitNumClients(f http.HandlerFunc, maxClients int) http.HandlerFunc {
	// Counting semaphore using a buffered channel
	sema := make(chan struct{}, maxClients)

	return func(w http.ResponseWriter, req *http.Request) {
		sema <- struct{}{}
		defer func() { <-sema }()
		f(w, req)
	}
}

func runHttpApi(port int, maxClients int) {
	httpAddress := fmt.Sprintf("0.0.0.0:%d", port)

	logInfo.Printf("Starting HTTP server: %s with max clients: %d", httpAddress, maxClients)

	http.HandleFunc("/getCompanies", limitNumClients(getClinicsHttp2, maxClients))
	http.HandleFunc("/setCompanies", limitNumClients(setCompaniesHttp, maxClients))
	http.HandleFunc("/processFile", processFile)

	err := http.ListenAndServe(httpAddress, nil)
	if err != nil {
		logError.Print(err)
	}
}
