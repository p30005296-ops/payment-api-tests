package tests

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"payment-api-tests/internal/client"
)

// Every transaction endpoint must reject requests without a valid Bearer token.
func TestTransactionEndpoints_RejectInvalidTokens(t *testing.T) {
	t.Parallel()

	token, err := api.Token()
	require.NoError(t, err)
	header, payload, signature := splitJWT(t, token)
	noneHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))

	cases := map[string]map[string]string{
		"no Authorization header": nil,
		"token without Bearer":    {"Authorization": token},
		"empty bearer":            {"Authorization": "Bearer "},
		"garbage token":           bearer("not.a.jwt"),
		"tampered signature":      bearer(header + "." + payload + "." + flipLastChars(signature)),
		"tampered payload":        bearer(header + "." + flipLastChars(payload) + "." + signature),
		"alg none":                bearer(noneHeader + "." + payload + "."),
	}

	endpoints := map[string]func(h map[string]string) (*client.Response, error){
		"authorise": func(h map[string]string) (*client.Response, error) {
			return api.Do("POST", client.PathTransactions, newAuthorisation(), h)
		},
		"complete": func(h map[string]string) (*client.Response, error) {
			return api.Do("POST", client.PathComplete(newAuthorisation().TransactionID), nil, h)
		},
	}

	for ep, call := range endpoints {
		for name, h := range cases {
			t.Run(ep+"/"+name, func(t *testing.T) {
				t.Parallel()
				resp, err := call(h)
				require.NoError(t, err)
				requireUnauthorised(t, resp)
			})
		}
	}
}

func splitJWT(t *testing.T, token string) (header, payload, signature string) {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	return parts[0], parts[1], parts[2]
}

// flipLastChars changes the end of a base64url segment so it no longer matches its signature.
func flipLastChars(s string) string {
	if strings.HasSuffix(s, "AAAA") {
		return s[:len(s)-4] + "BBBB"
	}
	return s[:len(s)-4] + "AAAA"
}
