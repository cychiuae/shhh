package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

var vaultForce bool

func init() {
	rootCmd.AddCommand(vaultCmd)
	vaultCmd.AddCommand(vaultCreateCmd)
	vaultCmd.AddCommand(vaultRemoveCmd)
	vaultCmd.AddCommand(vaultListCmd)

	vaultRemoveCmd.Flags().BoolVarP(&vaultForce, "force", "f", false, "Skip confirmation")
}

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage vaults",
	Long:  `Create, remove, or list vaults for organizing secrets.`,
}

var vaultCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultCreate,
}

var vaultRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultRemove,
}

var vaultListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vaults",
	RunE:  runVaultList,
}

func runVaultCreate(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	name := args[0]
	if err := s.CreateVault(name); err != nil {
		return err
	}

	usersData := config.NewVaultUsers()
	if err := usersData.Save(s, name); err != nil {
		return fmt.Errorf("failed to initialize users: %w", err)
	}

	filesData := config.NewVaultFiles()
	if err := filesData.Save(s, name); err != nil {
		return fmt.Errorf("failed to initialize files: %w", err)
	}

	fmt.Printf("Created vault %q\n", name)
	return nil
}

func runVaultRemove(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	name := args[0]

	if name == store.DefaultVault {
		return fmt.Errorf("cannot remove default vault")
	}

	if !s.VaultExists(name) {
		return fmt.Errorf("vault %q does not exist", name)
	}

	files, err := config.LoadVaultFiles(s, name)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !vaultForce {
		fileCount := len(files.Files)
		if fileCount > 0 {
			fmt.Printf("Vault %q contains %d registered file(s).\n", name, fileCount)
		}
		fmt.Printf("Are you sure you want to remove vault %q? [y/N] ", name)

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Aborted")
			return nil
		}
	}

	if err := s.RemoveVault(name); err != nil {
		return err
	}

	fmt.Printf("Removed vault %q\n", name)
	return nil
}

func runVaultList(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	cfg, err := config.Load(s)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	vaults, err := s.ListVaults()
	if err != nil {
		return err
	}

	if len(vaults) == 0 {
		fmt.Println("No vaults found")
		return nil
	}

	for _, v := range vaults {
		users, _ := config.LoadVaultUsers(s, v)
		files, _ := config.LoadVaultFiles(s, v)

		userCount := 0
		fileCount := 0
		if users != nil {
			userCount = len(users.Users)
		}
		if files != nil {
			fileCount = len(files.Files)
		}

		marker := " "
		if v == cfg.DefaultVault {
			marker = "*"
		}

		fmt.Printf("%s %s (%d users, %d files)\n", marker, v, userCount, fileCount)
	}

	return nil
}
