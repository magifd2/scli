package cmd

import (
	"fmt"
	"os"

	"github.com/magifd2/scli/internal/auth"
	"github.com/magifd2/scli/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Slack authentication",
}

var authLoginManual bool

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Slack via OAuth",
	RunE:  runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved authentication for a workspace",
	RunE:  runAuthLogout,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List authenticated workspaces",
	RunE:  runAuthList,
}

func init() {
	authLoginCmd.Flags().BoolVar(&authLoginManual, "manual", false,
		"Manual code entry mode: print the auth URL and prompt for the redirect URL.\n"+
			"Use this in headless environments or when the automatic browser flow fails.\n"+
			"The redirect URI is the same as automatic mode: "+auth.DefaultRedirectURI())
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authListCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	ws := workspace
	if ws == "" {
		ws = "default"
	}

	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("SLACK_CLIENT_ID and SLACK_CLIENT_SECRET environment variables must be set")
	}

	authorizer := auth.New(auth.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  auth.DefaultRedirectURI(),
	})

	var tokenResp *auth.TokenResponse
	var err error
	if authLoginManual {
		tokenResp, err = authorizer.LoginManual(cmd.Context())
	} else {
		tokenResp, err = authorizer.Login(cmd.Context())
	}
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	ks := newKeychainStore()
	if err := ks.Set(ws, tokenResp.AccessToken); err != nil {
		return fmt.Errorf("save token to keychain: %w", err)
	}

	mgr, err := newConfigManager()
	if err != nil {
		return err
	}
	cfg, err := mgr.Load()
	if err != nil {
		return err
	}
	cfg.Workspaces[ws] = config.WorkspaceConfig{
		TeamID: tokenResp.TeamID,
		UserID: tokenResp.UserID,
	}
	if cfg.DefaultWorkspace == "" {
		cfg.DefaultWorkspace = ws
	}
	if err := mgr.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(),
		"Authenticated: user %s in %s (saved as workspace %q)\n",
		tokenResp.UserID, tokenResp.TeamName, ws)
	return nil
}

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	ws := workspace
	if ws == "" {
		ws = "default"
	}

	ks := newKeychainStore()
	if err := ks.Delete(ws); err != nil {
		return fmt.Errorf("remove token for workspace %q: %w", ws, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Logged out from workspace %q\n", ws)
	return nil
}

func runAuthList(cmd *cobra.Command, _ []string) error {
	mgr, err := newConfigManager()
	if err != nil {
		return err
	}
	cfg, err := mgr.Load()
	if err != nil {
		return err
	}

	if len(cfg.Workspaces) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No authenticated workspaces. Run: scli auth login")
		return nil
	}

	for name, ws := range cfg.Workspaces {
		marker := "  "
		if name == cfg.DefaultWorkspace {
			marker = "* "
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s  (team: %s, user: %s)\n",
			marker, name, ws.TeamID, ws.UserID)
	}
	return nil
}
