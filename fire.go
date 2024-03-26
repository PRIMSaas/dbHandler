package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"firebase.google.com/go/auth"
)

func getClinicsHttp2(w http.ResponseWriter, r *http.Request) {
	// Initialize Firebase App (replace with your project credentials)

	// Get an auth.Client from the firebase.App
	authClient, err := gApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error getting Auth client: %v", err)
	}

	// firestoreClient, err := firestore.NewClient(context.Background(), projectID)
	// if err != nil {
	// 	log.Fatalf("Error creating Firestore client: %v", err)
	// }
	logInfo.Print("getClinicsHttp2 has been called")

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logError.Print("Did not find an authorisation header")
		http.Error(w, "no Authorization header", http.StatusUnauthorized)
		return
	}
	// Validate and extract UID from token (replace with your token validation logic)
	uid, err := validateAndExtractUID(authHeader, authClient)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Unauthorized: %v", err)
		return
	}
	logInfo.Printf("Found token: %v", uid)
	data, err := getCompanies(context.Background(), gClient, uid)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	logInfo.Printf("Found companies: %v", data)

	w.Header().Set("Content-Type", "application/json")

	// Use json.NewEncoder to write the data to the response writer
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, "Error encoding response body", http.StatusInternalServerError)
	}
}

func validateAndExtractUID(token string, firebaseApp *auth.Client) (string, error) {
	// Verify the Firebase Authentication token
	tokenClaims, err := firebaseApp.VerifyIDToken(context.Background(), token)
	if err != nil {
		return "", fmt.Errorf("error verifying token: %w", err)
	}

	// Extract the user ID (uid) from the token claims
	uid := tokenClaims.Subject // Assuming 'sub' claim holds the user ID

	return uid, nil
}
