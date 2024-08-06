package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/cors"
)

func processFile(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()

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
		http.Error(writer, errs, http.StatusBadRequest)
		return
	}
	resp, err := processFileContent(file)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		http.Error(writer, fmt.Sprint(err), http.StatusUnprocessableEntity)
		return
	}

	err = json.NewEncoder(writer).Encode(resp)
	if err != nil {
		http.Error(writer, "Error encoding response body", http.StatusInternalServerError)
		return
	}
	// On success, set the Content-Type header to application/json
	writer.Header().Set("Content-Type", "application/json")
	duration := time.Since(start)
	logInfo.Printf("Processing file took: %v", duration)
}

func runHttpApi(port int, maxClients int) {
	httpAddress := fmt.Sprintf("0.0.0.0:%d", port)

	co := cors.Options{
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowedOrigins:   []string{"http://127.0.0.1:5055", "*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"POST", "OPTIONS"},
		MaxAge:           7200,
	}
	if properties.Debug {
		logInfo.Println("Enabling debug")
		co.Debug = true
	}
	c := cors.New(co)
	logInfo.Printf("Starting HTTP server: %s with max clients: %d", httpAddress, maxClients)
	mux := http.NewServeMux()
	mux.HandleFunc("/processFile", processFile)
	handler := c.Handler(mux)

	err := http.ListenAndServe(httpAddress, handler)
	if err != nil {
		logError.Print(err)
	}
}
