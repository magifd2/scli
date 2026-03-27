package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage local cache",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove cached channel and user data for the current workspace",
	RunE:  runCacheClear,
}

func init() {
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}

func runCacheClear(cmd *cobra.Command, _ []string) error {
	_, cacheDir, err := resolveTokenAndCacheDir()
	if err != nil {
		return err
	}

	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear cache: %w", err)
	}

	p := newPrinter(cmd)
	p.Success(fmt.Sprintf("Cache cleared: %s", cacheDir))
	return nil
}
