// Package models mirrors the payment API's request and response payloads.
//
// Every field is `omitempty` so a test can drop a field simply by leaving it at
// its zero value. IDs are plain strings so tests can send malformed values.
package models

type LoginRequest struct {
	Email        string `json:"email,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type AuthorisationRequest struct {
	TransactionID     string      `json:"transactionID,omitempty"`
	SystemTraceNumber int         `json:"systemTraceNumber,omitempty"`
	IdempotencyKey    string      `json:"idempotencyKey,omitempty"`
	Reference         string      `json:"reference,omitempty"`
	Type              string      `json:"type,omitempty"`
	AcquirerID        string      `json:"acquirerID,omitempty"`
	RequestedAt       string      `json:"requestedAt,omitempty"`
	Amounts           *Amounts    `json:"amounts,omitempty"`
	Merchant          *Merchant   `json:"merchant,omitempty"`
	Terminal          *Terminal   `json:"terminal,omitempty"`
	Card              *Card       `json:"card,omitempty"`
	References        *References `json:"references,omitempty"`
}

type Amounts struct {
	Amount   int    `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
}

type Merchant struct {
	Mid          string `json:"mid,omitempty"`
	Name         string `json:"name,omitempty"`
	Domain       string `json:"domain,omitempty"` // documented as "DomainID"
	CategoryCode string `json:"categoryCode,omitempty"`
}

type Terminal struct {
	Tid             string        `json:"tid,omitempty"`
	SerialNumber    string        `json:"serialNumber,omitempty"`
	SoftwareVersion string        `json:"softwareVersion,omitempty"`
	Type            string        `json:"type,omitempty"`
	CountryCode     string        `json:"countryCode,omitempty"`
	EntryMode       string        `json:"entryMode,omitempty"`
	Capabilities    *Capabilities `json:"capabilities,omitempty"`
}

type Capabilities struct {
	Supported   string `json:"supported,omitempty"`
	Used        string `json:"used,omitempty"`
	PartialAuth bool   `json:"partialAuth,omitempty"`
}

type Card struct {
	CaptureMethod      string `json:"captureMethod,omitempty"`
	Track2             string `json:"track2,omitempty"`
	CardNumber         string `json:"cardNumber,omitempty"`
	ExpiryMonth        string `json:"expiryMonth,omitempty"`
	ExpiryYear         string `json:"expiryYear,omitempty"`
	VerificationMethod string `json:"verificationMethod,omitempty"`
}

type References struct {
	SchemeReference    string `json:"schemeReference,omitempty"`
	AcquirerReference  string `json:"acquirerReference,omitempty"`
	RetrievalReference string `json:"retrievalReference,omitempty"`
	TxnLifecycleID     string `json:"txnLifecycleID,omitempty"`
}

type AuthorisationResponse struct {
	TransactionID     string        `json:"transactionID"`
	Reference         string        `json:"reference"`
	SystemTraceNumber int           `json:"systemTraceNumber"`
	RespondedAt       string        `json:"respondedAt"`
	IsDuplicate       *bool         `json:"isDuplicate"`
	Status            Status        `json:"status"`
	Authorisation     AuthResult    `json:"authorisation"`
	References        References    `json:"references"`
	Card              *CardResponse `json:"card"`
}

type Status struct {
	Outcome string      `json:"outcome"`
	Error   StatusError `json:"error"`
}

type StatusError struct {
	Code                  string `json:"code"`
	Message               string `json:"message"`
	ExtendedDeclineReason string `json:"extendedDeclineReason"`
}

type AuthResult struct {
	ResponseCode string `json:"responseCode"`
	Code         string `json:"code"`
	Message      string `json:"message"`
	Entity       string `json:"entity"`
}

type CardResponse struct {
	TokenID   string `json:"tokenID"`
	Scheme    string `json:"scheme"`
	MaskedPan string `json:"maskedPan"`
}

// APIError is the error envelope used by the login and transaction handlers.
type APIError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Outcomes and response codes returned by the API.
const (
	OutcomeApproved  = "APPROVED"
	OutcomeDeclined  = "DECLINED"
	OutcomeCompleted = "COMPLETED"

	ResponseCodeApproved = "00"
	ResponseCodeDeclined = "05"
)
