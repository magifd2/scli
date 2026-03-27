package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nlink-jp/scli/internal/config"
	"github.com/nlink-jp/scli/internal/keychain"
)

func TestLoad_EmptyWhenFileNotExist(t *testing.T) {
	mgr := config.NewManager(filepath.Join(t.TempDir(), "config.json"))
	cfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultWorkspace != "" {
		t.Errorf("expected empty default workspace, got %q", cfg.DefaultWorkspace)
	}
	if len(cfg.Workspaces) != 0 {
		t.Errorf("expected empty workspaces, got %v", cfg.Workspaces)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))

	original := &config.Config{
		DefaultWorkspace: "myteam",
		Workspaces: map[string]config.WorkspaceConfig{
			"myteam": {TeamID: "T123", UserID: "U456"},
		},
	}
	if err := mgr.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.DefaultWorkspace != "myteam" {
		t.Errorf("DefaultWorkspace: got %q, want %q", loaded.DefaultWorkspace, "myteam")
	}
	ws := loaded.Workspaces["myteam"]
	if ws.TeamID != "T123" || ws.UserID != "U456" {
		t.Errorf("workspace: got %+v, want TeamID=T123 UserID=U456", ws)
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr := config.NewManager(path)
	if _, err := mgr.Load(); err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	// Use a nested path that doesn't exist yet
	mgr := config.NewManager(filepath.Join(dir, "a", "b", "config.json"))
	cfg := &config.Config{Workspaces: make(map[string]config.WorkspaceConfig)}
	if err := mgr.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestResolveToken_EnvVar(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	t.Setenv("SLACK_TOKEN_MYTEAM", "env-token")
	token, err := mgr.ResolveToken("myteam", ks)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if token != "env-token" {
		t.Errorf("got %q, want %q", token, "env-token")
	}
}

func TestResolveToken_FallbackEnvVar(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	t.Setenv("SLACK_TOKEN", "fallback-token")
	token, err := mgr.ResolveToken("myteam", ks)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if token != "fallback-token" {
		t.Errorf("got %q, want %q", token, "fallback-token")
	}
}

func TestResolveToken_ConfigFile(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	cfg := &config.Config{
		Workspaces: map[string]config.WorkspaceConfig{
			"myteam": {Token: "config-token"},
		},
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}

	token, err := mgr.ResolveToken("myteam", ks)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if token != "config-token" {
		t.Errorf("got %q, want %q", token, "config-token")
	}
}

func TestResolveToken_Keychain(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()
	_ = ks.Set("myteam", "keychain-token")

	token, err := mgr.ResolveToken("myteam", ks)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if token != "keychain-token" {
		t.Errorf("got %q, want %q", token, "keychain-token")
	}
}

func TestResolveToken_NotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	if _, err := mgr.ResolveToken("missing", ks); err == nil {
		t.Error("expected error when token not found, got nil")
	}
}

func TestResolveToken_DotEnvFile(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	// Write a .env file in the config directory
	envPath := filepath.Join(mgr.ConfigDir(), ".env")
	if err := os.MkdirAll(mgr.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("SLACK_TOKEN_MYTEAM=dotenv-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := mgr.ResolveToken("myteam", ks)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if token != "dotenv-token" {
		t.Errorf("got %q, want %q", token, "dotenv-token")
	}
}
