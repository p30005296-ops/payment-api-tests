// Package client is a thin, typed wrapper around Playwright's APIRequestContext
// for the payment API. It returns raw status and body so tests can assert on
// both success and error responses.
package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"

	"payment-api-tests/internal/config"
	"payment-api-tests/internal/models"
)

const (
	PathLogin        = "/login"
	PathTransactions = "/api/v1/transactions/"
)

// PathComplete returns the completion endpoint for a transaction.
func PathComplete(id string) string { return PathTransactions + id + "/complete" }

// refreshMargin renews the token this long before its `exp` claim.
const refreshMargin = 30 * time.Second

type Client struct {
	req playwright.APIRequestContext
	cfg config.Config

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// Response is a fully-read HTTP response.
type Response struct {
	Status int
	Body   []byte
}

// JSON decodes the body into v.
func (r *Response) JSON(v any) error {
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("decode %d response %q: %w", r.Status, r.Body, err)
	}
	return nil
}

func (r *Response) String() string { return fmt.Sprintf("HTTP %d: %s", r.Status, r.Body) }

func New(pw *playwright.Playwright, cfg config.Config) (*Client, error) {
	timeoutMs := float64(cfg.Timeout.Milliseconds())
	req, err := pw.Request.NewContext(playwright.APIRequestNewContextOptions{
		BaseURL:          playwright.String(cfg.BaseURL),
		ExtraHttpHeaders: map[string]string{"Content-Type": "application/json"},
		Timeout:          &timeoutMs,
	})
	if err != nil {
		return nil, fmt.Errorf("create API request context: %w", err)
	}
	return &Client{req: req, cfg: cfg}, nil
}

func (c *Client) Close() error { return c.req.Dispose() }

// Do sends a request. body may be nil, raw bytes/string (sent verbatim) or any
// value that is JSON-encoded. headers override the defaults.
func (c *Client) Do(method, path string, body any, headers map[string]string) (*Response, error) {
	opts := playwright.APIRequestContextFetchOptions{
		Method:  playwright.String(method),
		Headers: headers,
	}
	switch b := body.(type) {
	case nil:
	case []byte:
		opts.Data = b
	case string:
		opts.Data = []byte(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		opts.Data = raw
	}

	// The hosting platform answers bursts with 502/503/504 before the request
	// reaches the application, so those are retried with a short backoff.
	var out *Response
	for attempt := 1; ; attempt++ {
		r, err := c.fetch(method, path, opts)
		if err != nil {
			return nil, err
		}
		out = r
		if !retryable(r.Status) || attempt == maxAttempts {
			return out, nil
		}
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}
}

const maxAttempts = 4

func retryable(status int) bool { return status == 502 || status == 503 || status == 504 }

func (c *Client) fetch(method, path string, opts playwright.APIRequestContextFetchOptions) (*Response, error) {
	resp, err := c.req.Fetch(path, opts)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Dispose()

	data, err := resp.Body()
	if err != nil {
		return nil, fmt.Errorf("read %s %s body: %w", method, path, err)
	}
	return &Response{Status: resp.Status(), Body: data}, nil
}

// Login calls POST /login with the given credentials.
func (c *Client) Login(req any) (*Response, error) {
	return c.Do("POST", PathLogin, req, nil)
}

// Authorise calls POST /api/v1/transactions/ using the cached session token.
func (c *Client) Authorise(req any) (*Response, error) {
	h, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	return c.Do("POST", PathTransactions, req, h)
}

// Complete calls POST /api/v1/transactions/{id}/complete using the cached session token.
func (c *Client) Complete(id string) (*Response, error) {
	h, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	return c.Do("POST", PathComplete(id), nil, h)
}

// Token returns a valid JWT, logging in again when the cached one is about to expire.
func (c *Client) Token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Until(c.tokenExp) > refreshMargin {
		return c.token, nil
	}

	resp, err := c.Login(models.LoginRequest{Email: c.cfg.Email, ClientSecret: c.cfg.ClientSecret})
	if err != nil {
		return "", err
	}
	if resp.Status != 200 {
		return "", fmt.Errorf("login failed: %s", resp)
	}
	var body models.LoginResponse
	if err := resp.JSON(&body); err != nil {
		return "", err
	}
	exp, err := TokenExpiry(body.Token)
	if err != nil {
		return "", err
	}
	c.token, c.tokenExp = body.Token, exp
	return c.token, nil
}

func (c *Client) authHeader() (map[string]string, error) {
	tok, err := c.Token()
	if err != nil {
		return nil, err
	}
	return map[string]string{"Authorization": "Bearer " + tok}, nil
}

// TokenExpiry reads the `exp` claim from a JWT without verifying its signature.
func TokenExpiry(token string) (time.Time, error) {
	claims, err := TokenClaims(token)
	if err != nil {
		return time.Time{}, err
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("token has no numeric exp claim")
	}
	return time.Unix(int64(exp), 0), nil
}

// TokenClaims decodes the payload segment of a JWT.
func TokenClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("token has %d segments, want 3", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode token payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, fmt.Errorf("parse token payload: %w", err)
	}
	return claims, nil
}
