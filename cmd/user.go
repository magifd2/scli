package cmd

import (
	"fmt"
	"strings"

	"github.com/magifd2/scli/internal/slack"
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

var userSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search users by name or display name",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runUserSearch,
}

func init() {
	userCmd.AddCommand(userListCmd, userInfoCmd, userSearchCmd)
	rootCmd.AddCommand(userCmd)
}

func runUserSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(strings.Join(args, " "))

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	users, err := client.ListUsers(cmd.Context())
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	var matches []slack.User
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Name), query) ||
			strings.Contains(strings.ToLower(u.DisplayName), query) ||
			strings.Contains(strings.ToLower(u.RealName), query) {
			matches = append(matches, u)
		}
	}

	if len(matches) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No users found matching %q.\n", query)
		return nil
	}

	p := newPrinter(cmd)
	return p.Users(matches)
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
