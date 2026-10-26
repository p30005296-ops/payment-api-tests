# Defect report

Found while building this suite (API version as deployed on 2026-10-26).
Each defect with an ID has a test in [`tests/known_issues_test.go`](tests/known_issues_test.go)
that asserts the **correct** behaviour and currently fails:

```bash
go test -tags knownissues -run KnownIssue -v ./tests/
```

| ID | Severity | Summary |
|----|----------|---------|
| [BUG-01](#bug-01) | High | 13-digit card numbers (incl. the documented Visa test card) crash the server with `500` |
| [BUG-02](#bug-02) | Medium | Masked PAN has the wrong length |
| [BUG-03](#bug-03) | Medium | Card scheme detection mislabels unknown ranges as VISA / AMERICAN_EXPRESS |
| [BUG-04](#bug-04) | High | A declined transaction can be completed |
| [BUG-05](#bug-05) | Medium | A transaction can be completed repeatedly |
| [BUG-06](#bug-06) | Medium | Completing an unknown or malformed ID returns `500` instead of `404`/`400` |
| [BUG-07](#bug-07) | Medium | Duplicates are not flagged; idempotency key is not enforced |
| [BUG-08](#bug-08) | Low | Declines carry no reason |
| [BUG-09](#bug-09) | Medium | Invalid or expired card expiry dates are approved |
| [BUG-10](#bug-10) | Low | Swagger enums disagree with the server |
| [BUG-11](#bug-11) | Low | Unsupported HTTP method returns `500` instead of `405` |
| [Other](#documentation-and-consistency-issues) | Low | Documentation and consistency issues |

---

### BUG-01
**13-digit card numbers crash the server.**
`Card.CardNumber` is documented as min length 13 and the README's own Visa test card
(`4929 1231 2312 3`) is 13 digits.

- **Steps:** authorise any request with a 13-digit `cardNumber`, e.g. `4929123123123` or `5444444444444`, any capture method.
- **Expected:** request processed (approved/declined) like 14–20 digit cards.
- **Actual:** `500 Internal Server Error` (plain text). Every length 14–20 works.
- **Likely cause:** the masking routine (see BUG-02) repeats `*` `len-14` times; a negative count panics for 13 digits.

### BUG-02
**Masked PAN has the wrong number of mask characters.**

- **Steps:** authorise card `5454545454545454` (16 digits).
- **Expected:** `545454******5454` (first 6, 6 masked, last 4: same length as the PAN).
- **Actual:** `545454**5454`. Mask count is `len-14` (15 digits → 1 `*`, 14 digits → none).
- **Impact:** receipts/UI show a PAN of the wrong length; root cause of BUG-01.

### BUG-03
**Card scheme detection is wrong for non-Visa/Mastercard ranges.**

- **Steps:** authorise `9999999999999999`, `6011111111111117` (Discover range), `3530111333300000` (JCB range).
- **Expected:** correct scheme or `UNKNOWN`/rejection.
- **Actual:** `VISA`, `VISA`, `AMERICAN_EXPRESS`. Any PAN starting with `3` is treated as Amex (only `34`/`37` are); unrecognised ranges default to `VISA`.

### BUG-04
**A declined transaction can be completed.** The spec says only authorised transactions can be completed.

- **Steps:** authorise amount `1005` (outcome `DECLINED`, responseCode `05`), then `POST /api/v1/transactions/{id}/complete`.
- **Expected:** `4xx` rejection.
- **Actual:** `200`, `status.outcome = COMPLETED` (while `responseCode` stays `05`).

### BUG-05
**A transaction can be completed more than once.**

- **Steps:** authorise, complete, complete again.
- **Expected:** second call rejected (`409`/`422`), or an explicit idempotent replay marker.
- **Actual:** `200 COMPLETED` every time with no indication it is a repeat.

### BUG-06
**Wrong status codes on `/complete` for bad IDs.**

| Request | Expected | Actual |
|---|---|---|
| unknown UUID | `404` | `500 {"code":"internal_server_error","message":"transaction not found"}` |
| `not-a-uuid` | `400` | `500 Internal Server Error` |

Client errors are reported as server errors, which breaks retry logic and alerting.

### BUG-07
**Duplicate detection / idempotency not visible or enforced.** The response model has `isDuplicate`.

- Re-sending the same `transactionID` returns `200` with the stored transaction, but `isDuplicate` is never set.
- Re-using an `idempotencyKey` with a new `transactionID` is processed as a brand-new approved transaction.
- **Expected:** `isDuplicate: true` (or `409`) in both cases.

### BUG-08
**Declines carry no reason.** For amount `1005`: `status.error` is `{}`, `authorisation.message` and `entity` are empty, `references` is `{}`.
Approved responses populate all of these. Expected a decline code/message (e.g. `"Do not honour"`).

### BUG-09
**Card expiry is only length-checked.** Approved:
`expiryMonth = "13"`, `expiryMonth = "00"`, `expiryYear = "2020"` (expired).
Expected `400` validation error or a decline.

### BUG-10
**Swagger and server disagree on enums and required fields.**

| Field | Swagger | Server |
|---|---|---|
| `type` | AUTH, PRE_AUTH, REFUND, COMPLETE, CAPTURE, INCREMENTAL, REVERSAL | rejects PRE_AUTH, COMPLETE, CAPTURE |
| `card.captureMethod` | 8 values incl. EMV_CONTACT, EMV_CONTACTLESS | only MSR, MANUAL, ECOMMERCE |
| `card.verificationMethod` | required | optional |
| `terminal.capabilities` | required | optional |
| length/max rules (tid, mid, amount max…) | not documented | enforced |

### BUG-11
`GET /api/v1/transactions/` returns `500 Internal Server Error`; expected `405 Method Not Allowed`.

---

### Documentation and consistency issues

- **Field name:** README documents `Merchant.DomainID`; the JSON key is `domain` (validation errors still say `DomainID`).
- **Inconsistent error envelopes:**
  JWT errors → `{"error": true, "msg": "..."}`;
  handler errors → `{"error": {"code","message","param","type"}}`;
  some 500s → plain text `Internal Server Error`.
- **Internal details leaked:** validation messages expose Go struct paths and validator tags
  (`Key: 'AuthorisationRequest.Terminal.Tid' Error:Field validation for 'Tid' failed on the 'min' tag`),
  and JWT errors expose parser internals (`could not JSON decode header: invalid character...`).
- **Swagger metadata** is template boilerplate: title `Template Go API`, description `This is a sample swagger`.
- **`requestedAt`** far in the future (e.g. `2099-01-01T00:00:00Z`) is accepted. Clock-skew limits are usually expected.
- **Rate limiting:** bursts of parallel requests receive `503 Service Unavailable` from the hosting layer.
  The client retries 502/503/504 with backoff so the suite stays deterministic.

### Verified as working correctly
Login validation and generic auth errors (no user enumeration); 5-minute JWT lifetime; rejection of
missing, malformed, tampered and `alg:none` tokens; the amount last-digit decision rule; every documented
length/format/required rule; the Track2 `required_if MSR` / `excluded_unless MSR` rule; PAN never returned in full.
