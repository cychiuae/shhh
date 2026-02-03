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

var (
	decryptVault string
	decryptAll   bool
	decryptForce bool
)

func init() {
	rootCmd.AddCommand(decryptCmd)

	decryptCmd.Flags().StringVarP(&decryptVault, "vault", "v", "", "Decrypt files in specific vault")
	decryptCmd.Flags().BoolVarP(&decryptAll, "all", "a", false, "Decrypt all registered files")
	decryptCmd.Flags().BoolVarP(&decryptForce, "force", "f", false, "Overwrite existing plaintext files")
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt [file]",
	Short: "Decrypt a file or all registered files",
	Long: `Decrypt an encrypted file to its plaintext form.

Use --vault to decrypt all files in a specific vault.
Use --all to decrypt all registered files across all vaults.
Use --force to overwrite existing plaintext files.`,
	RunE: runDecrypt,
}

func runDecrypt(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	if decryptAll {
		return decryptAllFiles(s)
	}

	if decryptVault != "" {
		return decryptVaultFiles(s, decryptVault)
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a file, --vault, or --all")
	}

	return decryptSingleFile(s, args[0])
}

func decryptSingleFile(s *store.Store, filePath string) error {
	filePath = strings.TrimSuffix(filePath, ".enc")

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

	return decryptFile(s, vault, fileReg)
}

func decryptVaultFiles(s *store.Store, vault string) error {
	if !s.VaultExists(vault) {
		return fmt.Errorf("vault %q does not exist", vault)
	}

	files, err := config.LoadVaultFiles(s, vault)
	if err != nil {
		return err
	}

	if len(files.Files) == 0 {
		fmt.Printf("No files registered in vault %s\n", vault)
		return nil
	}

	var errs []error
	for _, f := range files.Files {
		if err := decryptFile(s, vault, &f); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.Path, err))
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		return fmt.Errorf("%d file(s) failed to decrypt", len(errs))
	}

	return nil
}

func decryptAllFiles(s *store.Store) error {
	vaults, err := s.ListVaults()
	if err != nil {
		return err
	}

	totalFiles := 0
	var errs []error

	for _, vault := range vaults {
		files, err := config.LoadVaultFiles(s, vault)
		if err != nil {
			continue
		}

		for _, f := range files.Files {
			totalFiles++
			if err := decryptFile(s, vault, &f); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): %w", f.Path, vault, err))
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
		return fmt.Errorf("%d file(s) failed to decrypt", len(errs))
	}

	return nil
}

func decryptFile(s *store.Store, vault string, fileReg *config.RegisteredFile) error {
	plainPath := filepath.Join(s.Root(), fileReg.Path)
	encPath := plainPath + ".enc"

	if _, err := os.Stat(encPath); os.IsNotExist(err) {
		return fmt.Errorf("encrypted file does not exist: %s.enc", fileReg.Path)
	}

	if !decryptForce {
		if _, err := os.Stat(plainPath); err == nil {
			return fmt.Errorf("plaintext file already exists (use --force to overwrite)")
		}
	}

	content, err := os.ReadFile(encPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	decrypted, err := crypto.DecryptFileContent(content, fileReg.Path)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	if err := os.WriteFile(plainPath, decrypted, 0600); err != nil {
		return fmt.Errorf("failed to write plaintext file: %w", err)
	}

	fmt.Printf("Decrypted %s.enc -> %s\n", fileReg.Path, fileReg.Path)
	return nil
}
