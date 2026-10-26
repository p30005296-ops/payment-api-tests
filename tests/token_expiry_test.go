//go:build slow

// Waits for a real token to expire (~5 minutes). Run with: go test -tags slow -run Expiry ./tests/
package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/client"
	"payment-api-tests/internal/models"
)

func TestToken_RejectedAfterExpiry(t *testing.T) {
	resp, err := api.Login(models.LoginRequest{Email: cfg.Email, ClientSecret: cfg.ClientSecret})
	require.NoError(t, err)
	var body models.LoginResponse
	require.NoError(t, resp.JSON(&body))

	exp, err := client.TokenExpiry(body.Token)
	require.NoError(t, err)

	authorise := func() *client.Response {
		r, err := api.Do("POST", client.PathTransactions, newAuthorisation(), bearer(body.Token))
		require.NoError(t, err)
		return r
	}

	require.Equal(t, 200, authorise().Status, "token must work before expiry")

	t.Logf("waiting until %s for the token to expire", exp.Add(5*time.Second).Format(time.TimeOnly))
	time.Sleep(time.Until(exp) + 5*time.Second)

	requireUnauthorised(t, authorise())
}
