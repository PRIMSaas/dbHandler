package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/require"
)

func startEmulators(t *testing.T) int {
	cmd := exec.Command("firebase", "emulators:start", "--only", "firestore")
	cmd.Start()
	require.NotNil(t, cmd.Process, "Firestore emulators failed to start")
	return cmd.Process.Pid
}
func stopEmulators(t *testing.T) {
	shutdownCmd := exec.Command("bash", "-c", fmt.Sprintf("lsof -i tcp:%d | grep LISTEN | awk '{print $2}' | xargs kill -9", 8080))
	err := shutdownCmd.Run()
	require.NoError(t, err)
}

func TestSetAndGetClinics(t *testing.T) {
	configureLogging()

	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	res, err := http.Get("http://localhost:8080")
	if err != nil || res.StatusCode != http.StatusOK {
		startEmulators(t)
		defer stopEmulators(t)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	secretPath := KEYPATH
	keys := filepath.Join(secretPath, KEYFILE)

	client := initClient(ctx, keys)

	userId := "testUser"
	// cleanup before start
	deleteUser(ctx, client, userId)
	exist, err := userExists(ctx, client, userId)
	require.NoError(t, err)
	require.False(t, exist)

	initCompanies := []companyDetails{}
	initCompanies = append(initCompanies, companyDetails{
		Name:    "Test Clinic",
		Address: "123 Test St",
	})
	initCompanies = append(initCompanies, companyDetails{
		Name:    "Test2 Clinic",
		Address: "123 Test2 St",
	})
	initCompanies = append(initCompanies, companyDetails{
		Name:    "Test3 Clinic",
		Address: "123 Test3 St",
	})
	companies := initCompanies
	// check adding a list of companies
	companies = checkCompanies(t, ctx, client, userId, companies)
	// check deleting a company from the list
	companies = companies[:len(companies)-1]
	checkCompanies(t, ctx, client, userId, companies)
	// check setting an empty list = delete all companies
	companies = []companyDetails{}
	checkCompanies(t, ctx, client, userId, companies)
	// now add all the companies and delete the  user
	checkCompanies(t, ctx, client, userId, initCompanies)
	// Cleanup
	deleteUser(ctx, client, userId)
	_, err = getCompanies(ctx, client, userId)
	require.Error(t, err)
	fmt.Println("Test has finished successfully!")
}

func checkCompanies(t *testing.T, ctx context.Context, client *firestore.Client,
	userId string, companies []companyDetails) []companyDetails {
	err := setCompanies(ctx, client, userId, companies)
	require.NoError(t, err)

	docs, err := getCompanies(ctx, client, userId)
	require.NotNil(t, docs)
	require.Equal(t, len(companies), len(docs))
	require.NoError(t, err)
	return docs
}
