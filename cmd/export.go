package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/magifd2/scli/internal/slack"
	"github.com/spf13/cobra"
)

var (
	exportOutput  string
	exportStart   string
	exportEnd     string
	exportSaveDir string
)

var channelExportCmd = &cobra.Command{
	Use:   "export <channel>",
	Short: "Export full channel history as JSON",
	Long: `Export all messages from a channel as JSON, compatible with scat and stail.

Threads are fully expanded: each reply appears immediately after its parent
with is_reply=true and thread_timestamp_unix set to the parent's timestamp.

When --save-dir is provided, attached files are downloaded into that directory
and local_path is set in the JSON output.`,
	Args: cobra.ExactArgs(1),
	RunE: runChannelExport,
}

func init() {
	channelExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "-", `Output file path ("-" writes to stdout)`)
	channelExportCmd.Flags().StringVar(&exportStart, "start", "", "Export messages after this time (RFC3339, e.g. 2026-01-01T00:00:00Z)")
	channelExportCmd.Flags().StringVar(&exportEnd, "end", "", "Export messages before this time (RFC3339, e.g. 2026-03-27T23:59:59Z)")
	channelExportCmd.Flags().StringVar(&exportSaveDir, "save-dir", "", "Directory to download attached files into")
	channelCmd.AddCommand(channelExportCmd)
}

func runChannelExport(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	oldest, err := rfc3339ToSlackTS(exportStart)
	if err != nil {
		return fmt.Errorf("invalid --start: %w", err)
	}
	latest, err := rfc3339ToSlackTS(exportEnd)
	if err != nil {
		return fmt.Errorf("invalid --end: %w", err)
	}

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	channelID, err := client.ResolveChannelID(ctx, nameOrID)
	if err != nil {
		return err
	}

	channelName := resolveExportChannelName(ctx, client, nameOrID, channelID)

	export, err := client.ExportChannel(ctx, channelID, channelName, oldest, latest, exportSaveDir)
	if err != nil {
		return fmt.Errorf("export channel: %w", err)
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	data = append(data, '\n')

	out, closeOut, err := openOutput(cmd, exportOutput)
	if err != nil {
		return err
	}
	defer closeOut() //nolint:errcheck

	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// rfc3339ToSlackTS converts an RFC3339 timestamp string to the Slack timestamp
// format ("1234567890.000000"). Returns an empty string for empty input.
func rfc3339ToSlackTS(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.000000", t.Unix()), nil
}

// openOutput returns a writer for the export output and a deferred close func.
// Path "-" or "" writes to cmd.OutOrStdout(); any other value creates the named file.
func openOutput(cmd *cobra.Command, path string) (io.Writer, func() error, error) {
	if path == "" || path == "-" {
		return cmd.OutOrStdout(), func() error { return nil }, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec
	if err != nil {
		return nil, nil, fmt.Errorf("open output file: %w", err)
	}
	return f, f.Close, nil
}

// resolveExportChannelName returns the channel name (without # prefix) for the
// export envelope. When nameOrID is a raw Slack ID the API is queried;
// if that fails the ID itself is used as a fallback.
func resolveExportChannelName(ctx context.Context, client *slack.Client, nameOrID, channelID string) string {
	if nameOrID != "" && nameOrID[0] != 'C' && nameOrID[0] != 'G' && nameOrID[0] != 'D' {
		name := nameOrID
		if name[0] == '#' {
			name = name[1:]
		}
		return name
	}
	detail, err := client.GetChannelDetail(ctx, channelID)
	if err != nil {
		return channelID
	}
	return detail.Name
}
