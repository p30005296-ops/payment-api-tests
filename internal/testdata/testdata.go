// Package testdata builds valid requests and holds the sample cards.
//
// NewAuthorisation returns a request the API approves; tests apply small
// mutations to it so each case states only what it changes.
package testdata

import (
	"time"

	"github.com/google/uuid"

	"payment-api-tests/internal/models"
)

// RequestedAtLayout is the datetime format the API validates against.
const RequestedAtLayout = time.RFC3339

type TestCard struct {
	Scheme      string
	Number      string
	ExpiryMonth string
	ExpiryYear  string
	Track2      string
}

var (
	Visa = TestCard{Scheme: "VISA", Number: "4929123123123", ExpiryMonth: "12", ExpiryYear: "2030",
		Track2: "4929123123123=3012300000000000"}
	Mastercard = TestCard{Scheme: "MASTERCARD", Number: "5454545454545454", ExpiryMonth: "12", ExpiryYear: "2030",
		Track2: "5454545454545454=3012300000000000"}
	Amex = TestCard{Scheme: "AMERICAN_EXPRESS", Number: "371449635398431", ExpiryMonth: "12", ExpiryYear: "2030",
		Track2: "371449635398431=3012300000000000"}
)

// Default amounts for each outcome: the API declines amounts ending in 5.
const (
	ApprovedAmount = 1000
	DeclinedAmount = 1005
)

type Mutator func(*models.AuthorisationRequest)

// NewAuthorisation returns a valid MANUAL Mastercard AUTH with fresh unique IDs.
// Mastercard is the default because the documented 13-digit Visa card triggers
// a server error (see BUGS.md).
func NewAuthorisation(mutators ...Mutator) *models.AuthorisationRequest {
	r := &models.AuthorisationRequest{
		TransactionID:     uuid.NewString(),
		SystemTraceNumber: 1,
		IdempotencyKey:    uuid.NewString(),
		Reference:         "test-ref-" + uuid.NewString()[:8],
		Type:              "AUTH",
		AcquirerID:        "ACQ-001",
		RequestedAt:       time.Now().UTC().Format(RequestedAtLayout),
		Amounts:           &models.Amounts{Amount: ApprovedAmount, Currency: "GBP"},
		Merchant: &models.Merchant{
			Mid:          "MID000123",
			Name:         "Test Merchant",
			Domain:       "test-domain",
			CategoryCode: "5999",
		},
		Terminal: &models.Terminal{
			Tid:          "TID00001",
			SerialNumber: "SN-0001",
			Type:         "POS",
			CountryCode:  "826",
			EntryMode:    "MANUAL",
			Capabilities: &models.Capabilities{},
		},
		Card: &models.Card{
			CaptureMethod:      "MANUAL",
			CardNumber:         Mastercard.Number,
			ExpiryMonth:        Mastercard.ExpiryMonth,
			ExpiryYear:         Mastercard.ExpiryYear,
			VerificationMethod: "NONE",
		},
	}
	for _, m := range mutators {
		m(r)
	}
	return r
}

func WithAmount(amount int) Mutator {
	return func(r *models.AuthorisationRequest) { r.Amounts.Amount = amount }
}

func WithType(t string) Mutator {
	return func(r *models.AuthorisationRequest) { r.Type = t }
}

// WithCard keys in the card manually (no Track2).
func WithCard(c TestCard) Mutator {
	return func(r *models.AuthorisationRequest) {
		r.Card.CaptureMethod = "MANUAL"
		r.Card.CardNumber, r.Card.ExpiryMonth, r.Card.ExpiryYear = c.Number, c.ExpiryMonth, c.ExpiryYear
		r.Card.Track2 = ""
	}
}

// WithSwipedCard presents the card via magnetic stripe (MSR), which requires Track2.
func WithSwipedCard(c TestCard) Mutator {
	return func(r *models.AuthorisationRequest) {
		WithCard(c)(r)
		r.Card.CaptureMethod = "MSR"
		r.Card.Track2 = c.Track2
	}
}

func WithCaptureMethod(m string) Mutator {
	return func(r *models.AuthorisationRequest) { r.Card.CaptureMethod = m }
}
