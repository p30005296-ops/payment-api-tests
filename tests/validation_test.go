package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/models"
	"payment-api-tests/internal/testdata"
)

type Req = models.AuthorisationRequest

// Each documented field rule is exercised just outside its limit (rejected)
// and, where it has a limit, exactly on it (accepted).
func TestAuthorise_FieldValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Req)
		field  string // empty = request must be accepted
		tag    string
	}{
		// Top-level fields
		{"transactionID missing", func(r *Req) { r.TransactionID = "" }, "TransactionID", "required"},
		{"systemTraceNumber missing", func(r *Req) { r.SystemTraceNumber = 0 }, "SystemTraceNumber", "required"},
		{"systemTraceNumber negative", func(r *Req) { r.SystemTraceNumber = -1 }, "SystemTraceNumber", "min"},
		{"idempotencyKey missing", func(r *Req) { r.IdempotencyKey = "" }, "IdempotencyKey", "required"},
		{"reference missing", func(r *Req) { r.Reference = "" }, "Reference", "required"},
		{"type missing", func(r *Req) { r.Type = "" }, "Type", "required"},
		{"type unknown", func(r *Req) { r.Type = "PURCHASE" }, "Type", "oneof"},
		{"type lowercase", func(r *Req) { r.Type = "auth" }, "Type", "oneof"},
		{"acquirerID missing", func(r *Req) { r.AcquirerID = "" }, "AcquirerID", "required"},
		{"requestedAt missing", func(r *Req) { r.RequestedAt = "" }, "RequestedAt", "required"},
		{"requestedAt without T and zone", func(r *Req) { r.RequestedAt = "2026-01-02 15:04:05" }, "RequestedAt", "datetime"},
		{"requestedAt date only", func(r *Req) { r.RequestedAt = "2026-01-02" }, "RequestedAt", "datetime"},
		{"requestedAt without zone", func(r *Req) { r.RequestedAt = "2026-01-02T15:04:05" }, "RequestedAt", "datetime"},

		// Amounts
		{"amounts missing", func(r *Req) { r.Amounts = nil }, "Amounts", "required"},
		{"amount zero", func(r *Req) { r.Amounts.Amount = 0 }, "Amounts.Amount", "required"},
		{"amount negative", func(r *Req) { r.Amounts.Amount = -1 }, "Amounts.Amount", "min"},
		{"amount above max", func(r *Req) { r.Amounts.Amount = 1_000_000_000 }, "Amounts.Amount", "max"},
		{"currency missing", func(r *Req) { r.Amounts.Currency = "" }, "Amounts.Currency", "required"},
		{"currency 2 chars", func(r *Req) { r.Amounts.Currency = "GB" }, "Amounts.Currency", "len"},
		{"currency 4 chars", func(r *Req) { r.Amounts.Currency = "GBPX" }, "Amounts.Currency", "len"},

		// Merchant
		{"merchant missing", func(r *Req) { r.Merchant = nil }, "Merchant", "required"},
		{"mid missing", func(r *Req) { r.Merchant.Mid = "" }, "Merchant.Mid", "required"},
		{"mid 20 chars", func(r *Req) { r.Merchant.Mid = repeat("M", 20) }, "", ""},
		{"mid 21 chars", func(r *Req) { r.Merchant.Mid = repeat("M", 21) }, "Merchant.Mid", "max"},
		{"name missing", func(r *Req) { r.Merchant.Name = "" }, "Merchant.Name", "required"},
		{"name 50 chars", func(r *Req) { r.Merchant.Name = repeat("N", 50) }, "", ""},
		{"name 51 chars", func(r *Req) { r.Merchant.Name = repeat("N", 51) }, "Merchant.Name", "max"},
		{"domain missing", func(r *Req) { r.Merchant.Domain = "" }, "Merchant.DomainID", "required"},
		{"domain 50 chars", func(r *Req) { r.Merchant.Domain = repeat("D", 50) }, "", ""},
		{"domain 51 chars", func(r *Req) { r.Merchant.Domain = repeat("D", 51) }, "Merchant.DomainID", "max"},
		{"categoryCode missing", func(r *Req) { r.Merchant.CategoryCode = "" }, "Merchant.CategoryCode", "required"},
		{"categoryCode 3 chars", func(r *Req) { r.Merchant.CategoryCode = "599" }, "Merchant.CategoryCode", "len"},
		{"categoryCode 5 chars", func(r *Req) { r.Merchant.CategoryCode = "59999" }, "Merchant.CategoryCode", "len"},

		// Terminal
		{"terminal missing", func(r *Req) { r.Terminal = nil }, "Terminal", "required"},
		{"tid missing", func(r *Req) { r.Terminal.Tid = "" }, "Terminal.Tid", "required"},
		{"tid 7 chars", func(r *Req) { r.Terminal.Tid = repeat("T", 7) }, "Terminal.Tid", "min"},
		{"tid 8 chars", func(r *Req) { r.Terminal.Tid = repeat("T", 8) }, "", ""},
		{"tid 15 chars", func(r *Req) { r.Terminal.Tid = repeat("T", 15) }, "", ""},
		{"tid 16 chars", func(r *Req) { r.Terminal.Tid = repeat("T", 16) }, "Terminal.Tid", "max"},
		{"serialNumber missing", func(r *Req) { r.Terminal.SerialNumber = "" }, "Terminal.SerialNumber", "required"},
		{"terminal type missing", func(r *Req) { r.Terminal.Type = "" }, "Terminal.Type", "required"},
		{"countryCode missing", func(r *Req) { r.Terminal.CountryCode = "" }, "Terminal.CountryCode", "required"},
		{"countryCode alpha", func(r *Req) { r.Terminal.CountryCode = "GBR" }, "Terminal.CountryCode", "numeric"},
		{"countryCode 2 digits", func(r *Req) { r.Terminal.CountryCode = "82" }, "Terminal.CountryCode", "len"},
		{"countryCode 4 digits", func(r *Req) { r.Terminal.CountryCode = "8260" }, "Terminal.CountryCode", "len"},
		{"entryMode missing", func(r *Req) { r.Terminal.EntryMode = "" }, "Terminal.EntryMode", "required"},

		// Card
		{"card missing", func(r *Req) { r.Card = nil }, "Card", "required"},
		{"captureMethod missing", func(r *Req) { r.Card.CaptureMethod = "" }, "Card.CaptureMethod", "required"},
		{"captureMethod unknown", func(r *Req) { r.Card.CaptureMethod = "CHIP" }, "Card.CaptureMethod", "oneof"},
		{"MSR without track2", func(r *Req) { r.Card.CaptureMethod = "MSR" }, "Card.Track2", "required_if"},
		{"MANUAL with track2", func(r *Req) { r.Card.Track2 = testdata.Mastercard.Track2 }, "Card.Track2", "excluded_unless"},
		{"ECOMMERCE with track2", func(r *Req) {
			r.Card.CaptureMethod, r.Card.Track2 = "ECOMMERCE", testdata.Mastercard.Track2
		}, "Card.Track2", "excluded_unless"},
		{"MSR track2 50 chars", func(r *Req) { r.Card.CaptureMethod, r.Card.Track2 = "MSR", repeat("5", 50) }, "", ""},
		{"MSR track2 51 chars", func(r *Req) { r.Card.CaptureMethod, r.Card.Track2 = "MSR", repeat("5", 51) }, "Card.Track2", "max"},
		{"cardNumber missing", func(r *Req) { r.Card.CardNumber = "" }, "Card.CardNumber", "required"},
		{"cardNumber 12 digits", func(r *Req) { r.Card.CardNumber = repeat("5", 12) }, "Card.CardNumber", "min"},
		// The 13-digit lower bound passes validation but crashes the server: BUG-01.
		{"cardNumber 14 digits", func(r *Req) { r.Card.CardNumber = "5" + repeat("4", 13) }, "", ""},
		{"cardNumber 20 digits", func(r *Req) { r.Card.CardNumber = "5" + repeat("4", 19) }, "", ""},
		{"cardNumber 21 digits", func(r *Req) { r.Card.CardNumber = repeat("5", 21) }, "Card.CardNumber", "max"},
		{"cardNumber with letters", func(r *Req) { r.Card.CardNumber = "5454abcd54545454" }, "Card.CardNumber", "numeric"},
		{"cardNumber with spaces", func(r *Req) { r.Card.CardNumber = "5454 5454 5454 5454" }, "Card.CardNumber", "numeric"},
		{"expiryMonth missing", func(r *Req) { r.Card.ExpiryMonth = "" }, "Card.ExpiryMonth", "required"},
		{"expiryMonth 1 digit", func(r *Req) { r.Card.ExpiryMonth = "1" }, "Card.ExpiryMonth", "len"},
		{"expiryMonth 3 digits", func(r *Req) { r.Card.ExpiryMonth = "012" }, "Card.ExpiryMonth", "len"},
		{"expiryMonth letters", func(r *Req) { r.Card.ExpiryMonth = "ab" }, "Card.ExpiryMonth", "numeric"},
		{"expiryYear missing", func(r *Req) { r.Card.ExpiryYear = "" }, "Card.ExpiryYear", "required"},
		{"expiryYear 2 digits", func(r *Req) { r.Card.ExpiryYear = "30" }, "Card.ExpiryYear", "len"},
		{"expiryYear 5 digits", func(r *Req) { r.Card.ExpiryYear = "20300" }, "Card.ExpiryYear", "len"},
		{"expiryYear letters", func(r *Req) { r.Card.ExpiryYear = "abcd" }, "Card.ExpiryYear", "numeric"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := newAuthorisation(tc.mutate)
			resp, err := api.Authorise(req)
			require.NoError(t, err)

			if tc.field == "" {
				require.Equal(t, 200, resp.Status, resp.String())
				return
			}
			requireValidationError(t, resp, tc.field, tc.tag)
		})
	}
}

// Bodies that cannot be decoded are rejected before field validation.
func TestAuthorise_UndecodableBody(t *testing.T) {
	t.Parallel()

	valid := newAuthorisation()
	cases := map[string]any{
		"malformed JSON":          `{"transactionID":`,
		"empty body":              "",
		"JSON array":              `[]`,
		"transactionID not UUID":  newAuthorisation(func(r *Req) { r.TransactionID = "txn-123" }),
		"idempotencyKey not UUID": newAuthorisation(func(r *Req) { r.IdempotencyKey = "key-123" }),
		"amount as string": map[string]any{
			"transactionID": valid.TransactionID, "idempotencyKey": valid.IdempotencyKey,
			"amounts": map[string]any{"amount": "1000", "currency": "GBP"},
		},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := api.Authorise(body)
			require.NoError(t, err)
			requireValidationError(t, resp, "", "")

			var e models.APIError
			require.NoError(t, resp.JSON(&e))
			assert.Equal(t, "Invalid request body", e.Error.Message)
		})
	}
}

// An empty object reports every missing required field in one response.
func TestAuthorise_EmptyObjectListsAllRequiredFields(t *testing.T) {
	t.Parallel()

	resp, err := api.Authorise(`{}`)
	require.NoError(t, err)
	requireValidationError(t, resp, "", "")

	var e models.APIError
	require.NoError(t, resp.JSON(&e))
	for _, f := range []string{"TransactionID", "SystemTraceNumber", "IdempotencyKey", "Reference", "Type",
		"AcquirerID", "RequestedAt", "Amounts", "Merchant", "Terminal", "Card"} {
		assert.Contains(t, e.Error.Message, "AuthorisationRequest."+f+"'")
	}
}
