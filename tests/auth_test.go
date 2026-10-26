package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/client"
	"payment-api-tests/internal/models"
)

func TestLogin_ValidCredentials_ReturnsShortLivedJWT(t *testing.T) {
	t.Parallel()

	resp, err := api.Login(models.LoginRequest{Email: cfg.Email, ClientSecret: cfg.ClientSecret})
	require.NoError(t, err)
	require.Equal(t, 200, resp.Status, resp.String())

	var body models.LoginResponse
	require.NoError(t, resp.JSON(&body))
	require.NotEmpty(t, body.Token)

	claims, err := client.TokenClaims(body.Token)
	require.NoError(t, err)
	assert.Equal(t, cfg.Email, claims["email"])

	exp, err := client.TokenExpiry(body.Token)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), exp, 30*time.Second, "token should be valid for 5 minutes")
}

func TestLogin_WrongCredentials_Returns401(t *testing.T) {
	t.Parallel()

	cases := map[string]models.LoginRequest{
		"wrong secret":  {Email: cfg.Email, ClientSecret: "wrong-secret-123"},
		"unknown email": {Email: "nobody@example.com", ClientSecret: cfg.ClientSecret},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := api.Login(req)
			require.NoError(t, err)
			require.Equal(t, 401, resp.Status, resp.String())

			var e models.APIError
			require.NoError(t, resp.JSON(&e))
			assert.Equal(t, "unauthorized", e.Error.Code)
			assert.Equal(t, "authentication_error", e.Error.Type)
			// The same message for both cases avoids revealing which accounts exist.
			assert.Equal(t, "Invalid email or client secret", e.Error.Message)
		})
	}
}

func TestLogin_InvalidRequest_Returns400(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		body    any
		wantMsg string
	}{
		{"missing email", models.LoginRequest{ClientSecret: cfg.ClientSecret}, "'Email' failed on the 'required' tag"},
		{"missing secret", models.LoginRequest{Email: cfg.Email}, "'ClientSecret' failed on the 'required' tag"},
		{"malformed email", models.LoginRequest{Email: "not-an-email", ClientSecret: cfg.ClientSecret}, "'Email' failed on the 'email' tag"},
		{"secret below 6 chars", models.LoginRequest{Email: cfg.Email, ClientSecret: "abcde"}, "'ClientSecret' failed on the 'min' tag"},
		{"secret above 100 chars", models.LoginRequest{Email: cfg.Email, ClientSecret: repeat("a", 101)}, "'ClientSecret' failed on the 'max' tag"},
		{"malformed JSON", `{"email":`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := api.Login(tc.body)
			require.NoError(t, err)
			require.Equal(t, 400, resp.Status, resp.String())

			var e models.APIError
			require.NoError(t, resp.JSON(&e))
			assert.Equal(t, "invalid_request", e.Error.Code)
			assert.Contains(t, e.Error.Message, tc.wantMsg)
		})
	}
}
