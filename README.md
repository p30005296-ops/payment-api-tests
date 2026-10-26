# Payment API Test Automation

An automated test suite for a card-payment gateway API. It covers authentication, transaction authorisation, completion and request validation.

It is written in **Go**, using **Playwright** ([playwright-go](https://github.com/mxschmitt/playwright-go)).
Playwright's `APIRequestContext` drives every API call, and a Chromium smoke test checks the published Swagger UI.

**About 140 test cases run in about 10 seconds.** **11 defects** were found and are documented in **[BUGS.md](BUGS.md)**.

---

## Quick start

**Prerequisites:** [Go 1.27+](https://go.dev/dl/) and internet access. Node.js is *not* required.

```bash
git clone <this-repo-url> payment-api-tests
cd payment-api-tests

go mod download
go run github.com/mxschmitt/playwright-go/cmd/playwright install --with-deps chromium  # driver + browser (one-off)
cp .env.example .env    # Windows: copy .env.example .env, then fill in the test credentials

go test ./...
```

If `make` is available, `make setup` and `make test` do the same.

> To skip the Chromium download, run `go test -short ./...` (API tests only).
> Alternatively, set `PW_BROWSER_CHANNEL=msedge` or `PW_BROWSER_CHANNEL=chrome` to use a browser that is already installed.

## Running tests

| Goal | Command | Make |
|---|---|---|
| Full regression suite (API + Swagger UI) | `go test ./...` | `make test` |
| API tests only, no browser | `go test -short ./...` | `make test-api` |
| One area | `go test ./tests/ -run TestAuthorise_FieldValidation -v` | |
| **Known defects** (expected to fail) | `go test -tags knownissues -run KnownIssue -v ./tests/` | `make test-known-issues` |
| Token expiry (waits ~5 min) | `go test -tags slow -run Expiry -v -timeout 10m ./tests/` | `make test-slow` |
| JUnit XML report → `reports/` | | `make report` |

Use `-count=1` to bypass Go's test cache when re-running against the live API.

## Configuration

Settings come from environment variables. A `.env` file in the repo root is also read; real environment variables take precedence.

| Variable | Default | Purpose |
|---|---|---|
| `API_BASE_URL` | the hosted test gateway | API under test |
| `API_EMAIL` | *(required)* | Login email |
| `API_CLIENT_SECRET` | *(required)* | Login secret |
| `API_TIMEOUT` | `30s` | Per-request timeout |
| `PW_BROWSER_CHANNEL` | *(bundled Chromium)* | `msedge` / `chrome` for the UI test |

Use the test login credentials supplied with the task brief. They are deliberately not committed. `.env` is git-ignored.

## Project structure

```
internal/
  config/     env + .env loading
  client/     Playwright APIRequestContext wrapper: Login / Authorise / Complete,
              cached JWT with auto-refresh, retry on 502/503/504
  models/     request/response structs mirroring the API
  testdata/   valid-request builder + sample cards
tests/
  main_test.go          shared Playwright + client setup, assertion helpers
  auth_test.go          login success, wrong credentials, input validation
  authorise_test.go     response contract, approve/decline rule, cards, types, PAN masking
  lifecycle_test.go     authorise → complete, declined state
  validation_test.go    table-driven boundaries for every documented field rule
  security_test.go      missing / malformed / tampered / alg:none tokens on every endpoint
  swagger_ui_test.go    browser smoke test of the API docs
  known_issues_test.go  [knownissues] correct-behaviour tests for open defects
  token_expiry_test.go  [slow] token rejected after its 5-minute lifetime
BUGS.md                 defect report
```

## Coverage

| Area | What is checked |
|---|---|
| **Authentication** | Valid login returns a JWT with `email` and a 5-minute `exp`. Wrong email or secret both give `401` with the same message (no user enumeration). Email format and secret length are validated. Malformed JSON is rejected. |
| **Authorisation** | Echoed `transactionID`, `reference` and `systemTraceNumber`. RFC 3339 `respondedAt`. Approval code and response codes `00`/`05`. Card scheme. PAN never returned in full. |
| **Business rule** | Approve or decline by the last digit of the amount: 1, 4, 6, 50, 1000, max → approved; 5, 15, 1005, 999999995 → declined. |
| **Variants** | Mastercard and Amex via MANUAL, MSR (with Track2) and ECOMMERCE. Types AUTH, REFUND, INCREMENTAL, REVERSAL. Optional fields. Time-zone offsets in `requestedAt`. |
| **Lifecycle** | Authorise then complete gives `COMPLETED` for the same transaction. Declined state. |
| **Validation** | 70+ cases: every required field, min/max/length/numeric/oneof rule on both sides of each boundary, the Track2 `required_if` / `excluded_unless` rule, undecodable bodies. |
| **Security** | Both transaction endpoints reject: no header, missing `Bearer`, empty or garbage token, tampered signature or payload, `alg:none`, expired token. |
| **UI** | Swagger UI loads, lists all three POST endpoints, offers Bearer auth and publishes the request model. |

## Design decisions

- **Playwright for API testing.** `APIRequestContext` provides base URL, default headers and timeouts. The same Playwright instance runs the browser test, so there is one dependency and one setup step.
- **Valid by default, mutate one thing.** `testdata.NewAuthorisation()` builds a request the API approves, with fresh UUIDs. Each test changes only the field under test, so its intent is clear in one line:
  ```go
  newAuthorisation(testdata.WithAmount(1005))
  newAuthorisation(func(r *Req) { r.Terminal.Tid = repeat("T", 16) })
  ```
- **Table-driven tests** for validation, decision rules and security. Adding a case is one line.
- **Precise assertions.** Validation tests check the status code *and* that the error names the expected field and rule (e.g. `Terminal.Tid` / `max`). This means a request rejected for the wrong reason fails the test.
- **Independent, parallel tests.** Each test creates its own transactions, so all tests run with `t.Parallel()` and there's no shared state or ordering.
- **Token management.** The client caches the JWT and logs in again 30 seconds before `exp`, so long runs never fail on the 5-minute expiry.
- **Resilience without masking bugs.** Only infrastructure responses (`502`/`503`/`504`) are retried, because the hosting layer rate-limits bursts. Application `500`s are never retried; several of them are real defects.
- **Known defects are kept apart from the regression suite.** The regression suite stays green and reliable for CI. Tests that assert the *correct* behaviour for open bugs sit behind the `knownissues` build tag; each maps to an entry in [BUGS.md](BUGS.md). When a bug is fixed, its test moves into the main suite.
- **No committed secrets.** Credentials come from the environment or a git-ignored `.env`. CI reads them from repository secrets (`API_EMAIL`, `API_CLIENT_SECRET`).

## CI

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on every push and PR, and on a weekday schedule:

1. `gofmt` and `go vet` (all build tags).
2. Installs the Playwright driver and Chromium.
3. Runs the regression suite and publishes a JUnit report artifact.
4. Runs the known-defect suite in a separate, non-blocking job, so the current state of the open bugs is visible on every run.
