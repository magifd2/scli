package cmd

import (
	"fmt"
	"strings"

	"github.com/magifd2/scli/internal/output"
	"github.com/magifd2/scli/internal/slack"
	"github.com/spf13/cobra"
)

// newSlackClient resolves the current workspace token and returns a Slack client.
func newSlackClient() (*slack.Client, error) {
	token, err := resolveToken()
	if err != nil {
		return nil, err
	}
	return slack.NewClient(token), nil
}

// resolveToken returns the token for the effective workspace.
func resolveToken() (string, error) {
	mgr, err := newConfigManager()
	if err != nil {
		return "", err
	}

	ws := workspace
	if ws == "" {
		cfg, err := mgr.Load()
		if err != nil {
			return "", err
		}
		ws = cfg.DefaultWorkspace
		if ws == "" {
			ws = "default"
		}
	}

	ks := newKeychainStore()
	token, err := mgr.ResolveToken(ws, ks)
	if err != nil {
		return "", fmt.Errorf("%w\nRun: scli auth login --workspace %s", err, ws)
	}
	return token, nil
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
