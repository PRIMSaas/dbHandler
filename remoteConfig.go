package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/firebaseremoteconfig/v1"
)

const BASE_URL = "https://firebaseremoteconfig.googleapis.com"
const REMOTE_CONFIG_ENDPOINT = "/v1/projects/" + PROJECT_ID + "/remoteConfig"
const REMOTE_CONFIG_URL = BASE_URL + REMOTE_CONFIG_ENDPOINT

/* type RemoteConfig struct {
	Parameters map[string]Parameter `json:"parameters"`
	Etag       string               `json:"etag"`
}

type Parameter struct {
	DefaultValue      Value            `json:"defaultValue"`
	ConditionalValues map[string]Value `json:"conditionalValues"`
	Description       string           `json:"description"`
}

type Value struct {
	Value string `json:"value"`
} */

var params map[string]firebaseremoteconfig.RemoteConfigParameter = make(map[string]firebaseremoteconfig.RemoteConfigParameter)

func initConfig() {
	token, err := getAccessToken(KEYFILE)
	if err != nil {
		logError.Printf("Failed to get access token: %v", err)
		return
	}

	remoteConfigStr, err := fetchRemoteConfig(token)
	if err != nil {
		logError.Printf("Failed to get remote config: %v", err)
		return
	}

	var remoteConfig firebaseremoteconfig.RemoteConfig
	err = json.Unmarshal([]byte(remoteConfigStr), &remoteConfig)
	if err != nil {
		logError.Printf("Failed to parse remote config: %v", err)
		return
	}
	params = remoteConfig.Parameters
}

func getConfig(key string, defaultVal string) string {

	val, ok := params[key]
	if !ok {
		logError.Printf("%v key not found in remote config parameters", key)
		return defaultVal
	}
	return val.DefaultValue.Value
}

func getAccessToken(credentialFile string) (string, error) {
	data, err := os.ReadFile(credentialFile)
	if err != nil {
		return "", err
	}

	conf, err := google.JWTConfigFromJSON(data, REMOTE_CONFIG_AUTH_URL)
	if err != nil {
		return "", err
	}
	//
	// Note to self. This is what is happening under the covers
	//
	// var c = struct {
	// 	Email      string `json:"client_email"`
	// 	PrivateKey string `json:"private_key"`
	// }{}
	// json.Unmarshal(b, &c)
	// config := &jwt.Config{
	// 	Email:      c.Email,
	// 	PrivateKey: []byte(c.PrivateKey),
	// 	Scopes: []string{
	// 		"https://www.googleapis.com/auth/firebase.remoteconfig",
	// 	},
	// 	TokenURL: google.JWTTokenURL,
	// }
	ctx, cancel := createContext()
	defer cancel()
	token, err := conf.TokenSource(ctx).Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func fetchRemoteConfig(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", REMOTE_CONFIG_URL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get remote config: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
