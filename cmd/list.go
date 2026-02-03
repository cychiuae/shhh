package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

var listVault string

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listVault, "vault", "v", "", "List files in specific vault (default: all vaults)")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered files",
	Long:  `List all files registered for encryption across all vaults or a specific vault.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	var vaults []string
	if listVault != "" {
		if !s.VaultExists(listVault) {
			return fmt.Errorf("vault %q does not exist", listVault)
		}
		vaults = []string{listVault}
	} else {
		vaults, err = s.ListVaults()
		if err != nil {
			return err
		}
	}

	totalFiles := 0

	for _, vaultName := range vaults {
		vault, err := config.LoadVault(s, vaultName)
		if err != nil {
			fmt.Printf("Warning: failed to load vault %s: %v\n", vaultName, err)
			continue
		}

		if len(vault.Files) == 0 {
			continue
		}

		fmt.Printf("Vault: %s\n", vaultName)
		fmt.Println()

		for _, f := range vault.Files {
			totalFiles++

			status := getFileStatus(s.Root(), f.Path)
			recipientCount := len(f.Recipients)
			recipientStr := "all users"
			if recipientCount > 0 {
				recipientStr = fmt.Sprintf("%d specific", recipientCount)
			}

			fmt.Printf("  %s\n", f.Path)
			fmt.Printf("    Mode: %s | Recipients: %s | Status: %s\n", f.Mode, recipientStr, status)
		}
		fmt.Println()
	}

	if totalFiles == 0 {
		fmt.Println("No files registered")
	}

	return nil
}

func getFileStatus(root, path string) string {
	plainPath := filepath.Join(root, path)
	encPath := plainPath + ".enc"

	plainExists := fileExists(plainPath)
	encExists := fileExists(encPath)

	switch {
	case encExists && plainExists:
		return "decrypted"
	case encExists && !plainExists:
		return "encrypted"
	case !encExists && plainExists:
		return "pending"
	default:
		return "missing"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
