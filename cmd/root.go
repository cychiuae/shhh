package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   = "development"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "shhh",
	Short: "A GitOps-friendly secret management tool",
	Long: `shhh is a CLI tool for managing secrets in Git repositories.

It encrypts values within YAML/JSON/INI/ENV files (or entire files),
manages users by GPG email, and supports multiple vaults with
per-file recipient controls.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("shhh version %s (built %s)\n", Version, BuildTime)
	},
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
