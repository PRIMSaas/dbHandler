package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	res, err := http.Get("http://localhost:8080")
	if err != nil || res.StatusCode != http.StatusOK {
		startEmulators(t)
		defer stopEmulators(t)
	}

	client := initClient()
	companies := []companyDetails{}
	companies = append(companies, companyDetails{
		Name:    "Test Clinic",
		Address: "123 Test St",
	})
	userId := "testUser"
	deleteUser(ctx, client, userId)
	err = setCompanies(ctx, client, userId, companies)
	require.NoError(t, err)
	docs, err := getCompanies(ctx, client, userId)

	require.NotNil(t, docs)
	require.NotZero(t, len(docs))
	require.NoError(t, err)
	deleteUser(ctx, client, userId)
}
