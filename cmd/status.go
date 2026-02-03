package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/gitignore"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

var statusVault string

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusVault, "vault", "v", "", "Show status for specific vault")
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all registered files",
	Long: `Display the encryption status of all registered files.

Shows:
- File encryption state (encrypted, decrypted, pending, missing)
- Warnings about expiring keys
- Gitignore status`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	var vaults []string
	if statusVault != "" {
		if !s.VaultExists(statusVault) {
			return fmt.Errorf("vault %q does not exist", statusVault)
		}
		vaults = []string{statusVault}
	} else {
		vaults, err = s.ListVaults()
		if err != nil {
			return err
		}
	}

	hasWarnings := false
	totalFiles := 0

	for _, vault := range vaults {
		users, _ := config.LoadVaultUsers(s, vault)
		files, err := config.LoadVaultFiles(s, vault)
		if err != nil {
			continue
		}

		if len(files.Files) == 0 && (statusVault == "" || len(vaults) > 1) {
			continue
		}

		fmt.Printf("Vault: %s\n", vault)

		if users != nil {
			for _, u := range users.Users {
				if crypto.IsExpired(u.ExpiresAt) {
					fmt.Printf("  ⚠ User %s: key has EXPIRED\n", u.Email)
					hasWarnings = true
				} else if crypto.IsExpiringSoon(u.ExpiresAt, 30) {
					fmt.Printf("  ⚠ User %s: key expires %s\n", u.Email, u.ExpiresAt.Format("2006-01-02"))
					hasWarnings = true
				}
			}
		}

		if len(files.Files) == 0 {
			fmt.Println("  No files registered")
			fmt.Println()
			continue
		}

		fmt.Println()

		for _, f := range files.Files {
			totalFiles++
			status := getFileStatusDetailed(s.Root(), f.Path)

			icon := "✓"
			switch status.State {
			case "encrypted":
				icon = "🔒"
			case "decrypted":
				icon = "🔓"
			case "pending":
				icon = "⏳"
			case "missing":
				icon = "❌"
				hasWarnings = true
			}

			fmt.Printf("  %s %s [%s]\n", icon, f.Path, status.State)

			if status.Warning != "" {
				fmt.Printf("      ⚠ %s\n", status.Warning)
				hasWarnings = true
			}

			if !gitignore.IsIgnored(s.Root(), f.Path) {
				fmt.Printf("      ⚠ Not in .gitignore!\n")
				hasWarnings = true
			}
		}

		fmt.Println()
	}

	if totalFiles == 0 {
		fmt.Println("No files registered")
		return nil
	}

	fmt.Printf("Total: %d file(s)\n", totalFiles)

	if hasWarnings {
		fmt.Println("\n⚠ Some issues need attention")
	}

	return nil
}

type FileStatusDetailed struct {
	State   string
	Warning string
}

func getFileStatusDetailed(root, path string) FileStatusDetailed {
	plainPath := filepath.Join(root, path)
	encPath := plainPath + ".enc"

	plainExists := fileExists(plainPath)
	encExists := fileExists(encPath)

	result := FileStatusDetailed{}

	switch {
	case encExists && plainExists:
		result.State = "decrypted"

		plainInfo, _ := os.Stat(plainPath)
		encInfo, _ := os.Stat(encPath)

		if plainInfo != nil && encInfo != nil {
			if plainInfo.ModTime().After(encInfo.ModTime()) {
				result.Warning = "Plaintext modified after encryption"
			}
		}

	case encExists && !plainExists:
		result.State = "encrypted"

	case !encExists && plainExists:
		result.State = "pending"
		result.Warning = "Not yet encrypted"

	default:
		result.State = "missing"
		result.Warning = "Neither plaintext nor encrypted file exists"
	}

	return result
}
