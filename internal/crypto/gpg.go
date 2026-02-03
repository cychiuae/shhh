package crypto

import (
	"errors"
	"time"
)

var (
	ErrKeyNotFound     = errors.New("GPG key not found")
	ErrKeyExpired      = errors.New("GPG key has expired")
	ErrInvalidKey      = errors.New("invalid GPG key")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrNoPrivateKey    = errors.New("no private key available for decryption")
)

type KeyInfo struct {
	Email       string
	KeyID       string
	Fingerprint string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	IsExpired   bool
	PublicKey   []byte
}

type GPGProvider interface {
	LookupKey(email string) (*KeyInfo, error)
	GetPublicKey(email string) ([]byte, error)
	Encrypt(data []byte, recipients []string) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	ImportPublicKey(armoredKey []byte) (*KeyInfo, error)
}

var defaultProvider GPGProvider

func GetProvider() GPGProvider {
	if defaultProvider == nil {
		native := NewNativeGPG()
		cli := NewCLIGPG()
		defaultProvider = &fallbackProvider{primary: native, fallback: cli}
	}
	return defaultProvider
}

func SetProvider(p GPGProvider) {
	defaultProvider = p
}

type fallbackProvider struct {
	primary  GPGProvider
	fallback GPGProvider
}

func (f *fallbackProvider) LookupKey(email string) (*KeyInfo, error) {
	key, err := f.primary.LookupKey(email)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return nil, err
	}
	return f.fallback.LookupKey(email)
}

func (f *fallbackProvider) GetPublicKey(email string) ([]byte, error) {
	key, err := f.primary.GetPublicKey(email)
	if err == nil {
		return key, nil
	}
	return f.fallback.GetPublicKey(email)
}

func (f *fallbackProvider) Encrypt(data []byte, recipients []string) ([]byte, error) {
	result, err := f.primary.Encrypt(data, recipients)
	if err == nil {
		return result, nil
	}
	return f.fallback.Encrypt(data, recipients)
}

func (f *fallbackProvider) Decrypt(data []byte) ([]byte, error) {
	result, err := f.primary.Decrypt(data)
	if err == nil {
		return result, nil
	}
	if errors.Is(err, ErrNoPrivateKey) {
		return f.fallback.Decrypt(data)
	}
	return nil, err
}

func (f *fallbackProvider) ImportPublicKey(armoredKey []byte) (*KeyInfo, error) {
	key, err := f.primary.ImportPublicKey(armoredKey)
	if err == nil {
		return key, nil
	}
	return f.fallback.ImportPublicKey(armoredKey)
}

func IsExpiringSoon(expiresAt *time.Time, days int) bool {
	if expiresAt == nil {
		return false
	}
	threshold := time.Now().AddDate(0, 0, days)
	return expiresAt.Before(threshold)
}

func IsExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return expiresAt.Before(time.Now())
}
