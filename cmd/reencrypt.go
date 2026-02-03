package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

var (
	reencryptVault string
	reencryptAll   bool
)

func init() {
	rootCmd.AddCommand(reencryptCmd)

	reencryptCmd.Flags().StringVarP(&reencryptVault, "vault", "v", "", "Re-encrypt files in specific vault")
	reencryptCmd.Flags().BoolVarP(&reencryptAll, "all", "a", false, "Re-encrypt all registered files")
}

var reencryptCmd = &cobra.Command{
	Use:   "reencrypt [file]",
	Short: "Re-encrypt files with current recipients",
	Long: `Re-encrypt files using the current recipient list.

This is useful after:
- Adding or removing users from a vault
- Changing per-file recipient settings
- Rotating encryption keys

Use --vault to re-encrypt all files in a specific vault.
Use --all to re-encrypt all registered files.`,
	RunE: runReencrypt,
}

func runReencrypt(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	if err := crypto.LoadCachedPublicKeys(s.PubkeysPath()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load cached keys: %v\n", err)
	}

	if reencryptAll {
		return reencryptAllFiles(s)
	}

	if reencryptVault != "" {
		return reencryptVaultFiles(s, reencryptVault)
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a file, --vault, or --all")
	}

	return reencryptSingleFile(s, args[0])
}

func reencryptSingleFile(s *store.Store, filePath string) error {
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

	return reencryptFile(s, vault, fileReg)
}

func reencryptVaultFiles(s *store.Store, vaultName string) error {
	if !s.VaultExists(vaultName) {
		return fmt.Errorf("vault %q does not exist", vaultName)
	}

	vault, err := config.LoadVault(s, vaultName)
	if err != nil {
		return err
	}

	if len(vault.Files) == 0 {
		fmt.Printf("No files registered in vault %s\n", vaultName)
		return nil
	}

	var errs []error
	successCount := 0

	for _, f := range vault.Files {
		if err := reencryptFile(s, vaultName, &f); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.Path, err))
		} else {
			successCount++
		}
	}

	fmt.Printf("\nRe-encrypted %d file(s) in vault %s\n", successCount, vaultName)

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return fmt.Errorf("%d file(s) failed to re-encrypt", len(errs))
	}

	return nil
}

func reencryptAllFiles(s *store.Store) error {
	vaults, err := s.ListVaults()
	if err != nil {
		return err
	}

	totalFiles := 0
	successCount := 0
	var errs []error

	for _, vaultName := range vaults {
		vault, err := config.LoadVault(s, vaultName)
		if err != nil {
			continue
		}

		for _, f := range vault.Files {
			totalFiles++
			if err := reencryptFile(s, vaultName, &f); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): %w", f.Path, vaultName, err))
			} else {
				successCount++
			}
		}
	}

	if totalFiles == 0 {
		fmt.Println("No files registered")
		return nil
	}

	fmt.Printf("\nRe-encrypted %d of %d file(s)\n", successCount, totalFiles)

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return fmt.Errorf("%d file(s) failed to re-encrypt", len(errs))
	}

	return nil
}

func reencryptFile(s *store.Store, vault string, fileReg *config.RegisteredFile) error {
	encPath := filepath.Join(s.Root(), fileReg.Path) + ".enc"

	if _, err := os.Stat(encPath); os.IsNotExist(err) {
		return fmt.Errorf("encrypted file does not exist")
	}

	encContent, err := os.ReadFile(encPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	decrypted, err := crypto.DecryptFileContent(encContent, fileReg.Path)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	recipients, err := config.GetEffectiveRecipients(s, vault, fileReg)
	if err != nil {
		return fmt.Errorf("failed to get recipients: %w", err)
	}

	if len(recipients) == 0 {
		return fmt.Errorf("no recipients available")
	}

	opts := crypto.EncryptOptions{
		Vault:      vault,
		Mode:       fileReg.Mode,
		Recipients: recipients,
	}

	encrypted, err := crypto.EncryptFileContent(decrypted, fileReg.Path, opts)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	if err := os.WriteFile(encPath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Printf("Re-encrypted %s.enc\n", fileReg.Path)

	if config.GetEffectiveGPGCopy(s, fileReg) {
		gpgPath := filepath.Join(s.Root(), fileReg.Path) + ".gpg"
		gpg := crypto.GetProvider()
		gpgEncrypted, err := gpg.Encrypt(decrypted, recipients)
		if err == nil {
			if err := os.WriteFile(gpgPath, gpgEncrypted, 0600); err == nil {
				fmt.Printf("  Updated GPG backup: %s.gpg\n", fileReg.Path)
			}
		}
	}

	return nil
}
