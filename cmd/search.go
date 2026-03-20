package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	searchLimit int
	searchAsc   bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages in the workspace",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Maximum number of results")
	searchCmd.Flags().BoolVar(&searchAsc, "asc", false, "Sort results oldest first")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	// Join all args as the query to allow unquoted multi-word queries.
	query := args[0]
	for _, a := range args[1:] {
		query += " " + a
	}

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	results, err := client.SearchMessages(cmd.Context(), query, searchLimit, searchAsc)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No results.")
		return nil
	}

	// Resolve usernames for human-posted messages.
	seen := map[string]string{}
	for i, r := range results {
		if r.Message.UserID == "" {
			continue
		}
		if name, ok := seen[r.Message.UserID]; ok {
			results[i].Message.UserName = name
			continue
		}
		name := client.ResolveUserName(cmd.Context(), r.Message.UserID)
		seen[r.Message.UserID] = name
		results[i].Message.UserName = name
	}

	p := newPrinter(cmd)
	return p.SearchResults(results)
}
