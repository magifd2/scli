package slack_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// ---------------------------------------------------------------------------
// Timestamp conversion
// ---------------------------------------------------------------------------

func TestExport_TimestampConversion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/conversations.history" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1740823200.000000", "user": "U001", "text": "hi"},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", "")
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}
	if len(exp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(exp.Messages))
	}
	msg := exp.Messages[0]
	if msg.TimestampUnix != "1740823200.000000" {
		t.Errorf("TimestampUnix: got %q, want %q", msg.TimestampUnix, "1740823200.000000")
	}
	if msg.Timestamp != "2025-03-01T10:00:00Z" {
		t.Errorf("Timestamp (RFC3339): got %q, want %q", msg.Timestamp, "2025-03-01T10:00:00Z")
	}
}

// ---------------------------------------------------------------------------
// Basic structure and post_type detection
// ---------------------------------------------------------------------------

func TestExportChannel_BasicStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.history":
			// Slack returns newest-first; bot (2000) is newer than user (1000).
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "2000.000000", "bot_id": "B001", "username": "MyBot", "text": "bot says hi"},
					{"ts": "1000.000000", "user": "U001", "text": "hello"},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		case "/users.info":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"user": map[string]interface{}{
					"id": "U001", "name": "alice",
					"profile": map[string]string{"display_name": "Alice", "real_name": "Alice Smith"},
				},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", "")
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}

	if exp.ChannelName != "#general" {
		t.Errorf("ChannelName: got %q, want %q", exp.ChannelName, "#general")
	}
	if exp.ExportTimestamp == "" {
		t.Error("ExportTimestamp is empty")
	}
	if len(exp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(exp.Messages))
	}

	user := exp.Messages[0]
	if user.PostType != "user" {
		t.Errorf("user PostType: got %q, want %q", user.PostType, "user")
	}
	if user.UserID != "U001" {
		t.Errorf("user UserID: got %q", user.UserID)
	}
	if user.IsReply {
		t.Error("expected IsReply=false for non-reply message")
	}
	if user.Files == nil {
		t.Error("Files should be an empty slice, not nil")
	}

	bot := exp.Messages[1]
	if bot.PostType != "bot" {
		t.Errorf("bot PostType: got %q, want %q", bot.PostType, "bot")
	}
	if bot.UserName != "MyBot" {
		t.Errorf("bot UserName: got %q, want %q", bot.UserName, "MyBot")
	}
}

// ---------------------------------------------------------------------------
// Cursor-based pagination
// ---------------------------------------------------------------------------

func TestExportChannel_Pagination(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/conversations.history" {
			http.NotFound(w, r)
			return
		}
		n := calls.Add(1)
		switch n {
		case 1:
			// Page 1: newest messages (3000, 2000), has_more=true.
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "3000.000000", "user": "U001", "text": "msg3"},
					{"ts": "2000.000000", "user": "U001", "text": "msg2"},
				},
				"has_more":          true,
				"response_metadata": map[string]string{"next_cursor": "cursor2"},
			})
		default:
			// Page 2: oldest message (1000), no more.
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1000.000000", "user": "U001", "text": "msg1"},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", "")
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 API pages, got %d", calls.Load())
	}
	if len(exp.Messages) != 3 {
		t.Fatalf("expected 3 messages total, got %d", len(exp.Messages))
	}
	// After reversal, oldest message should be first.
	if exp.Messages[0].TimestampUnix != "1000.000000" {
		t.Errorf("first message should be oldest, got ts=%q", exp.Messages[0].TimestampUnix)
	}
}

// ---------------------------------------------------------------------------
// Thread expansion
// ---------------------------------------------------------------------------

func TestExportChannel_ThreadExpansion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1000.000000", "user": "U001", "text": "parent", "reply_count": 1},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		case "/conversations.replies":
			// conversations.replies includes the parent as the first item.
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1000.000000", "user": "U001", "text": "parent", "thread_ts": "1000.000000"},
					{"ts": "1001.000000", "user": "U002", "text": "reply1", "thread_ts": "1000.000000"},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		case "/users.info":
			uid := r.URL.Query().Get("user")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"user": map[string]interface{}{
					"id": uid, "name": uid,
					"profile": map[string]string{"display_name": uid, "real_name": uid},
				},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", "")
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}
	if len(exp.Messages) != 2 {
		t.Fatalf("expected 2 messages (parent + 1 reply), got %d", len(exp.Messages))
	}

	parent := exp.Messages[0]
	if parent.IsReply {
		t.Error("parent should have IsReply=false")
	}
	if parent.ThreadTimestampUnix != "" {
		t.Errorf("parent should have empty ThreadTimestampUnix, got %q", parent.ThreadTimestampUnix)
	}

	reply := exp.Messages[1]
	if !reply.IsReply {
		t.Error("reply should have IsReply=true")
	}
	if reply.ThreadTimestampUnix != "1000.000000" {
		t.Errorf("reply ThreadTimestampUnix: got %q, want %q", reply.ThreadTimestampUnix, "1000.000000")
	}
}

// ---------------------------------------------------------------------------
// File download with --save-dir
// ---------------------------------------------------------------------------

func TestExportChannel_FileDownload(t *testing.T) {
	fileContent := []byte("hello file content")

	var srvRef *httptest.Server
	srvRef = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { //nolint:staticcheck
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts": "1000.000000", "user": "U001", "text": "see file",
						"files": []map[string]interface{}{
							{
								"id": "F001", "name": "test.txt", "mimetype": "text/plain",
								"url_private_download": "http://" + srvRef.Listener.Addr().String() + "/files/F001",
							},
						},
					},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		case "/files/F001":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(fileContent)
		case "/users.info":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"user": map[string]interface{}{
					"id": "U001", "name": "alice",
					"profile": map[string]string{"display_name": "Alice", "real_name": "Alice"},
				},
			})
		}
	}))
	defer srvRef.Close()

	saveDir := t.TempDir()
	c := newTestClient(t, srvRef)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", saveDir)
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}

	if len(exp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(exp.Messages))
	}
	files := exp.Messages[0].Files
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	ef := files[0]
	if ef.ID != "F001" {
		t.Errorf("file ID: got %q, want %q", ef.ID, "F001")
	}
	if ef.LocalPath == "" {
		t.Error("LocalPath should be set after download")
	}

	downloaded, err := os.ReadFile(filepath.Join(saveDir, "F001_test.txt"))
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(downloaded) != string(fileContent) {
		t.Errorf("downloaded content: got %q, want %q", downloaded, fileContent)
	}
}

// ---------------------------------------------------------------------------
// No save-dir: LocalPath stays empty
// ---------------------------------------------------------------------------

func TestExportChannel_NoSaveDir_LocalPathEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts": "1000.000000", "user": "U001", "text": "msg",
						"files": []map[string]interface{}{
							{"id": "F001", "name": "doc.pdf", "mimetype": "application/pdf",
								"url_private_download": "http://files.slack.com/F001"},
						},
					},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		case "/users.info":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"user": map[string]interface{}{
					"id": "U001", "name": "alice",
					"profile": map[string]string{"display_name": "Alice", "real_name": "Alice"},
				},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", "")
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}
	files := exp.Messages[0].Files
	if len(files) != 1 {
		t.Fatalf("expected 1 file metadata, got %d", len(files))
	}
	if files[0].LocalPath != "" {
		t.Errorf("LocalPath should be empty without save-dir, got %q", files[0].LocalPath)
	}
}

// ---------------------------------------------------------------------------
// JSON schema compatibility with scat/stail
// ---------------------------------------------------------------------------

func TestExportChannel_JSONSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1000.000000", "user": "U001", "text": "hi"},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			})
		case "/users.info":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"user": map[string]interface{}{
					"id": "U001", "name": "alice",
					"profile": map[string]string{"display_name": "Alice", "real_name": "Alice"},
				},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	exp, err := c.ExportChannel(t.Context(), "C001", "general", "", "", "")
	if err != nil {
		t.Fatalf("ExportChannel: %v", err)
	}

	b, err := json.Marshal(exp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)

	for _, key := range []string{`"export_timestamp"`, `"channel_name"`, `"messages"`} {
		if !strings.Contains(s, key) {
			t.Errorf("top-level JSON key missing: %s", key)
		}
	}
	for _, key := range []string{`"user_id"`, `"post_type"`, `"timestamp"`, `"timestamp_unix"`, `"text"`, `"files"`, `"is_reply"`} {
		if !strings.Contains(s, key) {
			t.Errorf("message JSON key missing: %s", key)
		}
	}
}
