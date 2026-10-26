package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/models"
	"payment-api-tests/internal/testdata"
)

func TestAuthorise_Approved_ResponseContract(t *testing.T) {
	t.Parallel()

	req := newAuthorisation(func(r *models.AuthorisationRequest) { r.SystemTraceNumber = 4242 })
	before := time.Now().Add(-time.Minute)
	resp := authorise(t, req)

	assert.Equal(t, req.TransactionID, resp.TransactionID, "transactionID must be echoed")
	assert.Equal(t, req.Reference, resp.Reference, "reference must be echoed")
	assert.Equal(t, req.SystemTraceNumber, resp.SystemTraceNumber, "systemTraceNumber must be echoed")

	assert.Equal(t, models.OutcomeApproved, resp.Status.Outcome)
	assert.Empty(t, resp.Status.Error.Code)
	assert.Equal(t, models.ResponseCodeApproved, resp.Authorisation.ResponseCode)
	assert.NotEmpty(t, resp.Authorisation.Code, "approval code expected")
	assert.Equal(t, "Transaction approved", resp.Authorisation.Message)

	respondedAt, err := time.Parse(time.RFC3339Nano, resp.RespondedAt)
	require.NoError(t, err, "respondedAt must be RFC3339")
	assert.True(t, respondedAt.After(before), "respondedAt %s should be current", resp.RespondedAt)

	require.NotNil(t, resp.Card)
	assert.Equal(t, testdata.Mastercard.Scheme, resp.Card.Scheme)
	assert.NotEmpty(t, resp.Card.TokenID)
}

// The full card number must never be returned; only first 6 and last 4 digits.
func TestAuthorise_MasksPAN(t *testing.T) {
	t.Parallel()

	for _, card := range []testdata.TestCard{testdata.Mastercard, testdata.Amex} {
		t.Run(card.Scheme, func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(testdata.WithCard(card)))

			require.NotNil(t, resp.Card)
			pan := card.Number
			assert.Equal(t, card.Scheme, resp.Card.Scheme)
			assert.NotContains(t, resp.Card.MaskedPan, pan)
			assert.Regexp(t, fmt.Sprintf(`^%s\*+%s$`, pan[:6], pan[len(pan)-4:]), resp.Card.MaskedPan)
		})
	}
}

// Business rule: an amount whose last digit is 5 is declined, anything else is approved.
func TestAuthorise_DecisionByLastDigitOfAmount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		amount  int
		outcome string
		code    string
	}{
		{1, models.OutcomeApproved, models.ResponseCodeApproved},         // minimum amount
		{4, models.OutcomeApproved, models.ResponseCodeApproved},         // neighbour below 5
		{6, models.OutcomeApproved, models.ResponseCodeApproved},         // neighbour above 5
		{50, models.OutcomeApproved, models.ResponseCodeApproved},        // 5 not in last position
		{1000, models.OutcomeApproved, models.ResponseCodeApproved},      // typical
		{999999999, models.OutcomeApproved, models.ResponseCodeApproved}, // maximum amount
		{5, models.OutcomeDeclined, models.ResponseCodeDeclined},
		{15, models.OutcomeDeclined, models.ResponseCodeDeclined},
		{1005, models.OutcomeDeclined, models.ResponseCodeDeclined},
		{999999995, models.OutcomeDeclined, models.ResponseCodeDeclined}, // largest declinable amount
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("amount_%d", tc.amount), func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(testdata.WithAmount(tc.amount)))
			assert.Equal(t, tc.outcome, resp.Status.Outcome)
			assert.Equal(t, tc.code, resp.Authorisation.ResponseCode)
		})
	}
}

func TestAuthorise_SupportedCardsAndCaptureMethods(t *testing.T) {
	t.Parallel()

	cases := map[string]testdata.Mutator{
		"Mastercard MANUAL":    testdata.WithCard(testdata.Mastercard),
		"Amex MANUAL":          testdata.WithCard(testdata.Amex),
		"Mastercard MSR":       testdata.WithSwipedCard(testdata.Mastercard),
		"Amex MSR":             testdata.WithSwipedCard(testdata.Amex),
		"Mastercard ECOMMERCE": testdata.WithCaptureMethod("ECOMMERCE"),
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(mutate))
			assert.Equal(t, models.OutcomeApproved, resp.Status.Outcome)
		})
	}
}

func TestAuthorise_SupportedTransactionTypes(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"AUTH", "REFUND", "INCREMENTAL", "REVERSAL"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			resp := authorise(t, newAuthorisation(testdata.WithType(typ)))
			assert.Equal(t, models.OutcomeApproved, resp.Status.Outcome)
		})
	}
}

func TestAuthorise_OptionalFieldsCanBeSet(t *testing.T) {
	t.Parallel()

	req := newAuthorisation(func(r *models.AuthorisationRequest) {
		r.Terminal.SoftwareVersion = "1.2.3"
		r.References = &models.References{
			SchemeReference:    "scheme-1",
			AcquirerReference:  "acq-1",
			RetrievalReference: "rrn-1",
			TxnLifecycleID:     "life-1",
		}
	})
	resp := authorise(t, req)
	assert.Equal(t, models.OutcomeApproved, resp.Status.Outcome)
}

func TestAuthorise_RequestedAtAcceptsTimezoneOffsets(t *testing.T) {
	t.Parallel()

	for _, loc := range []*time.Location{time.UTC, time.FixedZone("plus2", 2*3600), time.FixedZone("minus5", -5*3600)} {
		t.Run(loc.String(), func(t *testing.T) {
			t.Parallel()
			req := newAuthorisation(func(r *models.AuthorisationRequest) {
				r.RequestedAt = time.Now().In(loc).Format(testdata.RequestedAtLayout)
			})
			authorise(t, req)
		})
	}
}
