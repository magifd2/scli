package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/magifd2/scli/internal/output"
	"github.com/magifd2/scli/internal/slack"
	"github.com/spf13/cobra"
)

// newSlackClient resolves the current workspace token and returns a Slack client
// with the workspace-specific disk cache configured.
func newSlackClient() (*slack.Client, error) {
	token, cacheDir, err := resolveTokenAndCacheDir()
	if err != nil {
		return nil, err
	}
	client := slack.NewClient(token)
	client.SetCacheDir(cacheDir)
	return client, nil
}

// resolveTokenAndCacheDir returns the token and the workspace-specific cache
// directory (~/.config/scli/cache/<workspace>/) for the effective workspace.
func resolveTokenAndCacheDir() (token, cacheDir string, err error) {
	mgr, err := newConfigManager()
	if err != nil {
		return "", "", err
	}

	ws := workspace
	if ws == "" {
		cfg, err := mgr.Load()
		if err != nil {
			return "", "", err
		}
		ws = cfg.DefaultWorkspace
		if ws == "" {
			ws = "default"
		}
	}

	ks := newKeychainStore()
	tok, err := mgr.ResolveToken(ws, ks)
	if err != nil {
		return "", "", fmt.Errorf("%w\nRun: scli auth login --workspace %s", err, ws)
	}
	return tok, filepath.Join(mgr.ConfigDir(), "cache", ws), nil
}

// unescapeText converts escape sequences in user-supplied message text.
// Currently handles \n (newline) and \t (tab).
func unescapeText(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}

// newPrinter returns an output.Printer configured from the global flags.
func newPrinter(cmd *cobra.Command) *output.Printer {
	return output.New(cmd.OutOrStdout(), jsonOutput, noColor)
}
