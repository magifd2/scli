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

var userInfoCmd = &cobra.Command{
	Use:   "info <user>",
	Short: "Show profile information for a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserInfo,
}

func init() {
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userInfoCmd)
	rootCmd.AddCommand(userCmd)
}

func runUserInfo(cmd *cobra.Command, args []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	userID, err := client.ResolveUserID(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	profile, err := client.GetUserProfile(cmd.Context(), userID)
	if err != nil {
		return fmt.Errorf("get user profile: %w", err)
	}

	p := newPrinter(cmd)
	return p.UserProfile(profile)
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
