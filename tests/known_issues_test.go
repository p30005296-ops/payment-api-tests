//go:build knownissues

// Tests in this file assert the CORRECT behaviour for defects listed in
// BUGS.md. They fail against the current API and are excluded from the
// default run; run them with: go test -tags knownissues -run KnownIssue ./tests/
package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/models"
	"payment-api-tests/internal/testdata"
)

// BUG-01: 13 digits is the documented minimum PAN length.
func TestKnownIssue_13DigitCardIsProcessed(t *testing.T) {
	t.Parallel()
	for name, mutate := range map[string]testdata.Mutator{
		"documented Visa MANUAL": testdata.WithCard(testdata.Visa),
		"documented Visa MSR":    testdata.WithSwipedCard(testdata.Visa),
		"13-digit Mastercard": func(r *models.AuthorisationRequest) {
			r.Card.CardNumber = "5" + strings.Repeat("4", 12)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(mutate))
			assert.Equal(t, models.OutcomeApproved, resp.Status.Outcome)
		})
	}
}

// BUG-02: the mask must keep the PAN's length (first 6 + one '*' per hidden digit + last 4).
func TestKnownIssue_MaskedPANKeepsCardLength(t *testing.T) {
	t.Parallel()
	for _, pan := range []string{testdata.Amex.Number, testdata.Mastercard.Number, "5" + strings.Repeat("4", 18)} {
		t.Run(fmt.Sprintf("%d_digits", len(pan)), func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(func(r *models.AuthorisationRequest) { r.Card.CardNumber = pan }))
			require.NotNil(t, resp.Card)
			want := pan[:6] + strings.Repeat("*", len(pan)-10) + pan[len(pan)-4:]
			assert.Equal(t, want, resp.Card.MaskedPan)
		})
	}
}

// BUG-03: unknown card ranges must not be labelled as a real scheme.
func TestKnownIssue_SchemeDetectionByBIN(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"9999999999999999": "VISA",             // unassigned range
		"6011111111111117": "VISA",             // Discover range
		"3530111333300000": "AMERICAN_EXPRESS", // JCB range (only 34/37 are Amex)
	}
	for pan, wrongScheme := range cases {
		t.Run(pan, func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(func(r *models.AuthorisationRequest) { r.Card.CardNumber = pan }))
			require.NotNil(t, resp.Card)
			assert.NotEqual(t, wrongScheme, resp.Card.Scheme)
		})
	}
}

// BUG-04
func TestKnownIssue_DeclinedTransactionCannotBeCompleted(t *testing.T) {
	t.Parallel()
	req := newAuthorisation(testdata.WithAmount(testdata.DeclinedAmount))
	require.Equal(t, models.OutcomeDeclined, authorise(t, req).Status.Outcome)

	resp, err := api.Complete(req.TransactionID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Status, 400, "completing a declined transaction must be rejected: %s", resp)
	assert.Less(t, resp.Status, 500, resp.String())
}

// BUG-05
func TestKnownIssue_TransactionCannotBeCompletedTwice(t *testing.T) {
	t.Parallel()
	req := newAuthorisation()
	authorise(t, req)
	complete(t, req.TransactionID)

	resp, err := api.Complete(req.TransactionID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Status, 400, "second completion must be rejected: %s", resp)
	assert.Less(t, resp.Status, 500, resp.String())
}

// BUG-06
func TestKnownIssue_CompleteUnknownTransactionReturns404(t *testing.T) {
	t.Parallel()
	resp, err := api.Complete(newAuthorisation().TransactionID)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.Status, resp.String())
}

// BUG-06
func TestKnownIssue_CompleteMalformedIDReturns400(t *testing.T) {
	t.Parallel()
	resp, err := api.Complete("not-a-uuid")
	require.NoError(t, err)
	assert.Equal(t, 400, resp.Status, resp.String())
}

// BUG-07
func TestKnownIssue_DuplicateTransactionIDIsFlagged(t *testing.T) {
	t.Parallel()
	req := newAuthorisation()
	authorise(t, req)

	replay := authorise(t, req)
	require.NotNil(t, replay.IsDuplicate, "isDuplicate must be present on a replay")
	assert.True(t, *replay.IsDuplicate)
}

// BUG-07
func TestKnownIssue_ReusedIdempotencyKeyIsNotProcessedAsNew(t *testing.T) {
	t.Parallel()
	first := newAuthorisation()
	authorise(t, first)

	second := newAuthorisation(func(r *models.AuthorisationRequest) { r.IdempotencyKey = first.IdempotencyKey })
	resp, err := api.Authorise(second)
	require.NoError(t, err)

	if resp.Status == 200 {
		var out models.AuthorisationResponse
		require.NoError(t, resp.JSON(&out))
		require.NotNil(t, out.IsDuplicate, "reused idempotency key must be detected: %s", resp)
		assert.True(t, *out.IsDuplicate)
		return
	}
	assert.Equal(t, 409, resp.Status, resp.String())
}

// BUG-08
func TestKnownIssue_DeclineCarriesReason(t *testing.T) {
	t.Parallel()
	resp := authorise(t, newAuthorisation(testdata.WithAmount(testdata.DeclinedAmount)))
	assert.NotEmpty(t, resp.Status.Error.Code, "status.error.code")
	assert.NotEmpty(t, resp.Authorisation.Message, "authorisation.message")
}

// BUG-09
func TestKnownIssue_InvalidOrExpiredCardIsRejected(t *testing.T) {
	t.Parallel()
	cases := map[string]testdata.Mutator{
		"expiry month 13": func(r *models.AuthorisationRequest) { r.Card.ExpiryMonth = "13" },
		"expiry month 00": func(r *models.AuthorisationRequest) { r.Card.ExpiryMonth = "00" },
		"expired in 2020": func(r *models.AuthorisationRequest) { r.Card.ExpiryYear = "2020" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := api.Authorise(newAuthorisation(mutate))
			require.NoError(t, err)
			if resp.Status == 200 {
				var out models.AuthorisationResponse
				require.NoError(t, resp.JSON(&out))
				assert.NotEqual(t, models.OutcomeApproved, out.Status.Outcome, "invalid expiry must not be approved")
				return
			}
			assert.Equal(t, 400, resp.Status, resp.String())
		})
	}
}

// BUG-10
func TestKnownIssue_DocumentedEnumValuesAreAccepted(t *testing.T) {
	t.Parallel()
	cases := map[string]testdata.Mutator{
		"type PRE_AUTH":       testdata.WithType("PRE_AUTH"),
		"type CAPTURE":        testdata.WithType("CAPTURE"),
		"capture EMV_CONTACT": testdata.WithCaptureMethod("EMV_CONTACT"),
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := api.Authorise(newAuthorisation(mutate))
			require.NoError(t, err)
			assert.Equal(t, 200, resp.Status, "value is listed in Swagger: %s", resp)
		})
	}
}

// BUG-11
func TestKnownIssue_UnsupportedMethodReturns405(t *testing.T) {
	t.Parallel()
	tok, err := api.Token()
	require.NoError(t, err)
	resp, err := api.Do("GET", "/api/v1/transactions/", nil, bearer(tok))
	require.NoError(t, err)
	assert.Equal(t, 405, resp.Status, resp.String())
}
