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
	encryptVault string
	encryptAll   bool
)

func init() {
	rootCmd.AddCommand(encryptCmd)

	encryptCmd.Flags().StringVarP(&encryptVault, "vault", "v", "", "Encrypt files in specific vault")
	encryptCmd.Flags().BoolVarP(&encryptAll, "all", "a", false, "Encrypt all registered files")
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt [file]",
	Short: "Encrypt a file or all registered files",
	Long: `Encrypt a registered file to its .enc counterpart.

Use --vault to encrypt all files in a specific vault.
Use --all to encrypt all registered files across all vaults.`,
	RunE: runEncrypt,
}

func runEncrypt(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	if encryptAll {
		return encryptAllFiles(s)
	}

	if encryptVault != "" {
		return encryptVaultFiles(s, encryptVault)
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a file, --vault, or --all")
	}

	return encryptSingleFile(s, args[0])
}

func encryptSingleFile(s *store.Store, filePath string) error {
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

	return encryptFile(s, vault, fileReg)
}

func encryptVaultFiles(s *store.Store, vaultName string) error {
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
	for _, f := range vault.Files {
		if err := encryptFile(s, vaultName, &f); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.Path, err))
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return fmt.Errorf("%d file(s) failed to encrypt", len(errs))
	}

	return nil
}

func encryptAllFiles(s *store.Store) error {
	vaults, err := s.ListVaults()
	if err != nil {
		return err
	}

	totalFiles := 0
	var errs []error

	for _, vaultName := range vaults {
		vault, err := config.LoadVault(s, vaultName)
		if err != nil {
			continue
		}

		for _, f := range vault.Files {
			totalFiles++
			if err := encryptFile(s, vaultName, &f); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): %w", f.Path, vaultName, err))
			}
		}
	}

	if totalFiles == 0 {
		fmt.Println("No files registered")
		return nil
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return fmt.Errorf("%d file(s) failed to encrypt", len(errs))
	}

	return nil
}

func encryptFile(s *store.Store, vault string, fileReg *config.RegisteredFile) error {
	plainPath := filepath.Join(s.Root(), fileReg.Path)
	encPath := plainPath + ".enc"

	if _, err := os.Stat(plainPath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist")
	}

	content, err := os.ReadFile(plainPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	recipients, err := config.GetEffectiveRecipients(s, vault, fileReg)
	if err != nil {
		return fmt.Errorf("failed to get recipients: %w", err)
	}

	if len(recipients) == 0 {
		return fmt.Errorf("no recipients available (add users to vault)")
	}

	opts := crypto.EncryptOptions{
		Vault:      vault,
		Mode:       fileReg.Mode,
		Recipients: recipients,
	}

	encrypted, err := crypto.EncryptFileContent(content, fileReg.Path, opts)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	if err := os.WriteFile(encPath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Printf("Encrypted %s -> %s.enc\n", fileReg.Path, fileReg.Path)

	if config.GetEffectiveGPGCopy(s, fileReg) {
		gpgPath := plainPath + ".gpg"
		gpg := crypto.GetProvider()
		gpgEncrypted, err := gpg.Encrypt(content, recipients)
		if err == nil {
			if err := os.WriteFile(gpgPath, gpgEncrypted, 0600); err == nil {
				fmt.Printf("  Created GPG backup: %s.gpg\n", fileReg.Path)
			}
		}
	}

	return nil
}
