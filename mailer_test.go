package main

import (
	"testing"

	"math/rand"

	// mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
	"github.com/stretchr/testify/require"
)

var sender = MailAddress{
	Name:  "Jimbotron",
	Email: "test1" + "@example.com",
}

func TestGetSenders(t *testing.T) {
	intiTest()

	res, err := getSenders()
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res)
}

// Needs to be tested against a known active email address
func SkipTestSenderActive(t *testing.T) {
	intiTest()

	sender.Email = "jimbotech007@gmail.com"
	active, exists, err := senderActive(sender)
	require.NoError(t, err)
	require.True(t, active)
	require.True(t, exists)
}
func TestSendNotActive(t *testing.T) {
	intiTest()

	sender.Email = "madeup@gmail.com"
	active, exists, err := senderActive(sender)
	require.NoError(t, err)
	require.False(t, active)
	require.False(t, exists)
}

// Can run this test manually, but running it repeatedly will just store
// a lot of addresses against that account
func SkipTestRegisterOrValidateThenDeleteSender(t *testing.T) {
	intiTest()
	rstr := randSeq(10)
	sender.Email = rstr + "@example.com"
	// Execute
	_, err := registerOrValidateSender(sender)
	require.NoError(t, err)

	res, err := getSenders()
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res)
	_, ok := res[sender.Email]
	require.True(t, ok)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		//nolint:gosec // G404 crypto random is not required here
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

/* func TestRegisterSender(t *testing.T) {
	// Setup
	intiTest()

	rstr := randSeq(10)
	sender := mailjet.RecipientV31{
		Name:  "Jimbotron",
		Email: rstr + "@mailjet.com",
	}

	// Execute
	res, err := registerJetSender(sender)
	require.NoError(t, err)
	require.NotNil(t, res)
} */
