package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cychiuae/shhh/internal/store"
)

const (
	ModeValues = "values"
	ModeFull   = "full"
)

func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return fmt.Errorf("path must be relative")
	}

	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("path cannot traverse parent directories")
	}

	if strings.Contains(cleaned, ".shhh") {
		return fmt.Errorf("cannot register files inside .shhh directory")
	}

	return nil
}

func RegisterFile(s *store.Store, vaultName, path string, mode string, recipients []string) error {
	if err := ValidateFilePath(path); err != nil {
		return err
	}

	if mode != ModeValues && mode != ModeFull {
		return fmt.Errorf("invalid mode: %s (must be 'values' or 'full')", mode)
	}

	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	for _, r := range recipients {
		if !vault.HasUser(r) {
			return fmt.Errorf("recipient %s is not a user in vault %s", r, vaultName)
		}
	}

	file := RegisteredFile{
		Path:         path,
		Mode:         mode,
		GPGCopy:      false,
		Recipients:   recipients,
		RegisteredAt: time.Now(),
	}

	vault.RegisterFile(file)

	if err := vault.Save(s, vaultName); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	return nil
}

func UnregisterFile(s *store.Store, vaultName, path string) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if !vault.UnregisterFile(path) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	if err := vault.Save(s, vaultName); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	return nil
}

func FindFileVault(s *store.Store, path string) (string, *RegisteredFile, error) {
	vaults, err := s.ListVaults()
	if err != nil {
		return "", nil, err
	}

	for _, vaultName := range vaults {
		vault, err := LoadVault(s, vaultName)
		if err != nil {
			continue
		}

		if f := vault.GetFile(path); f != nil {
			return vaultName, f, nil
		}
	}

	return "", nil, fmt.Errorf("file %s not registered in any vault", path)
}

func GetEffectiveRecipients(s *store.Store, vaultName string, file *RegisteredFile) ([]string, error) {
	if len(file.Recipients) > 0 {
		return file.Recipients, nil
	}

	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return nil, err
	}

	return vault.Emails(), nil
}

func SetFileRecipients(s *store.Store, vaultName, path string, recipients []string) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	for _, r := range recipients {
		if !vault.HasUser(r) {
			return fmt.Errorf("recipient %s is not a user in vault %s", r, vaultName)
		}
	}

	if !vault.UpdateFile(path, func(f *RegisteredFile) {
		f.Recipients = recipients
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	return vault.Save(s, vaultName)
}

func ClearFileRecipients(s *store.Store, vaultName, path string) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if !vault.UpdateFile(path, func(f *RegisteredFile) {
		f.Recipients = nil
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	return vault.Save(s, vaultName)
}

func SetFileMode(s *store.Store, vaultName, path, mode string) error {
	if mode != ModeValues && mode != ModeFull {
		return fmt.Errorf("invalid mode: %s (must be 'values' or 'full')", mode)
	}

	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if !vault.UpdateFile(path, func(f *RegisteredFile) {
		f.Mode = mode
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	return vault.Save(s, vaultName)
}

func SetFileGPGCopy(s *store.Store, vaultName, path string, gpgCopy bool) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if !vault.UpdateFile(path, func(f *RegisteredFile) {
		f.GPGCopy = gpgCopy
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	return vault.Save(s, vaultName)
}

func AddFileRecipients(s *store.Store, vaultName, path string, recipients []string) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	for _, r := range recipients {
		if !vault.HasUser(r) {
			return fmt.Errorf("recipient %s is not a user in vault %s", r, vaultName)
		}
	}

	if !vault.UpdateFile(path, func(f *RegisteredFile) {
		for _, r := range recipients {
			found := false
			for _, existing := range f.Recipients {
				if existing == r {
					found = true
					break
				}
			}
			if !found {
				f.Recipients = append(f.Recipients, r)
			}
		}
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	return vault.Save(s, vaultName)
}

func RemoveFileRecipients(s *store.Store, vaultName, path string, recipients []string) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if !vault.UpdateFile(path, func(f *RegisteredFile) {
		newRecipients := make([]string, 0, len(f.Recipients))
		for _, existing := range f.Recipients {
			remove := false
			for _, r := range recipients {
				if existing == r {
					remove = true
					break
				}
			}
			if !remove {
				newRecipients = append(newRecipients, existing)
			}
		}
		f.Recipients = newRecipients
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vaultName)
	}

	return vault.Save(s, vaultName)
}
