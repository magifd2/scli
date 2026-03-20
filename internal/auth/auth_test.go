package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCodeChallenge(t *testing.T) {
	verifier := "test-verifier-string"
	got := codeChallenge(verifier)

	// Manually compute expected S256 challenge
	h := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(h[:])

	if got != want {
		t.Errorf("codeChallenge(%q) = %q, want %q", verifier, got, want)
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	v1, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}
	v2, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}

	// Must be non-empty and base64url-safe
	if v1 == "" {
		t.Error("expected non-empty verifier")
	}
	// Must be unique
	if v1 == v2 {
		t.Error("expected unique verifiers, got duplicates")
	}
}

func TestGenerateState(t *testing.T) {
	s1, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	s2, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if s1 == s2 {
		t.Error("expected unique states, got duplicates")
	}
}

func TestExchangeCode_Success(t *testing.T) {
	// Spin up a mock Slack token endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"authed_user": {"id": "U123", "access_token": "xoxp-test-token"},
			"team": {"id": "T456", "name": "TestTeam"}
		}`))
	}))
	defer srv.Close()

	a := &Authorizer{
		cfg:        Config{ClientID: "cid", ClientSecret: "csecret", RedirectURI: "http://localhost/cb"},
		httpClient: srv.Client(),
		tokenURL:   srv.URL,
	}

	resp, err := a.exchangeCode(t.Context(), "auth-code", "verifier")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}
	if resp.AccessToken != "xoxp-test-token" {
		t.Errorf("AccessToken: got %q, want %q", resp.AccessToken, "xoxp-test-token")
	}
	if resp.UserID != "U123" {
		t.Errorf("UserID: got %q, want %q", resp.UserID, "U123")
	}
	if resp.TeamID != "T456" {
		t.Errorf("TeamID: got %q, want %q", resp.TeamID, "T456")
	}
}

func TestExchangeCode_SlackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": false, "error": "invalid_code"}`))
	}))
	defer srv.Close()

	a := &Authorizer{
		cfg:        Config{ClientID: "cid", ClientSecret: "csecret", RedirectURI: "http://localhost/cb"},
		httpClient: srv.Client(),
		tokenURL:   srv.URL,
	}

	_, err := a.exchangeCode(t.Context(), "bad-code", "verifier")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid_code") {
		t.Errorf("expected 'invalid_code' in error, got: %v", err)
	}
}

func TestBuildAuthURL(t *testing.T) {
	a := &Authorizer{
		cfg: Config{
			ClientID:    "my-client-id",
			RedirectURI: "http://localhost:7777/callback",
		},
		authURL: slackAuthURL,
	}

	rawURL := a.buildAuthURL("challenge123", "state456")

	for _, want := range []string{
		"client_id=my-client-id",
		"code_challenge=challenge123",
		"code_challenge_method=S256",
		"state=state456",
	} {
		if !strings.Contains(rawURL, want) {
			t.Errorf("auth URL missing %q: %s", want, rawURL)
		}
	}
}
