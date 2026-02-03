package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(fileCmd)
	fileCmd.AddCommand(fileSetRecipientsCmd)
	fileCmd.AddCommand(fileClearRecipientsCmd)
	fileCmd.AddCommand(fileAddRecipientsCmd)
	fileCmd.AddCommand(fileRemoveRecipientsCmd)
	fileCmd.AddCommand(fileSetModeCmd)
	fileCmd.AddCommand(fileSetGPGCopyCmd)
	fileCmd.AddCommand(fileClearGPGCopyCmd)
	fileCmd.AddCommand(fileShowCmd)
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage file-specific settings",
	Long:  `Configure per-file encryption settings including recipients, mode, and GPG backup.`,
}

var fileSetRecipientsCmd = &cobra.Command{
	Use:   "set-recipients <file> <email>...",
	Short: "Set specific recipients for a file",
	Long: `Restrict encryption to specific recipients instead of all vault users.

Recipients must be users in the file's vault.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runFileSetRecipients,
}

var fileClearRecipientsCmd = &cobra.Command{
	Use:   "clear-recipients <file>",
	Short: "Clear per-file recipients",
	Long:  `Remove per-file recipient restrictions. The file will use all vault users.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFileClearRecipients,
}

var fileAddRecipientsCmd = &cobra.Command{
	Use:   "add-recipients <file> <email>...",
	Short: "Add recipients to a file",
	Long: `Add recipients to the file's recipient list.

Recipients must be users in the file's vault.
If the file has no per-file recipients, this enables per-file recipient restriction.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runFileAddRecipients,
}

var fileRemoveRecipientsCmd = &cobra.Command{
	Use:   "remove-recipients <file> <email>...",
	Short: "Remove recipients from a file",
	Long: `Remove recipients from the file's recipient list.

If all recipients are removed, the file will use all vault users.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runFileRemoveRecipients,
}

var fileSetModeCmd = &cobra.Command{
	Use:   "set-mode <file> <mode>",
	Short: "Set encryption mode for a file",
	Long: `Set the encryption mode: 'values' or 'full'.

- values: Encrypt only the values in structured files (YAML, JSON, etc.)
- full: Encrypt the entire file contents`,
	Args: cobra.ExactArgs(2),
	RunE: runFileSetMode,
}

var fileSetGPGCopyCmd = &cobra.Command{
	Use:   "set-gpg-copy <file> <true|false>",
	Short: "Enable or disable GPG backup for a file",
	Long: `Set per-file GPG backup setting, overriding the global config.

When enabled, a native .gpg file will be created alongside the .enc file.
Use 'clear-gpg-copy' to remove the per-file setting and use global config.`,
	Args: cobra.ExactArgs(2),
	RunE: runFileSetGPGCopy,
}

var fileClearGPGCopyCmd = &cobra.Command{
	Use:   "clear-gpg-copy <file>",
	Short: "Clear per-file GPG backup setting",
	Long:  `Remove the per-file GPG backup setting. The file will use the global gpg_copy config.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFileClearGPGCopy,
}

var fileShowCmd = &cobra.Command{
	Use:   "show <file>",
	Short: "Show file settings and status",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileShow,
}

func runFileSetRecipients(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]
	recipients := args[1:]

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.SetFileRecipients(s, vault, relPath, recipients); err != nil {
		return err
	}

	fmt.Printf("Set recipients for %s: %v\n", relPath, recipients)
	fmt.Println("Note: Run 'shhh reencrypt' to apply the new recipients")
	return nil
}

func runFileClearRecipients(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.ClearFileRecipients(s, vault, relPath); err != nil {
		return err
	}

	fmt.Printf("Cleared recipients for %s (will use all vault users)\n", relPath)
	fmt.Println("Note: Run 'shhh reencrypt' to apply the change")
	return nil
}

func runFileAddRecipients(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]
	recipients := args[1:]

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.AddFileRecipients(s, vault, relPath, recipients); err != nil {
		return err
	}

	fmt.Printf("Added recipients to %s: %v\n", relPath, recipients)
	fmt.Println("Note: Run 'shhh reencrypt' to apply the new recipients")
	return nil
}

func runFileRemoveRecipients(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]
	recipients := args[1:]

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.RemoveFileRecipients(s, vault, relPath, recipients); err != nil {
		return err
	}

	fmt.Printf("Removed recipients from %s: %v\n", relPath, recipients)
	fmt.Println("Note: Run 'shhh reencrypt' to apply the change")
	return nil
}

func runFileSetMode(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]
	mode := args[1]

	if mode != "values" && mode != "full" {
		return fmt.Errorf("invalid mode: %s (must be 'values' or 'full')", mode)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.SetFileMode(s, vault, relPath, mode); err != nil {
		return err
	}

	fmt.Printf("Set mode for %s: %s\n", relPath, mode)
	fmt.Println("Note: Run 'shhh reencrypt' to apply the new mode")
	return nil
}

func runFileSetGPGCopy(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]
	valueStr := strings.ToLower(args[1])

	gpgCopy := valueStr == "true" || valueStr == "1" || valueStr == "yes"

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.SetFileGPGCopy(s, vault, relPath, gpgCopy); err != nil {
		return err
	}

	if gpgCopy {
		fmt.Printf("Enabled GPG backup for %s (overrides global setting)\n", relPath)
	} else {
		fmt.Printf("Disabled GPG backup for %s (overrides global setting)\n", relPath)
	}

	return nil
}

func runFileClearGPGCopy(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := args[0]

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, _, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	if err := config.ClearFileGPGCopy(s, vault, relPath); err != nil {
		return err
	}

	fmt.Printf("Cleared GPG backup setting for %s (will use global config)\n", relPath)
	return nil
}

func runFileShow(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	filePath := strings.TrimSuffix(args[0], ".enc")

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	relPath, err := filepath.Rel(s.Root(), absPath)
	if err != nil {
		return fmt.Errorf("file must be within project directory: %w", err)
	}

	vault, fileReg, err := config.FindFileVault(s, relPath)
	if err != nil {
		return err
	}

	fmt.Printf("File: %s\n\n", relPath)

	fmt.Printf("Registration:\n")
	fmt.Printf("  Vault: %s\n", vault)
	fmt.Printf("  Mode: %s\n", fileReg.Mode)

	// Display GPG Copy with source indication
	effectiveGPGCopy := config.GetEffectiveGPGCopy(s, fileReg)
	if fileReg.GPGCopy != nil {
		fmt.Printf("  GPG Copy: %v (per-file override)\n", effectiveGPGCopy)
	} else {
		fmt.Printf("  GPG Copy: %v (from global config)\n", effectiveGPGCopy)
	}

	fmt.Printf("  Registered: %s\n", fileReg.RegisteredAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

	fmt.Printf("Recipients:\n")
	if len(fileReg.Recipients) > 0 {
		fmt.Printf("  (per-file restriction)\n")
		for _, r := range fileReg.Recipients {
			fmt.Printf("  - %s\n", r)
		}
	} else {
		fmt.Printf("  (all vault users)\n")
		v, _ := config.LoadVault(s, vault)
		if v != nil {
			for _, u := range v.Users {
				fmt.Printf("  - %s\n", u.Email)
			}
		}
	}
	fmt.Println()

	plainPath := filepath.Join(s.Root(), relPath)
	encPath := plainPath + ".enc"

	plainExists := fileExists(plainPath)
	encExists := fileExists(encPath)

	fmt.Printf("Status:\n")
	fmt.Printf("  Plaintext: ")
	if plainExists {
		info, _ := os.Stat(plainPath)
		fmt.Printf("exists (%d bytes)\n", info.Size())
	} else {
		fmt.Printf("not present\n")
	}

	fmt.Printf("  Encrypted: ")
	if encExists {
		info, _ := os.Stat(encPath)
		fmt.Printf("exists (%d bytes)\n", info.Size())

		content, err := os.ReadFile(encPath)
		if err == nil {
			meta, _ := crypto.GetFileMetadata(content, relPath)
			if meta != nil {
				fmt.Printf("    Version: %s\n", meta.Version)
				fmt.Printf("    Encrypted: %s\n", meta.EncryptedAt.Format("2006-01-02 15:04:05"))
				if len(meta.Recipients) > 0 {
					fmt.Printf("    Recipients: %s\n", strings.Join(meta.Recipients, ", "))
				}
			}
		}
	} else {
		fmt.Printf("not present\n")
	}

	return nil
}
