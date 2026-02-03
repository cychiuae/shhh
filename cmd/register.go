package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/gitignore"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

var (
	registerVault      string
	registerMode       string
	registerRecipients []string
	registerNoEncrypt  bool
)

func init() {
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(unregisterCmd)

	registerCmd.Flags().StringVarP(&registerVault, "vault", "v", "", "Vault to register file in")
	registerCmd.Flags().StringVarP(&registerMode, "mode", "m", "values", "Encryption mode: values or full")
	registerCmd.Flags().StringSliceVarP(&registerRecipients, "recipients", "r", nil, "Specific recipients (default: all vault users)")
	registerCmd.Flags().BoolVar(&registerNoEncrypt, "no-encrypt", false, "Skip automatic encryption after registration")

	unregisterCmd.Flags().StringVarP(&registerVault, "vault", "v", "", "Vault to unregister file from")
}

var registerCmd = &cobra.Command{
	Use:   "register <file>",
	Short: "Register a file for encryption",
	Long: `Register a file to be managed by shhh.

The file will be encrypted automatically after registration.
Use --no-encrypt to skip automatic encryption.
The file will be added to .gitignore automatically.
By default, all vault users can decrypt the file.
Use --recipients to restrict access to specific users.`,
	Args: cobra.ExactArgs(1),
	RunE: runRegister,
}

var unregisterCmd = &cobra.Command{
	Use:   "unregister <file>",
	Short: "Unregister a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runUnregister,
}

func runRegister(cmd *cobra.Command, args []string) error {
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

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	vault := registerVault
	if vault == "" {
		cfg, err := config.Load(s)
		if err != nil {
			return err
		}
		vault = cfg.DefaultVault
	}

	if !s.VaultExists(vault) {
		return fmt.Errorf("vault %q does not exist", vault)
	}

	if err := config.RegisterFile(s, vault, relPath, registerMode, registerRecipients); err != nil {
		return err
	}

	if err := gitignore.EnsureIgnored(s.Root(), relPath); err != nil {
		fmt.Printf("Warning: failed to add to .gitignore: %v\n", err)
	}

	fmt.Printf("Registered %s in vault %s\n", relPath, vault)
	fmt.Printf("  Mode: %s\n", registerMode)
	if len(registerRecipients) > 0 {
		fmt.Printf("  Recipients: %v\n", registerRecipients)
	} else {
		fmt.Println("  Recipients: all vault users")
	}

	// Auto-encrypt unless --no-encrypt is specified
	if !registerNoEncrypt {
		files, err := config.LoadVaultFiles(s, vault)
		if err == nil {
			if fileReg := files.Get(relPath); fileReg != nil {
				if err := encryptFile(s, vault, fileReg); err != nil {
					fmt.Printf("Warning: encryption failed: %v\n", err)
					fmt.Println("Run 'shhh encrypt' manually after resolving the issue")
				}
			}
		}
	}

	return nil
}

func runUnregister(cmd *cobra.Command, args []string) error {
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

	vault := registerVault
	if vault == "" {
		foundVault, _, err := config.FindFileVault(s, relPath)
		if err != nil {
			return err
		}
		vault = foundVault
	}

	if err := config.UnregisterFile(s, vault, relPath); err != nil {
		return err
	}

	fmt.Printf("Unregistered %s from vault %s\n", relPath, vault)
	return nil
}
