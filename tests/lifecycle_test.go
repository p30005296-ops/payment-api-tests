package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"payment-api-tests/internal/models"
	"payment-api-tests/internal/testdata"
)

func TestLifecycle_AuthoriseThenComplete(t *testing.T) {
	t.Parallel()

	req := newAuthorisation()
	auth := authorise(t, req)
	assert.Equal(t, models.OutcomeApproved, auth.Status.Outcome)

	done := complete(t, req.TransactionID)

	assert.Equal(t, models.OutcomeCompleted, done.Status.Outcome)
	assert.Equal(t, models.ResponseCodeApproved, done.Authorisation.ResponseCode)
	assert.Equal(t, "Transaction completed", done.Authorisation.Message)
	// Completion must refer to the same transaction, not create a new one.
	assert.Equal(t, auth.TransactionID, done.TransactionID)
	assert.Equal(t, auth.Reference, done.Reference)
	assert.Equal(t, auth.SystemTraceNumber, done.SystemTraceNumber)
	assert.Equal(t, auth.Authorisation.Code, done.Authorisation.Code)
	assert.Equal(t, auth.Card, done.Card)
}

func TestLifecycle_CompleteEachSupportedCard(t *testing.T) {
	t.Parallel()

	for _, card := range []testdata.TestCard{testdata.Mastercard, testdata.Amex} {
		t.Run(card.Scheme, func(t *testing.T) {
			t.Parallel()
			req := newAuthorisation(testdata.WithSwipedCard(card))
			authorise(t, req)
			assert.Equal(t, models.OutcomeCompleted, complete(t, req.TransactionID).Status.Outcome)
		})
	}
}

func TestLifecycle_DeclinedTransactionResponse(t *testing.T) {
	t.Parallel()

	resp := authorise(t, newAuthorisation(testdata.WithAmount(testdata.DeclinedAmount)))

	assert.Equal(t, models.OutcomeDeclined, resp.Status.Outcome)
	assert.Equal(t, models.ResponseCodeDeclined, resp.Authorisation.ResponseCode)
	assert.Empty(t, resp.Authorisation.Code, "a declined transaction must not carry an approval code")
}
