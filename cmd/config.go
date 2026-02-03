package cmd

import (
	"fmt"
	"sort"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage project configuration",
	Long:  `Get, set, or list project configuration values.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE:  runConfigList,
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	cfg, err := config.Load(s)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	key := args[0]
	value, ok := cfg.Get(key)
	if !ok {
		return fmt.Errorf("unknown config key: %s", key)
	}

	fmt.Println(value)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	cfg, err := config.Load(s)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	key, value := args[0], args[1]
	if !cfg.Set(key, value) {
		return fmt.Errorf("unknown or read-only config key: %s", key)
	}

	if err := cfg.Save(s); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	cfg, err := config.Load(s)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	values := cfg.List()
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%s = %s\n", k, values[k])
	}

	return nil
}
