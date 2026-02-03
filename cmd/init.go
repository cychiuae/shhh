package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize shhh in the current directory",
	Long: `Initialize a new shhh project in the current directory.

This creates a .shhh/ directory with the default configuration
and a default vault. If the current directory is a git repository,
.shhh/ will be configured for version control.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	s := store.New(cwd)

	if s.IsInitialized() {
		return fmt.Errorf("shhh already initialized in %s", cwd)
	}

	if err := s.Initialize(); err != nil {
		return err
	}

	cfg := config.NewConfig()
	if err := cfg.Save(s); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	usersData := config.NewVaultUsers()
	if err := usersData.Save(s, store.DefaultVault); err != nil {
		return fmt.Errorf("failed to initialize users: %w", err)
	}

	filesData := config.NewVaultFiles()
	if err := filesData.Save(s, store.DefaultVault); err != nil {
		return fmt.Errorf("failed to initialize files: %w", err)
	}

	isGit := isGitRepo(cwd)

	fmt.Println("Initialized shhh in", cwd)
	fmt.Println("  Created .shhh/ directory")
	fmt.Println("  Created default vault")
	if isGit {
		fmt.Println("  Detected git repository")
	}

	return nil
}

func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
