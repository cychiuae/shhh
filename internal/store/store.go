package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ShhhDir       = ".shhh"
	ConfigFile    = "config.json"
	VaultsDir     = "vaults"
	PubkeysDir    = "pubkeys"
	UsersFile     = "users.json"
	FilesFile     = "files.json"
	DirPerms      = 0700
	FilePerms     = 0600
	DefaultVault  = "default"
)

var ErrNotInitialized = errors.New("shhh not initialized (run 'shhh init' first)")

type Store struct {
	root string
}

func New(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) ShhhPath() string {
	return filepath.Join(s.root, ShhhDir)
}

func (s *Store) ConfigPath() string {
	return filepath.Join(s.ShhhPath(), ConfigFile)
}

func (s *Store) VaultsPath() string {
	return filepath.Join(s.ShhhPath(), VaultsDir)
}

func (s *Store) VaultPath(name string) string {
	return filepath.Join(s.VaultsPath(), name)
}

func (s *Store) VaultUsersPath(vault string) string {
	return filepath.Join(s.VaultPath(vault), UsersFile)
}

func (s *Store) VaultFilesPath(vault string) string {
	return filepath.Join(s.VaultPath(vault), FilesFile)
}

func (s *Store) PubkeysPath() string {
	return filepath.Join(s.ShhhPath(), PubkeysDir)
}

func (s *Store) PubkeyPath(email string) string {
	return filepath.Join(s.PubkeysPath(), email+".asc")
}

func (s *Store) IsInitialized() bool {
	info, err := os.Stat(s.ShhhPath())
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (s *Store) Initialize() error {
	if s.IsInitialized() {
		return fmt.Errorf("shhh already initialized in %s", s.root)
	}

	dirs := []string{
		s.ShhhPath(),
		s.VaultsPath(),
		s.VaultPath(DefaultVault),
		s.PubkeysPath(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, DirPerms); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (s *Store) EnsureInitialized() error {
	if !s.IsInitialized() {
		return ErrNotInitialized
	}
	return nil
}

func (s *Store) CreateVault(name string) error {
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid vault name: %w", err)
	}

	vaultPath := s.VaultPath(name)
	if _, err := os.Stat(vaultPath); err == nil {
		return fmt.Errorf("vault %q already exists", name)
	}

	if err := os.MkdirAll(vaultPath, DirPerms); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	return nil
}

func (s *Store) RemoveVault(name string) error {
	if name == DefaultVault {
		return fmt.Errorf("cannot remove default vault")
	}

	vaultPath := s.VaultPath(name)
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		return fmt.Errorf("vault %q does not exist", name)
	}

	if err := os.RemoveAll(vaultPath); err != nil {
		return fmt.Errorf("failed to remove vault: %w", err)
	}

	return nil
}

func (s *Store) ListVaults() ([]string, error) {
	entries, err := os.ReadDir(s.VaultsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	var vaults []string
	for _, entry := range entries {
		if entry.IsDir() {
			vaults = append(vaults, entry.Name())
		}
	}

	return vaults, nil
}

func (s *Store) VaultExists(name string) bool {
	info, err := os.Stat(s.VaultPath(name))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func validateName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if name == "." || name == ".." {
		return errors.New("name cannot be . or ..")
	}
	for _, c := range name {
		if c == '/' || c == '\\' || c == '\x00' {
			return errors.New("name contains invalid characters")
		}
	}
	return nil
}

func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		shhhPath := filepath.Join(dir, ShhhDir)
		if info, err := os.Stat(shhhPath); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInitialized
		}
		dir = parent
	}
}

func GetStore() (*Store, error) {
	root, err := FindRoot()
	if err != nil {
		return nil, err
	}
	return New(root), nil
}

func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DirPerms); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(path, data, FilePerms); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}
