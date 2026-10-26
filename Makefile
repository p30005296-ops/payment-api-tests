GOTESTSUM := go run gotest.tools/gotestsum@v1.13.0

.PHONY: setup test test-api test-known-issues test-slow report lint

setup: ## Download modules, the Playwright driver and Chromium
	go mod download
	go run github.com/mxschmitt/playwright-go/cmd/playwright install --with-deps chromium
	@test -f .env || { cp .env.example .env; echo "Fill in API_EMAIL / API_CLIENT_SECRET in .env"; }

test: ## Regression suite (API + Swagger UI)
	go test -count=1 ./...

test-api: ## API tests only, no browser needed
	go test -count=1 -short ./...

test-known-issues: ## Tests asserting correct behaviour for open defects (expected to FAIL)
	go test -count=1 -tags knownissues -run KnownIssue -v ./tests/

test-slow: ## Token expiry test (waits ~5 minutes)
	go test -count=1 -tags slow -run Expiry -v -timeout 10m ./tests/

report: ## Regression suite with JUnit XML in reports/
	mkdir -p reports
	$(GOTESTSUM) --format testname --junitfile reports/junit.xml -- -count=1 ./...

lint:
	test -z "$$(gofmt -l .)" && go vet -tags "knownissues slow" ./...
