package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage workspace users",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspace members",
	RunE:  runUserList,
}

func init() {
	userCmd.AddCommand(userListCmd)
	rootCmd.AddCommand(userCmd)
}

func runUserList(cmd *cobra.Command, _ []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	users, err := client.ListUsers(cmd.Context())
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	if len(users) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No users found.")
		return nil
	}

	p := newPrinter(cmd)
	return p.Users(users)
}
