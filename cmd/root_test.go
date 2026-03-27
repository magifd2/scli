package cmd_test

import (
	"bytes"
	"testing"

	"github.com/nlink-jp/scli/cmd"
)

func TestRootCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd := cmd.RootCmd()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected help output, got empty")
	}
}

func TestRootCommandVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd := cmd.RootCmd()
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})

	// --version exits with code 0; cobra writes to out
	_ = rootCmd.Execute()

	if buf.Len() == 0 {
		t.Error("expected version output, got empty")
	}
}
