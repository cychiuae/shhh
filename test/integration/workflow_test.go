package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/store"
)

func TestFullWorkflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	alice, err := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create alice entity: %v", err)
	}

	gpg := crypto.NewNativeGPG()
	gpg.AddEntity(alice)
	crypto.SetProvider(gpg)
	defer crypto.SetProvider(nil)

	s := store.New(tmpDir)
	if err := s.Initialize(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	cfg := config.NewConfig()
	if err := cfg.Save(s); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	vault := config.NewVault()
	if err := vault.Save(s, store.DefaultVault); err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}

	vault.AddUser(config.User{
		Email:       "alice@test.com",
		KeyID:       "TESTKEY",
		Fingerprint: "TESTFINGERPRINT",
	})
	if err := vault.Save(s, store.DefaultVault); err != nil {
		t.Fatalf("failed to save vault: %v", err)
	}

	secretContent := []byte(`database:
  host: localhost
  password: supersecret123
`)
	secretPath := filepath.Join(tmpDir, "secrets.yaml")
	if err := os.WriteFile(secretPath, secretContent, 0600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	if err := config.RegisterFile(s, store.DefaultVault, "secrets.yaml", "values", nil); err != nil {
		t.Fatalf("failed to register file: %v", err)
	}

	vault, _ = config.LoadVault(s, store.DefaultVault)
	fileReg := vault.GetFile("secrets.yaml")
	if fileReg == nil {
		t.Fatal("file not registered")
	}

	recipients, _ := config.GetEffectiveRecipients(s, store.DefaultVault, fileReg)

	opts := crypto.EncryptOptions{
		Vault:      store.DefaultVault,
		Mode:       fileReg.Mode,
		Recipients: recipients,
	}

	encrypted, err := crypto.EncryptFileContent(secretContent, "secrets.yaml", opts)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	encPath := secretPath + ".enc"
	if err := os.WriteFile(encPath, encrypted, 0600); err != nil {
		t.Fatalf("failed to write encrypted file: %v", err)
	}

	encContent, err := os.ReadFile(encPath)
	if err != nil {
		t.Fatalf("failed to read encrypted file: %v", err)
	}

	decrypted, err := crypto.DecryptFileContent(encContent, "secrets.yaml")
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decrypted) != string(secretContent) {
		t.Errorf("decrypted content does not match original")
	}
}

func TestMultiVaultWorkflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-multivault-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	alice, _ := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	bob, _ := openpgp.NewEntity("Bob", "Test User", "bob@test.com", nil)

	gpg := crypto.NewNativeGPG()
	gpg.AddEntity(alice)
	gpg.AddEntity(bob)
	crypto.SetProvider(gpg)
	defer crypto.SetProvider(nil)

	s := store.New(tmpDir)
	s.Initialize()

	config.NewConfig().Save(s)

	defaultVault := config.NewVault()
	defaultVault.Save(s, store.DefaultVault)

	s.CreateVault("production")
	prodVault := config.NewVault()
	prodVault.Save(s, "production")

	defaultVault.AddUser(config.User{Email: "alice@test.com", KeyID: "ALICE"})
	defaultVault.AddUser(config.User{Email: "bob@test.com", KeyID: "BOB"})
	defaultVault.Save(s, store.DefaultVault)

	prodVault.AddUser(config.User{Email: "alice@test.com", KeyID: "ALICE"})
	prodVault.Save(s, "production")

	devSecret := filepath.Join(tmpDir, "dev-secrets.yaml")
	os.WriteFile(devSecret, []byte("password: dev123"), 0600)
	config.RegisterFile(s, store.DefaultVault, "dev-secrets.yaml", "values", nil)

	prodSecret := filepath.Join(tmpDir, "prod-secrets.yaml")
	os.WriteFile(prodSecret, []byte("password: prod123"), 0600)
	config.RegisterFile(s, "production", "prod-secrets.yaml", "values", nil)

	devVault, _ := config.LoadVault(s, store.DefaultVault)
	if len(devVault.Files) != 1 {
		t.Errorf("expected 1 file in default vault, got %d", len(devVault.Files))
	}

	prodVault, _ = config.LoadVault(s, "production")
	if len(prodVault.Files) != 1 {
		t.Errorf("expected 1 file in production vault, got %d", len(prodVault.Files))
	}

	devRecipients, _ := config.GetEffectiveRecipients(s, store.DefaultVault, devVault.GetFile("dev-secrets.yaml"))
	if len(devRecipients) != 2 {
		t.Errorf("expected 2 recipients for dev, got %d", len(devRecipients))
	}

	prodRecipients, _ := config.GetEffectiveRecipients(s, "production", prodVault.GetFile("prod-secrets.yaml"))
	if len(prodRecipients) != 1 {
		t.Errorf("expected 1 recipient for prod, got %d", len(prodRecipients))
	}
}

func TestPerFileRecipients(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-perfile-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	alice, _ := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	bob, _ := openpgp.NewEntity("Bob", "Test User", "bob@test.com", nil)

	gpg := crypto.NewNativeGPG()
	gpg.AddEntity(alice)
	gpg.AddEntity(bob)
	crypto.SetProvider(gpg)
	defer crypto.SetProvider(nil)

	s := store.New(tmpDir)
	s.Initialize()
	config.NewConfig().Save(s)

	vault := config.NewVault()
	vault.AddUser(config.User{Email: "alice@test.com", KeyID: "ALICE"})
	vault.AddUser(config.User{Email: "bob@test.com", KeyID: "BOB"})
	vault.Save(s, store.DefaultVault)

	secretPath := filepath.Join(tmpDir, "secrets.yaml")
	os.WriteFile(secretPath, []byte("password: secret"), 0600)
	config.RegisterFile(s, store.DefaultVault, "secrets.yaml", "values", nil)

	config.SetFileRecipients(s, store.DefaultVault, "secrets.yaml", []string{"alice@test.com"})

	vault, _ = config.LoadVault(s, store.DefaultVault)
	fileReg := vault.GetFile("secrets.yaml")

	recipients, _ := config.GetEffectiveRecipients(s, store.DefaultVault, fileReg)

	if len(recipients) != 1 || recipients[0] != "alice@test.com" {
		t.Errorf("expected only alice as recipient, got %v", recipients)
	}

	config.ClearFileRecipients(s, store.DefaultVault, "secrets.yaml")

	vault, _ = config.LoadVault(s, store.DefaultVault)
	fileReg = vault.GetFile("secrets.yaml")
	recipients, _ = config.GetEffectiveRecipients(s, store.DefaultVault, fileReg)

	if len(recipients) != 2 {
		t.Errorf("expected 2 recipients after clearing, got %d", len(recipients))
	}
}

func TestFileStateDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-state-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	plainPath := filepath.Join(tmpDir, "secrets.yaml")
	encPath := plainPath + ".enc"

	getState := func() string {
		plainExists := fileExists(plainPath)
		encExists := fileExists(encPath)

		switch {
		case encExists && plainExists:
			return "decrypted"
		case encExists && !plainExists:
			return "encrypted"
		case !encExists && plainExists:
			return "pending"
		default:
			return "missing"
		}
	}

	if state := getState(); state != "missing" {
		t.Errorf("expected 'missing', got %q", state)
	}

	os.WriteFile(plainPath, []byte("secret"), 0600)
	if state := getState(); state != "pending" {
		t.Errorf("expected 'pending', got %q", state)
	}

	os.WriteFile(encPath, []byte("encrypted"), 0600)
	if state := getState(); state != "decrypted" {
		t.Errorf("expected 'decrypted', got %q", state)
	}

	os.Remove(plainPath)
	if state := getState(); state != "encrypted" {
		t.Errorf("expected 'encrypted', got %q", state)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
