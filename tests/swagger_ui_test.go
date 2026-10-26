package tests

import (
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

// Browser smoke test: the published API docs load and describe every endpoint.
// Skipped with -short so the API suite can run without a browser installed.
func TestSwaggerUI_DocumentsAllEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("browser test skipped in -short mode")
	}
	t.Parallel()

	opts := playwright.BrowserTypeLaunchOptions{}
	if cfg.BrowserChannel != "" {
		opts.Channel = playwright.String(cfg.BrowserChannel)
	}
	browser, err := pw.Chromium.Launch(opts)
	require.NoError(t, err, "launch Chromium (install it, or set PW_BROWSER_CHANNEL=msedge|chrome)")
	t.Cleanup(func() { browser.Close() })

	page, err := browser.NewPage()
	require.NoError(t, err)

	resp, err := page.Goto(cfg.BaseURL + "/swagger/index.html")
	require.NoError(t, err)
	require.Equal(t, 200, resp.Status())

	expect := playwright.NewPlaywrightAssertions()
	for _, path := range []string{"/login", "/api/v1/transactions/", "/api/v1/transactions/{id}/complete"} {
		op := page.Locator(`.opblock-post .opblock-summary-path[data-path="` + path + `"]`)
		require.NoError(t, expect.Locator(op).ToBeVisible(), "POST %s missing from Swagger UI", path)
	}

	// Bearer auth is offered, and the request model is published.
	require.NoError(t, expect.Locator(page.Locator("button.authorize")).ToBeVisible())
	require.NoError(t, expect.Locator(page.GetByText("models.AuthorisationRequest", playwright.PageGetByTextOptions{
		Exact: playwright.Bool(true),
	})).ToBeVisible())
}
