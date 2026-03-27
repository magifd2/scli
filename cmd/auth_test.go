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

func TestAuthList_Empty(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	cmd.SetServicesForTest(ks, mgr)
	defer cmd.ResetServices()

	buf := new(bytes.Buffer)
	root := cmd.RootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"auth", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No authenticated workspaces") {
		t.Errorf("expected 'No authenticated workspaces', got: %q", buf.String())
	}
}

func TestAuthList_WithWorkspaces(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(filepath.Join(dir, "config.json"))
	ks := keychain.NewMockStore()

	cfg := &config.Config{
		DefaultWorkspace: "myteam",
		Workspaces: map[string]config.WorkspaceConfig{
			"myteam":  {TeamID: "T111", UserID: "U222"},
			"another": {TeamID: "T333", UserID: "U444"},
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
	root.SetArgs([]string{"auth", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "myteam") {
		t.Errorf("expected 'myteam' in output, got: %q", out)
	}
	if !strings.Contains(out, "* myteam") {
		t.Errorf("expected default marker for 'myteam', got: %q", out)
	}
}
