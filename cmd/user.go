package cmd

import (
	"fmt"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

var userVault string

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userRemoveCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCheckCmd)

	userCmd.PersistentFlags().StringVarP(&userVault, "vault", "v", "", "Vault to operate on (default: default vault)")
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users in a vault",
	Long:  `Add, remove, or list users who can encrypt/decrypt secrets in a vault.`,
}

var userAddCmd = &cobra.Command{
	Use:   "add <email>",
	Short: "Add a user to a vault",
	Long: `Add a user by their GPG email address.

The user's GPG public key must be available in the local keyring.
The key will be cached in .shhh/pubkeys/ for other team members.`,
	Args: cobra.ExactArgs(1),
	RunE: runUserAdd,
}

var userRemoveCmd = &cobra.Command{
	Use:   "remove <email>",
	Short: "Remove a user from a vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserRemove,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users in a vault",
	RunE:  runUserList,
}

var userCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify all user keys are valid",
	Long: `Check the status of all user keys in a vault.

Reports on:
- Missing keys (not in local keyring)
- Changed keys (fingerprint mismatch)
- Expired keys
- Keys expiring within 30 days`,
	RunE: runUserCheck,
}

func getVault(s *store.Store) (string, error) {
	if userVault != "" {
		if !s.VaultExists(userVault) {
			return "", fmt.Errorf("vault %q does not exist", userVault)
		}
		return userVault, nil
	}

	cfg, err := config.Load(s)
	if err != nil {
		return "", err
	}
	return cfg.DefaultVault, nil
}

func runUserAdd(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	vault, err := getVault(s)
	if err != nil {
		return err
	}

	email := args[0]
	user, err := config.AddUser(s, vault, email)
	if err != nil {
		return err
	}

	fmt.Printf("Added user %s to vault %s\n", email, vault)
	fmt.Printf("  Key ID: %s\n", user.KeyID)
	fmt.Printf("  Fingerprint: %s\n", user.Fingerprint)
	if user.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", user.ExpiresAt.Format("2006-01-02"))
	} else {
		fmt.Println("  Expires: never")
	}
	fmt.Println("Note: Run 'shhh reencrypt' to grant access to existing secrets")

	return nil
}

func runUserRemove(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	vault, err := getVault(s)
	if err != nil {
		return err
	}

	email := args[0]
	if err := config.RemoveUser(s, vault, email); err != nil {
		return err
	}

	fmt.Printf("Removed user %s from vault %s\n", email, vault)
	fmt.Println("Note: Run 'shhh reencrypt' to remove their access to existing secrets")
	return nil
}

func runUserList(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	vault, err := getVault(s)
	if err != nil {
		return err
	}

	v, err := config.LoadVault(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if len(v.Users) == 0 {
		fmt.Printf("No users in vault %s\n", vault)
		return nil
	}

	fmt.Printf("Users in vault %s:\n\n", vault)

	for _, u := range v.Users {
		status := "valid"
		if u.ExpiresAt != nil {
			if crypto.IsExpired(u.ExpiresAt) {
				status = "EXPIRED"
			} else if crypto.IsExpiringSoon(u.ExpiresAt, 30) {
				status = "expiring soon"
			}
		}

		fmt.Printf("  %s\n", u.Email)
		fmt.Printf("    Key ID: %s\n", u.KeyID)
		fmt.Printf("    Fingerprint: %s\n", u.Fingerprint)
		if u.ExpiresAt != nil {
			fmt.Printf("    Expires: %s (%s)\n", u.ExpiresAt.Format("2006-01-02"), status)
		} else {
			fmt.Printf("    Expires: never\n")
		}
		fmt.Printf("    Added: %s\n", u.AddedAt.Format("2006-01-02"))
		fmt.Println()
	}

	return nil
}

func runUserCheck(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	vault, err := getVault(s)
	if err != nil {
		return err
	}

	statuses, err := config.CheckUserKeys(s, vault)
	if err != nil {
		return err
	}

	if len(statuses) == 0 {
		fmt.Printf("No users in vault %s\n", vault)
		return nil
	}

	fmt.Printf("Key status for vault %s:\n\n", vault)

	hasIssues := false
	for _, status := range statuses {
		icon := "✓"
		if status.Status != "valid" {
			icon = "✗"
			hasIssues = true
		}
		if status.Status == "expiring" {
			icon = "!"
		}

		fmt.Printf("  %s %s: %s\n", icon, status.Email, status.Message)
	}

	if hasIssues {
		fmt.Println("\nSome keys have issues. Please address them before encrypting.")
		return fmt.Errorf("key validation failed")
	}

	fmt.Println("\nAll keys are valid.")
	return nil
}
