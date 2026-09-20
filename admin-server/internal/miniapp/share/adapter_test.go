package share

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("dial failed for " + req.URL.String())
}

func TestTransportErrorsDoNotExposeCredentials(t *testing.T) {
	const secret = "dummy-secret-must-not-leak"
	const token = "dummy-token-must-not-leak"
	client := &defaultWeChatShareClient{httpClient: &http.Client{Transport: failingTransport{}}}

	_, _, tokenErr := client.GetAccessToken(context.Background(), "dummy-app", secret)
	_, _, qrErr := client.GetUnlimitedQRCode(context.Background(), token, WeChatQRCodeParams{Scene: "p=1"})
	for _, err := range []error{tokenErr, qrErr} {
		if err == nil {
			t.Fatal("expected transport error")
		}
		if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "https://") {
			t.Fatalf("credential-bearing transport detail escaped safe boundary: %q", err)
		}
	}

	// The API serializes service errors with err.Error(); verify that even this
	// final JSON boundary cannot reveal credentials.
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"code":-1,"message":"` + tokenErr.Error() + `"}`))
	body := w.Body.String()
	if strings.Contains(body, secret) || strings.Contains(body, token) || strings.Contains(body, "https://") {
		t.Fatalf("API JSON leaked credential-bearing detail: %s", body)
	}
}

type credentialTokenClient struct {
	calls int
}

func (c *credentialTokenClient) GetAccessToken(_ context.Context, _ string, secret string) (string, int, error) {
	c.calls++
	return "token-for-" + secret, 120, nil
}

func (*credentialTokenClient) GetUnlimitedQRCode(context.Context, string, WeChatQRCodeParams) ([]byte, string, error) {
	return nil, "", nil
}

func TestTokenManagerRefreshesWhenSecretChanges(t *testing.T) {
	client := &credentialTokenClient{}
	manager := NewTokenManager(client, nil)
	first, err := manager.GetToken(context.Background(), "same-app", "secret-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.GetToken(context.Background(), "same-app", "secret-b")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || client.calls != 2 {
		t.Fatalf("credential rotation reused stale token: first=%q second=%q calls=%d", first, second, client.calls)
	}
}
