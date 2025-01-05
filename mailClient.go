package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const POST = "POST"
const GET = "GET"
const DELETE = "DELETE"
const mailServer = "https://api.mailjet.com/v3/REST/"

type MailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
type MultiMailAddress []MailAddress

type registerSenderResponse struct {
	Count int `json:"count"`
	Data  []SenderDetails
	Total int `json:"total"`
}

// from resources Sender struct
type SenderDetails struct {
	CreatedAt       *RFC3339DateTime `mailjet:"read_only"`
	DNS             string           `mailjet:"read_only"` // deprecated
	DNSID           int64            `mailjet:"read_only"`
	Email           string
	EmailType       string `json:",omitempty"`
	Filename        string `mailjet:"read_only"`
	ID              int64  `mailjet:"read_only"`
	IsDefaultSender bool   `json:",omitempty"`
	Name            string `json:",omitempty"`
	Status          string `mailjet:"read_only"`
}

// also from resources
type RFC3339DateTime struct {
	time.Time
}

type Attachment struct {
	ContentType   string `json:"ContentType,omitempty"`
	Base64Content string `json:"Base64Content,omitempty"`
	Filename      string `json:"Filename,omitempty"`
}
type MultiAttachment []Attachment

type MailMsg struct {
	From        MailAddress
	To          MultiMailAddress
	Attachments MultiAttachment
	Subject     string
	TextPart    string
	HTMLPart    string `json:",omitempty"`
}

type SendMailMsg struct {
	Message []MailMsg `json:"Messages"`
}

type DeGeneratedMessage struct {
	Email       string
	MessageUUID string
	MessageID   int64
	MessageHref string
}

type MailResult struct {
	Status   string
	CustomID string `json:",omitempty"`
	To       []DeGeneratedMessage
	Cc       []DeGeneratedMessage
	Bcc      []DeGeneratedMessage
}

func processSendingMail(messages []MailMsg) error {
	for _, msg := range messages {
		details, ok, err := findSender(msg.To[1])
		if err != nil {
			return fmt.Errorf("failed to get senders: %v", err)
		}
		if !ok || details.Status != "Active" {
			logError.Printf("Sender %v not found or not active. Mail NOT sent", msg.To[1].Email)
		}

		err = sendMail(msg)
		if err != nil {
			return fmt.Errorf("failed to send mail: %v", err)
		}
		logInfo.Printf("Mail sent to %v from %v", msg.To, msg.To[1])
	}
	return nil
}

func sendMail(msg MailMsg) error {
	sendMsg := SendMailMsg{Message: []MailMsg{msg}}
	jsonData, err := json.Marshal(sendMsg)
	if err != nil {
		return fmt.Errorf("failed to convert %v to json: %v", sendMsg, err)
	}
	status, body, err := sendHttp(jsonData, "send", POST)
	if err != nil {
		return fmt.Errorf("failed to send mail: %v", err)
	}
	if status < http.StatusOK || status >= 300 {
		return fmt.Errorf("failed to send mail: received status code %d with message %v",
			status, string(body))
	}
	res := MailResult{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %v", body)
	}
	if res.Status != "success" {
		return fmt.Errorf("failed to send mail: %+v", res)
	}
	return nil
}

func getSenders() (map[string]SenderDetails, error) {
	var mailAddrLookup = make(map[string]SenderDetails)
	senderDetails := registerSenderResponse{}
	status, body, err := sendHttp([]byte{}, "sender", GET)
	if err != nil {
		return mailAddrLookup, fmt.Errorf("failed to get senders: %v", err)
	}
	if status < http.StatusOK || status >= 300 {
		return mailAddrLookup, fmt.Errorf("failed to get senders: received status code %d with message %v",
			status, string(body))
	}

	err = json.Unmarshal(body, &senderDetails)
	if err != nil {
		logError.Printf("Failed to unmarshal response body: %v", body)
	}
	if senderDetails.Count == 0 || len(senderDetails.Data) == 0 {
		return mailAddrLookup, errors.New("failed to get senders: no senders found")
	}
	for _, sd := range senderDetails.Data {
		mailAddrLookup[sd.Email] = sd
	}
	return mailAddrLookup, nil
}

func findSender(mad MailAddress) (SenderDetails, bool, error) {
	senders, err := getSenders()
	if err != nil {
		return SenderDetails{}, false, fmt.Errorf("failed to get senders: %v", err)
	}
	details, ok := senders[mad.Email]
	return details, ok, nil
}

// returns active, exists or error
func senderActive(mad MailAddress) (bool, bool, error) {
	details, ok, err := findSender(mad)
	if err != nil {
		return false, false, fmt.Errorf("failed to get senders: %v", err)
	}
	if !ok {
		return false, false, nil
	}
	if details.Status != "Active" {
		return false, true, nil
	}
	return true, true, nil
}

func registerOrValidateSender(mad MailAddress) (bool, error) {
	active, exists, err := senderActive(mad)
	if err != nil {
		return false, fmt.Errorf("failed to get senders: %v", err)
	}
	if !exists {
		err = registerSender(mad)
		if err != nil {
			return false, fmt.Errorf("failed to register sender: %v", err)
		}
		return false, nil
	}
	if !active {
		err = validateSender(mad)
		if err != nil {
			return false, fmt.Errorf("failed to validate sender: %v", err)
		}
		return false, nil
	}
	return true, nil
}

func registerSender(mad MailAddress) error {

	jsonData, err := json.Marshal(mad)
	if err != nil {
		logError.Printf("Failed to convert %v to json: %v", mad, err)
		return err
	}
	status, body, err := sendHttp(jsonData, "sender", POST)
	if err != nil {
		return fmt.Errorf("failed to register sender: %v", err)
	}
	//
	// This can fail if the sender already exists, I don't care
	//
	if status < http.StatusOK || status >= 300 {
		logInfo.Printf("failed to register sender: received status code %d with message %v",
			status, string(body))
	} else {
		var senderDetails registerSenderResponse
		err = json.Unmarshal(body, &senderDetails)
		if err != nil {
			logError.Printf("Failed to unmarshal response body: %v", err)
			return err
		}
		logInfo.Printf("Sender %v registerd with ID: %+v\n", senderDetails.Data[0].Name, senderDetails.Data[0].Email)
	}
	return nil
}

func validateSender(mad MailAddress) error {
	url := fmt.Sprintf("sender/%s/validate", url.QueryEscape(mad.Email))
	status, body, err := sendHttp([]byte{}, url, POST)
	if err != nil {
		return fmt.Errorf("failed to register sender: %v", err)
	}
	// Check if the response status code is a 2xx value
	if status < http.StatusOK || status >= 300 {
		return fmt.Errorf("failed to validate sender: received status code %d with message %v",
			status, string(body))
	}
	logInfo.Printf("Sender %v validate: %v\n", mad.Name, mad.Email)
	return nil
}

func sendHttp(msg []byte, action string, method string) (int, []byte, error) {
	key, err := basicAuth()
	if err != nil {
		return returnError(fmt.Errorf("failed to get basic auth: %v", err))
	}
	var req *http.Request
	if method == POST {
		req, err = http.NewRequest("POST", mailServer+action, bytes.NewBuffer(msg))
	} else if method == GET || method == DELETE {
		req, err = http.NewRequest(method, mailServer+action, nil)
	} else {
		return returnError(fmt.Errorf("invalid http method: %v", method))
	}
	if err != nil {
		return returnError(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Add("Authorization", "Basic "+key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		//	Jar: cookieJar,
		CheckRedirect: redirectPolicyFunc,
	}

	resp, err := client.Do(req)
	if err != nil {
		return returnError(fmt.Errorf("failed to register sender: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return returnError(fmt.Errorf("failed to read response body: %v", err))
	}
	return resp.StatusCode, body, nil
}

func returnError(errMsg error) (int, []byte, error) {
	return 0, []byte{}, errMsg
}

func basicAuth() (string, error) {
	pubkey := getConfig(MAIL_PUBLIC_KEY, "")
	privKey := getConfig(MAIL_PRIVATE_KEY, "")
	if pubkey == "" || privKey == "" {
		log.Fatal("Mailjet public or private key not found")
		return "", errors.New("Mailjet public or private key not found")
	}

	auth := pubkey + ":" + privKey
	return base64.StdEncoding.EncodeToString([]byte(auth)), nil
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	key, err := basicAuth()
	if err != nil {
		logError.Printf("Failed to get basic auth for redirect: %v", err)
		return err
	}
	req.Header.Add("Authorization", "Basic "+key)
	return nil
}
