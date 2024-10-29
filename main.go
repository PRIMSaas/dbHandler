package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

const PROJECT_ID = "drjim-f2087"
const REMOTE_CONFIG_AUTH_URL = "https://www.googleapis.com/auth/firebase.remoteconfig"
const KEYFILE = "drjim-f2087-fc31272ae857.json"
const KEYPATH = "./secret"
const CONFIG = "config.yaml"
const CONFIGPATH = "config"
const PORT = 8088
const CONTEXT_TIMEOUT = 60

var (
	gClient  *firestore.Client
	gApp     *firebase.App
	logError *log.Logger
	logInfo  *log.Logger
)

func main() {
	var keys string
	var config string

	configPath := CONFIGPATH
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	config = filepath.Join(configPath, CONFIG)
	secretPath := KEYPATH
	if len(os.Args) > 2 {
		secretPath = os.Args[2]
	}
	keys = filepath.Join(secretPath, KEYFILE)

	ctx, cancel := createContext()
	defer cancel()

	initStart(keys, config)

	gClient = initClient(ctx, keys)
	defer gClient.Close()

	gApp = initApp(keys)

	runHttpApi(PORT)
}

func initStart(key string, config string) {
	configureLogging()
	if fileExists(config, "config") && fileExists(key, "keyfile") {
		initConfig(key, config)
	}
	//
	// Are we running local against emulators?
	//
	env := getConfig("env", "local")
	if env == "local" {
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
		logInfo.Println("Using Firestore emulator")
	}
}

func initClient(ctx context.Context, keys string) *firestore.Client {
	// Create a Firestore client
	sa := option.WithCredentialsFile(keys)
	client, err := firestore.NewClient(ctx, PROJECT_ID, sa)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}
	return client
}

func initApp(keys string) *firebase.App {
	sa := option.WithCredentialsFile(keys)
	var firebaseConfig = &firebase.Config{
		ProjectID: "drjim-f2087",
	}
	firebaseApp, err := firebase.NewApp(context.Background(), firebaseConfig, sa)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}
	return firebaseApp
}

func configureLogging() {
	logError = log.New(os.Stdout, " ERROR --- ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix)
	logInfo = log.New(os.Stdout, " INFO  --- ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix)
}

func createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), CONTEXT_TIMEOUT*time.Second)
}

func fileExists(file string, name string) bool {
	logInfo.Printf("%v: %v", name, file)
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		logError.Printf("The bloody %v file does not exist %v", name, err)
		return false
	}
	return true
}
