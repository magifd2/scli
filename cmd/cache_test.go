package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scli/cmd"
	"github.com/nlink-jp/scli/internal/config"
	"github.com/nlink-jp/scli/internal/keychain"
)

// setupWorkspace creates a config manager with a single workspace that uses an
// inline token, so tests do not need a real keychain.
func setupWorkspace(t *testing.T) (*config.Manager, string) {
	t.Helper()
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	cfg := &config.Config{
		DefaultWorkspace: "test",
		Workspaces: map[string]config.WorkspaceConfig{
			"test": {Token: "xoxp-test"},
		},
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}
	return mgr, dir
}

func TestCacheClear_RemovesCacheDir(t *testing.T) {
	mgr, dir := setupWorkspace(t)
	ks := keychain.NewMockStore()

	// Create cache files to verify they are removed.
	cacheDir := filepath.Join(dir, "cache", "test")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "channels.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd.SetServicesForTest(ks, mgr)
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"cache", "clear"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("expected cache dir %q to be removed, but it still exists", cacheDir)
	}
	if !strings.Contains(buf.String(), "Cache cleared") {
		t.Errorf("expected 'Cache cleared' in output, got: %q", buf.String())
	}
}

func TestCacheClear_NoCacheDir(t *testing.T) {
	mgr, _ := setupWorkspace(t)
	ks := keychain.NewMockStore()

	cmd.SetServicesForTest(ks, mgr)
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"cache", "clear"})

	// Should succeed even when the cache dir does not exist yet.
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error when cache dir is missing: %v", err)
	}
}
