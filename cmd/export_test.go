package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scli/cmd"
	"github.com/nlink-jp/scli/internal/slack"
)

func newExportTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general", "is_member": true,
					"purpose": map[string]string{"value": ""}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})

	mux.HandleFunc("/conversations.history", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{"ts": "1000.000000", "user": "U001", "text": "hello export"},
			},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})

	mux.HandleFunc("/users.info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id": "U001", "name": "alice",
				"profile": map[string]string{"display_name": "Alice", "real_name": "Alice Smith"},
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestChannelExport_ToStdout(t *testing.T) {
	srv := newExportTestServer(t)
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
	root.SetArgs([]string{"channel", "export", "#general", "--output", "-"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"channel_name"`) {
		t.Errorf("expected JSON output with channel_name, got: %q", out)
	}
	if !strings.Contains(out, "#general") {
		t.Errorf("expected #general in output, got: %q", out)
	}
	if !strings.Contains(out, "hello export") {
		t.Errorf("expected message text in output, got: %q", out)
	}

	// Verify it is valid JSON.
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Errorf("output is not valid JSON: %v\n%s", err, out)
	}
}

func TestChannelExport_ToFile(t *testing.T) {
	srv := newExportTestServer(t)
	defer srv.Close()

	cmd.SetSlackClientForTest(func() (*slack.Client, error) {
		c := slack.NewClient("xoxp-test")
		c.SetBaseURL(srv.URL)
		return c, nil
	})
	defer cmd.ResetServices()

	outFile := filepath.Join(t.TempDir(), "export.json")

	root := cmd.RootCmd()
	root.SetArgs([]string{"channel", "export", "#general", "--output", outFile})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("output file is not valid JSON: %v", err)
	}
	if result["channel_name"] != "#general" {
		t.Errorf("channel_name: got %v, want %q", result["channel_name"], "#general")
	}
}

func TestChannelExport_StartEndFlags(t *testing.T) {
	var gotOldest, gotLatest string
	srv := httptest.NewServer(http.NewServeMux())
	mux := http.NewServeMux()

	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general", "is_member": true,
					"purpose": map[string]string{"value": ""}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		gotOldest = r.URL.Query().Get("oldest")
		gotLatest = r.URL.Query().Get("latest")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":                true,
			"messages":          []map[string]interface{}{},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	srv.Close()
	srv = httptest.NewServer(mux)
	defer srv.Close()

	cmd.SetSlackClientForTest(func() (*slack.Client, error) {
		c := slack.NewClient("xoxp-test")
		c.SetBaseURL(srv.URL)
		return c, nil
	})
	defer cmd.ResetServices()

	root := cmd.RootCmd()
	root.SetArgs([]string{
		"channel", "export", "#general",
		"--output", "-",
		"--start", "2026-01-01T00:00:00Z",
		"--end", "2026-03-27T00:00:00Z",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotOldest != "1767225600.000000" {
		t.Errorf("oldest param: got %q, want %q", gotOldest, "1767225600.000000")
	}
	if gotLatest != "1774569600.000000" {
		t.Errorf("latest param: got %q, want %q", gotLatest, "1774569600.000000")
	}
}
