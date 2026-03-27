package cmd_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scli/cmd"
	"github.com/nlink-jp/scli/internal/config"
	"github.com/nlink-jp/scli/internal/keychain"
)

func TestWorkspaceList_Empty(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	cmd.SetServicesForTest(ks, mgr)
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"workspace", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No workspaces configured") {
		t.Errorf("expected 'No workspaces configured', got: %q", buf.String())
	}
}

func TestWorkspaceUse_Success(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	cfg := &config.Config{
		DefaultWorkspace: "ws1",
		Workspaces: map[string]config.WorkspaceConfig{
			"ws1": {TeamID: "T1"},
			"ws2": {TeamID: "T2"},
		},
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}

	cmd.SetServicesForTest(ks, mgr)
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"workspace", "use", "ws2"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultWorkspace != "ws2" {
		t.Errorf("DefaultWorkspace: got %q, want %q", loaded.DefaultWorkspace, "ws2")
	}
}

func TestWorkspaceUse_NotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	cfg := &config.Config{
		Workspaces: map[string]config.WorkspaceConfig{
			"ws1": {},
		},
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}

	cmd.SetServicesForTest(ks, mgr)
	defer cmd.ResetServices()

	root := cmd.RootCmd()
	root.SetArgs([]string{"workspace", "use", "nonexistent"})

	if err := root.Execute(); err == nil {
		t.Error("expected error for unknown workspace, got nil")
	}
}
