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

func RegisterFile(s *store.Store, vault, path string, mode string, recipients []string) error {
	if err := ValidateFilePath(path); err != nil {
		return err
	}

	if mode != ModeValues && mode != ModeFull {
		return fmt.Errorf("invalid mode: %s (must be 'values' or 'full')", mode)
	}

	users, err := LoadVaultUsers(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault users: %w", err)
	}

	for _, r := range recipients {
		if !users.HasUser(r) {
			return fmt.Errorf("recipient %s is not a user in vault %s", r, vault)
		}
	}

	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	file := RegisteredFile{
		Path:         path,
		Mode:         mode,
		GPGCopy:      false,
		Recipients:   recipients,
		RegisteredAt: time.Now(),
	}

	files.Register(file)

	if err := files.Save(s, vault); err != nil {
		return fmt.Errorf("failed to save files: %w", err)
	}

	return nil
}

func UnregisterFile(s *store.Store, vault, path string) error {
	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Unregister(path) {
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	if err := files.Save(s, vault); err != nil {
		return fmt.Errorf("failed to save files: %w", err)
	}

	return nil
}

func FindFileVault(s *store.Store, path string) (string, *RegisteredFile, error) {
	vaults, err := s.ListVaults()
	if err != nil {
		return "", nil, err
	}

	for _, vault := range vaults {
		files, err := LoadVaultFiles(s, vault)
		if err != nil {
			continue
		}

		if f := files.Get(path); f != nil {
			return vault, f, nil
		}
	}

	return "", nil, fmt.Errorf("file %s not registered in any vault", path)
}

func GetEffectiveRecipients(s *store.Store, vault string, file *RegisteredFile) ([]string, error) {
	if len(file.Recipients) > 0 {
		return file.Recipients, nil
	}

	users, err := LoadVaultUsers(s, vault)
	if err != nil {
		return nil, err
	}

	return users.Emails(), nil
}

func SetFileRecipients(s *store.Store, vault, path string, recipients []string) error {
	users, err := LoadVaultUsers(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault users: %w", err)
	}

	for _, r := range recipients {
		if !users.HasUser(r) {
			return fmt.Errorf("recipient %s is not a user in vault %s", r, vault)
		}
	}

	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Update(path, func(f *RegisteredFile) {
		f.Recipients = recipients
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	return files.Save(s, vault)
}

func ClearFileRecipients(s *store.Store, vault, path string) error {
	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Update(path, func(f *RegisteredFile) {
		f.Recipients = nil
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	return files.Save(s, vault)
}

func SetFileMode(s *store.Store, vault, path, mode string) error {
	if mode != ModeValues && mode != ModeFull {
		return fmt.Errorf("invalid mode: %s (must be 'values' or 'full')", mode)
	}

	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Update(path, func(f *RegisteredFile) {
		f.Mode = mode
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	return files.Save(s, vault)
}

func SetFileGPGCopy(s *store.Store, vault, path string, gpgCopy bool) error {
	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Update(path, func(f *RegisteredFile) {
		f.GPGCopy = gpgCopy
	}) {
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	return files.Save(s, vault)
}

func AddFileRecipients(s *store.Store, vault, path string, recipients []string) error {
	users, err := LoadVaultUsers(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault users: %w", err)
	}

	for _, r := range recipients {
		if !users.HasUser(r) {
			return fmt.Errorf("recipient %s is not a user in vault %s", r, vault)
		}
	}

	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Update(path, func(f *RegisteredFile) {
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
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	return files.Save(s, vault)
}

func RemoveFileRecipients(s *store.Store, vault, path string, recipients []string) error {
	files, err := LoadVaultFiles(s, vault)
	if err != nil {
		return fmt.Errorf("failed to load vault files: %w", err)
	}

	if !files.Update(path, func(f *RegisteredFile) {
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
		return fmt.Errorf("file %s not registered in vault %s", path, vault)
	}

	return files.Save(s, vault)
}
