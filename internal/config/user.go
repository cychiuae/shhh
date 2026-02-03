package config

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"

	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/store"
)

func ValidateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	if len(email) > 254 {
		return fmt.Errorf("email too long")
	}

	safePattern := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !safePattern.MatchString(email) {
		return fmt.Errorf("email contains invalid characters")
	}

	return nil
}

func AddUser(s *store.Store, vaultName, email string) (*User, error) {
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	gpg := crypto.GetProvider()
	keyInfo, err := gpg.LookupKey(email)
	if err != nil {
		return nil, fmt.Errorf("failed to find GPG key for %s: %w", email, err)
	}

	if keyInfo.IsExpired {
		return nil, fmt.Errorf("GPG key for %s has expired", email)
	}

	pubKey, err := gpg.GetPublicKey(email)
	if err != nil {
		return nil, fmt.Errorf("failed to export public key: %w", err)
	}

	pubKeyPath := s.PubkeyPath(email)
	if err := store.WriteFile(pubKeyPath, pubKey); err != nil {
		return nil, fmt.Errorf("failed to cache public key: %w", err)
	}

	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault: %w", err)
	}

	user := User{
		Email:       email,
		KeyID:       keyInfo.KeyID,
		Fingerprint: keyInfo.Fingerprint,
		ExpiresAt:   keyInfo.ExpiresAt,
		AddedAt:     time.Now(),
	}

	vault.AddUser(user)

	if err := vault.Save(s, vaultName); err != nil {
		return nil, fmt.Errorf("failed to save vault: %w", err)
	}

	return &user, nil
}

func RemoveUser(s *store.Store, vaultName, email string) error {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}

	if !vault.RemoveUser(email) {
		return fmt.Errorf("user %s not found in vault %s", email, vaultName)
	}

	if err := vault.Save(s, vaultName); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	return nil
}

func CheckUserKeys(s *store.Store, vaultName string) ([]UserKeyStatus, error) {
	vault, err := LoadVault(s, vaultName)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault: %w", err)
	}

	gpg := crypto.GetProvider()
	var statuses []UserKeyStatus

	for _, user := range vault.Users {
		status := UserKeyStatus{
			Email:       user.Email,
			Fingerprint: user.Fingerprint,
		}

		keyInfo, err := gpg.LookupKey(user.Email)
		if err != nil {
			status.Status = "missing"
			status.Message = "Key not found in keyring"
		} else if keyInfo.Fingerprint != user.Fingerprint {
			status.Status = "changed"
			status.Message = "Key fingerprint has changed"
		} else if keyInfo.IsExpired {
			status.Status = "expired"
			status.Message = "Key has expired"
		} else if crypto.IsExpiringSoon(keyInfo.ExpiresAt, 30) {
			status.Status = "expiring"
			if keyInfo.ExpiresAt != nil {
				status.Message = fmt.Sprintf("Key expires on %s", keyInfo.ExpiresAt.Format("2006-01-02"))
			}
		} else {
			status.Status = "valid"
			status.Message = "Key is valid"
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

type UserKeyStatus struct {
	Email       string
	Fingerprint string
	Status      string
	Message     string
}
