// Package tests holds the end-to-end suite for the payment API.
//
// One Playwright instance and one API client are shared by every test; each
// test creates its own transactions with fresh UUIDs, so tests run in parallel.
package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/client"
	"payment-api-tests/internal/config"
	"payment-api-tests/internal/models"
	"payment-api-tests/internal/testdata"
)

var (
	pw  *playwright.Playwright
	cfg config.Config
	api *client.Client
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	var err error
	if cfg, err = config.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		return 1
	}
	if pw, err = playwright.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "start playwright (did you run the install step in README?):", err)
		return 1
	}
	defer pw.Stop()

	if api, err = client.New(pw, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer api.Close()

	return m.Run()
}

// authorise sends req and decodes a 200 response.
func authorise(t *testing.T, req *models.AuthorisationRequest) models.AuthorisationResponse {
	t.Helper()
	resp, err := api.Authorise(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.Status, resp.String())
	var out models.AuthorisationResponse
	require.NoError(t, resp.JSON(&out))
	return out
}

// complete completes a transaction and decodes a 200 response.
func complete(t *testing.T, id string) models.AuthorisationResponse {
	t.Helper()
	resp, err := api.Complete(id)
	require.NoError(t, err)
	require.Equal(t, 200, resp.Status, resp.String())
	var out models.AuthorisationResponse
	require.NoError(t, resp.JSON(&out))
	return out
}

// requireValidationError asserts a 400 invalid_request whose message names the
// failing field and validator tag, e.g. ("Terminal.Tid", "min").
func requireValidationError(t *testing.T, resp *client.Response, field, tag string) {
	t.Helper()
	require.Equal(t, 400, resp.Status, resp.String())
	var e models.APIError
	require.NoError(t, resp.JSON(&e))
	require.Equal(t, "invalid_request", e.Error.Code)
	require.Equal(t, "invalid_request_error", e.Error.Type)
	if field != "" {
		require.Contains(t, e.Error.Message, "AuthorisationRequest."+field+"'", "wrong field rejected")
		require.Contains(t, e.Error.Message, "'"+tag+"' tag", "wrong validation rule")
	}
}

// requireUnauthorised asserts the JWT middleware rejected the request.
func requireUnauthorised(t *testing.T, resp *client.Response) {
	t.Helper()
	require.Equal(t, 401, resp.Status, resp.String())
}

// newAuthorisation is a shorthand used by table tests.
var newAuthorisation = testdata.NewAuthorisation

// bearer builds an Authorization header for a raw token value.
func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func repeat(s string, n int) string { return strings.Repeat(s, n) }
