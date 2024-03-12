package main

import (
	"context"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

const PROJECT_ID = "drjim-f2087"
const REMOTE_CONFIG_AUTH_URL = "https://www.googleapis.com/auth/firebase.remoteconfig"
const KEYFILE = "./drjim-f2087-fc31272ae857.json"
const PORT = 8088

var (
	ctx      context.Context
	client   *firestore.Client
	logError *log.Logger
	logInfo  *log.Logger
)

func main() {
	timeoutContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = timeoutContext
	defer cancel()

	initStart()

	runHttpApi(PORT, 100)
}

func initStart() {
	configureLogging()

	initConfig()
	//
	// Are we running local against emulators?
	//
	env := getConfig("env", "local")
	if env == "local" {
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
		logInfo.Println("Using Firestore emulator")
	}
	client = initClient()
}

func initClient() *firestore.Client {
	// Create a Firestore client
	sa := option.WithCredentialsFile(KEYFILE)
	client, err := firestore.NewClient(ctx, PROJECT_ID, sa)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}
	return client
}

func configureLogging() {
	logError = log.New(os.Stdout, " ERROR --- ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix)
	logInfo = log.New(os.Stdout, " INFO  --- ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix)
}
