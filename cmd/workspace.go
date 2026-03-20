package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage Slack workspaces",
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured workspaces",
	RunE:  runWorkspaceList,
}

var workspaceUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkspaceUse,
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd, workspaceUseCmd)
	rootCmd.AddCommand(workspaceCmd)
}

func runWorkspaceList(cmd *cobra.Command, _ []string) error {
	mgr, err := newConfigManager()
	if err != nil {
		return err
	}
	cfg, err := mgr.Load()
	if err != nil {
		return err
	}

	if len(cfg.Workspaces) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No workspaces configured. Run: scli auth login")
		return nil
	}

	for name := range cfg.Workspaces {
		marker := "  "
		if name == cfg.DefaultWorkspace {
			marker = "* "
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", marker, name)
	}
	return nil
}

func runWorkspaceUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := newConfigManager()
	if err != nil {
		return err
	}
	cfg, err := mgr.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Workspaces[name]; !ok {
		return fmt.Errorf("workspace %q not found — run: scli auth login --workspace %s", name, name)
	}

	cfg.DefaultWorkspace = name
	if err := mgr.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Default workspace set to %q\n", name)
	return nil
}
