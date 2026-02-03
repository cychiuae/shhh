package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit <file>",
	Short: "Edit an encrypted file",
	Long: `Decrypt a file to a temporary location, open it in $EDITOR,
and re-encrypt when the editor closes.

The original encrypted file is only updated if changes were made.
Temporary files are securely cleaned up.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func runEdit(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	if err := crypto.LoadCachedPublicKeys(s.PubkeysPath()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load cached keys: %v\n", err)
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

	encPath := filepath.Join(s.Root(), relPath) + ".enc"
	if _, err := os.Stat(encPath); os.IsNotExist(err) {
		return fmt.Errorf("encrypted file does not exist: %s.enc", relPath)
	}

	encContent, err := os.ReadFile(encPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	decrypted, err := crypto.DecryptFileContent(encContent, relPath)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "shhh-edit-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up temp directory: %v\n", err)
		}
	}()

	if err := os.Chmod(tmpDir, 0700); err != nil {
		return fmt.Errorf("failed to set temp directory permissions: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, filepath.Base(relPath))
	if err := os.WriteFile(tmpFile, decrypted, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	editor := getEditor()
	if editor == "" {
		return fmt.Errorf("no editor found (set $EDITOR or $VISUAL)")
	}

	editorCmd := exec.Command(editor, tmpFile)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	editedContent, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	if string(editedContent) == string(decrypted) {
		fmt.Println("No changes made")
		return nil
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

	encrypted, err := crypto.EncryptFileContent(editedContent, relPath, opts)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	if err := os.WriteFile(encPath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Printf("Updated %s.enc\n", relPath)
	return nil
}

func getEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	editors := []string{"vim", "vi", "nano", "emacs"}
	for _, e := range editors {
		if path, err := exec.LookPath(e); err == nil {
			return path
		}
	}

	return ""
}
