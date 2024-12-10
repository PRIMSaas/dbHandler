package main

import (
	"testing"

	"math/rand"

	"github.com/stretchr/testify/require"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

func TestRegisterSender(t *testing.T) {
	// Setup
	intiTest()

	rstr := randSeq(10)
	sender := mailjet.RecipientV31{
		Name:  "Jimbotron",
		Email: rstr + "@mailjet.com",
	}

	// Execute
	res, err := registerSender(sender)
	require.NoError(t, err)
	require.NotNil(t, res)
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
