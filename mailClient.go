package main

import (
	"errors"
	"fmt"
	"log"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
	"github.com/mailjet/mailjet-apiv3-go/v4/resources"
)

type MailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (a *MailAddress) cast() *mailjet.RecipientV31 {
	rec := (*mailjet.RecipientV31)(a)
	return rec
}

type AttachmentV31 struct {
	ContentType   string `json:"ContentType,omitempty"`
	Base64Content string `json:"Base64Content,omitempty"`
	Filename      string `json:"Filename,omitempty"`
}

func (a AttachmentV31) cast() mailjet.AttachmentV31 {
	att := (mailjet.AttachmentV31)(a)
	if att.ContentType == "" {
		att.ContentType = "application/pdf"
	}
	return att
}

type mailMessage struct {
	From        MailAddress
	To          []MailAddress
	Subject     string          `json:"subject"`
	TextPart    string          `json:"textPart"`
	Attachments []AttachmentV31 `json:"attachments"`
}

func (m mailMessage) copyMsg() mailjet.InfoMessagesV31 {
	return mailjet.InfoMessagesV31{
		From:        m.From.cast(),
		To:          copyMailAddress(m.To),
		Subject:     m.Subject,
		TextPart:    m.TextPart,
		Attachments: copyAttachments(m.Attachments),
	}
}

func copyAttachments(att []AttachmentV31) *mailjet.AttachmentsV31 {
	attachments := make(mailjet.AttachmentsV31, len(att))
	for i, a := range att {
		attachments[i] = a.cast()
	}
	return &attachments
}

func copyMailAddress(ma []MailAddress) *mailjet.RecipientsV31 {
	recipients := make(mailjet.RecipientsV31, len(ma))
	for i, addr := range ma {
		recipients[i] = *addr.cast()
	}
	return &recipients
}

func send(msg []mailMessage) (*mailjet.ResultsV31, error) {
	pubkey := getConfig(MAIL_PUBLIC_KEY, "")
	privKey := getConfig(MAIL_PRIVATE_KEY, "")
	if pubkey == "" || privKey == "" {
		log.Fatal("Mailjet public or private key not found")
		return nil, errors.New("Mailjet public or private key not found")
	}

	mailjetClient := mailjet.NewMailjetClient(pubkey, privKey)

	messagesInfo := []mailjet.InfoMessagesV31{}
	for _, m := range msg {
		messagesInfo = append(messagesInfo, m.copyMsg())
	}

	messages := mailjet.MessagesV31{Info: messagesInfo}

	res, err := mailjetClient.SendMailV31(&messages)
	if err != nil {
		logError.Printf("Failed to send mail: %v", err)
		return nil, err
	}
	fmt.Printf("Data: %+v\n", res)
	return res, nil
}

func sendMail(msgs []mailjet.InfoMessagesV31) (mailjet.ResultsV31, error) {
	pubkey := getConfig(MAIL_PUBLIC_KEY, "")
	privKey := getConfig(MAIL_PRIVATE_KEY, "")
	if pubkey == "" || privKey == "" {
		log.Fatal("Mailjet public or private key not found")
		return mailjet.ResultsV31{}, errors.New("Mailjet public or private key not found")
	}

	mailjetClient := mailjet.NewMailjetClient(pubkey, privKey)
	messages := mailjet.MessagesV31{Info: msgs}

	res, err := mailjetClient.SendMailV31(&messages)
	if err != nil {
		logError.Printf("Failed to send mail: %v", err)
		return mailjet.ResultsV31{}, err
	}
	fmt.Printf("Data: %+v\n", res)
	return *res, err
}

func registerSender(mad mailjet.RecipientV31) (*resources.Sender, error) {
	m, err := getMailClient()
	if err != nil {
		logError.Printf("Failed to get mail client: %v", err)
		return nil, err
	}

	var data []resources.Sender
	logInfo.Printf("Create new contact: %s\n", mad.Email)
	//exclude := false
	fmr := &mailjet.FullRequest{
		Info:    &mailjet.Request{Resource: "sender"},
		Payload: &resources.Contact{Name: mad.Name, Email: mad.Email, IsExcludedFromCampaigns: false},
	}
	err = m.Post(fmr, &data)
	if err != nil {
		logError.Printf("Unexpected error: %v", err)
		return nil, err
	}
	if data == nil {
		logError.Print("Empty result")
		return nil, errors.New("register sender empty result")
	} else {
		logInfo.Printf("Data: %+v\n", data[0])
	}
	return &data[0], nil
}

func getMailClient() (*mailjet.Client, error) {
	pubkey := getConfig(MAIL_PUBLIC_KEY, "")
	privKey := getConfig(MAIL_PRIVATE_KEY, "")
	if pubkey == "" || privKey == "" {
		logError.Print("Mailjet public or private key not found")
		return nil, errors.New("Mailjet public or private key not found")
	}
	return mailjet.NewMailjetClient(pubkey, privKey), nil
}
