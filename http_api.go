package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
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
		errStr := fmt.Sprintf("Error processing file: %v", err)
		http.Error(writer, errStr, http.StatusUnprocessableEntity)
		return
	}
	// On success, set the Content-Type header to application/json
	writer.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(writer).Encode(resp)
	if err != nil {
		errStr := fmt.Sprintf("Error encoding response body: %v", err)
		http.Error(writer, errStr, http.StatusInternalServerError)
		return
	}
	duration := time.Since(start)
	logInfo.Printf("Processing file took: %v", duration)
}

// Returns 200 for successfully sent validation and 202 for already ACTIVE
func registerNewSender(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()

	body, err := io.ReadAll(request.Body)
	if err != nil {
		errs := fmt.Sprintf("Error reading request body: %v", err)
		http.Error(writer, errs, http.StatusInternalServerError)
		return
	}
	mad := MailAddress{}
	err = json.Unmarshal(body, &mad)
	if err != nil {
		errs := fmt.Sprintf("Error parsing json body: %v", err)
		http.Error(writer, errs, http.StatusBadRequest)
		return
	}
	active, err := registerOrValidateSender(mad)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		errStr := fmt.Sprintf("Error registerNewSender: %v", err)
		http.Error(writer, errStr, http.StatusServiceUnavailable)
		return
	}
	if active {
		writer.WriteHeader(http.StatusAccepted)
	}
	duration := time.Since(start)
	logInfo.Printf("registerNewSender file took: %v", duration)
}

func checkSenderActive(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		errs := fmt.Sprintf("Error reading request body: %v", err)
		http.Error(writer, errs, http.StatusInternalServerError)
		return
	}
	mad := MailAddress{}
	err = json.Unmarshal(body, &mad)
	if err != nil {
		errs := fmt.Sprintf("Error parsing json body: %v", err)
		http.Error(writer, errs, http.StatusBadRequest)
		return
	}
	active, _, err := senderActive(mad)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		errStr := fmt.Sprintf("Error senderActive: %v", err)
		http.Error(writer, errStr, http.StatusServiceUnavailable)
		return
	}
	if !active {
		http.Error(writer, "Not active", http.StatusNotFound)
		return
	}
	duration := time.Since(start)
	logInfo.Printf("senderActive file took: %v", duration)

}
func processMail(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()

	body, err := io.ReadAll(request.Body)
	if err != nil {
		errs := fmt.Sprintf("Error reading mail request body: %v", err)
		http.Error(writer, errs, http.StatusInternalServerError)
		return
	}
	msg := []MailMsg{}
	err = json.Unmarshal(body, &msg)
	if err != nil {
		errs := fmt.Sprintf("Error parsing mail json body: %v", err)
		http.Error(writer, errs, http.StatusBadRequest)
		return
	}
	err = processSendingMail(msg)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		errStr := fmt.Sprintf("Error sending mail: %v", err)
		http.Error(writer, errStr, http.StatusUnprocessableEntity)
		return
	}
	// On success, set the Content-Type header to application/json
	writer.Header().Set("Content-Type", "application/json")
	duration := time.Since(start)
	logInfo.Printf("Send mail took: %v", duration)
}

func runHttpApi(port int) {
	httpAddress := fmt.Sprintf("0.0.0.0:%d", port)

	co := cors.Options{
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowedOrigins:   []string{"http://127.0.0.1:5055", "*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"POST", "OPTIONS", "GET"},
		MaxAge:           7200,
	}
	if properties.Debug {
		logInfo.Println("Enabling debug")
		co.Debug = true
	}
	c := cors.New(co)
	logInfo.Printf("Starting HTTP server: %s", httpAddress)
	mux := http.NewServeMux()
	mux.HandleFunc("/processFile", processFile)

	//mux.HandleFunc("/register", registerNewSender)
	//mux.HandleFunc("/active", checkSenderActive)
	//mux.HandleFunc("/mail", processMail)
	mux.HandleFunc("/profile", pprof.Profile)
	mux.HandleFunc("/health",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	handler := c.Handler(mux)

	err := http.ListenAndServe(httpAddress, handler)
	if err != nil {
		logError.Print(err)
	}
}
