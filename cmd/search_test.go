package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magifd2/scli/cmd"
	"github.com/magifd2/scli/internal/slack"
)

// newSearchTestServer starts a test HTTP server that responds to
// conversations.list and users.list with fixed fixtures.
func newSearchTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general", "is_member": true,
					"purpose": map[string]string{"value": "Company-wide"}},
				{"id": "C002", "name": "engineering", "is_member": true,
					"purpose": map[string]string{"value": "Tech talk"}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})

	mux.HandleFunc("/users.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{"id": "U001", "name": "alice", "deleted": false, "is_bot": false,
					"profile": map[string]string{"display_name": "Alice", "real_name": "Alice Smith"}},
				{"id": "U002", "name": "bob", "deleted": false, "is_bot": false,
					"profile": map[string]string{"display_name": "Bob", "real_name": "Bob Jones"}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})

	return httptest.NewServer(mux)
}

func TestChannelSearch_MatchByName(t *testing.T) {
	srv := newSearchTestServer(t)
	defer srv.Close()

	cmd.SetSlackClientForTest(func() (*slack.Client, error) {
		c := slack.NewClient("xoxp-test")
		c.SetBaseURL(srv.URL)
		return c, nil
	})
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"channel", "search", "engineering"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "engineering") {
		t.Errorf("expected 'engineering' in output, got: %q", out)
	}
	if strings.Contains(out, "general") {
		t.Errorf("unexpected 'general' in search results, got: %q", out)
	}
}

func TestChannelSearch_NoMatch(t *testing.T) {
	srv := newSearchTestServer(t)
	defer srv.Close()

	cmd.SetSlackClientForTest(func() (*slack.Client, error) {
		c := slack.NewClient("xoxp-test")
		c.SetBaseURL(srv.URL)
		return c, nil
	})
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"channel", "search", "zzznomatch"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No channels found") {
		t.Errorf("expected 'No channels found' message, got: %q", buf.String())
	}
}

func TestUserSearch_MatchByName(t *testing.T) {
	srv := newSearchTestServer(t)
	defer srv.Close()

	cmd.SetSlackClientForTest(func() (*slack.Client, error) {
		c := slack.NewClient("xoxp-test")
		c.SetBaseURL(srv.URL)
		return c, nil
	})
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"user", "search", "alice"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "alice") {
		t.Errorf("expected 'alice' in output, got: %q", out)
	}
	if strings.Contains(out, "bob") {
		t.Errorf("unexpected 'bob' in search results, got: %q", out)
	}
}

func TestUserSearch_NoMatch(t *testing.T) {
	srv := newSearchTestServer(t)
	defer srv.Close()

	cmd.SetSlackClientForTest(func() (*slack.Client, error) {
		c := slack.NewClient("xoxp-test")
		c.SetBaseURL(srv.URL)
		return c, nil
	})
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"user", "search", "zzznomatch"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No users found") {
		t.Errorf("expected 'No users found' message, got: %q", buf.String())
	}
}
