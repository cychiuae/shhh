package security

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/cychiuae/shhh/internal/crypto"
	"github.com/cychiuae/shhh/internal/parser"
)

func TestEncryptedOutputNotPlaintext(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	plaintext := "supersecret123"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if strings.Contains(encrypted, plaintext) {
		t.Error("encrypted output contains plaintext")
	}

	if !strings.HasPrefix(encrypted, parser.EncPrefix) {
		t.Errorf("encrypted output should start with %s", parser.EncPrefix)
	}
}

func TestDecryptionRequiresCorrectKey(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	plaintext := "supersecret123"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := crypto.DecryptValue(encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted value %q does not match plaintext %q", decrypted, plaintext)
	}
}

func TestDifferentPlaintextsDifferentCiphertexts(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	plaintext1 := "secret1"
	plaintext2 := "secret2"

	encrypted1, err := crypto.EncryptValue(plaintext1, []string{"alice@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	encrypted2, err := crypto.EncryptValue(plaintext2, []string{"alice@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encrypted1 == encrypted2 {
		t.Error("different plaintexts produced identical ciphertexts")
	}
}

func TestTamperedCiphertextFailsDecryption(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	plaintext := "supersecret123"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Tamper with the ciphertext by modifying some characters
	tampered := encrypted[:len(encrypted)/2] + "TAMPERED" + encrypted[len(encrypted)/2+8:]

	_, err = crypto.DecryptValue(tampered)
	if err == nil {
		t.Error("decryption should fail for tampered ciphertext")
	}
}

func TestMultiRecipientEncryption(t *testing.T) {
	gpg, cleanup := setupTestGPGWithBob(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	plaintext := "shared-secret"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com", "bob@test.com"})
	if err != nil {
		t.Fatalf("multi-recipient encryption failed: %v", err)
	}

	decrypted, err := crypto.DecryptValue(encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted value %q does not match plaintext %q", decrypted, plaintext)
	}
}

func TestFullFileEncryption(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	content := []byte("This is a secret file content\nwith multiple lines\nand sensitive data: password123")

	opts := crypto.EncryptOptions{
		Vault:      "default",
		Mode:       "full",
		Recipients: []string{"alice@test.com"},
	}

	encrypted, err := crypto.EncryptFileContent(content, "test.txt", opts)
	if err != nil {
		t.Fatalf("full file encryption failed: %v", err)
	}

	if !bytes.HasPrefix(encrypted, []byte(crypto.FullFileHeader)) {
		t.Error("full file encryption should produce shhh header")
	}

	if bytes.Contains(encrypted, []byte("password123")) {
		t.Error("encrypted file contains plaintext secret")
	}

	decrypted, err := crypto.DecryptFileContent(encrypted, "test.txt")
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Error("decrypted content does not match original")
	}
}

func TestValueModePreservesStructure(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	content := []byte(`database:
  host: localhost
  password: secret123
`)

	opts := crypto.EncryptOptions{
		Vault:      "default",
		Mode:       "values",
		Recipients: []string{"alice@test.com"},
	}

	encrypted, err := crypto.EncryptFileContent(content, "test.yaml", opts)
	if err != nil {
		t.Fatalf("value mode encryption failed: %v", err)
	}

	if !bytes.Contains(encrypted, []byte("database:")) {
		t.Error("value mode should preserve YAML structure keys")
	}

	if !bytes.Contains(encrypted, []byte("host:")) {
		t.Error("value mode should preserve YAML structure keys")
	}

	if bytes.Contains(encrypted, []byte("secret123")) {
		t.Error("value mode should encrypt the secret value")
	}
}

// Test helper functions

func setupTestGPG(t *testing.T) (*crypto.NativeGPG, func()) {
	t.Helper()

	gpg := crypto.NewNativeGPG()

	entity, err := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create test entity: %v", err)
	}

	gpg.AddEntity(entity)

	return gpg, func() {
		crypto.SetProvider(nil)
	}
}

func setupTestGPGWithBob(t *testing.T) (*crypto.NativeGPG, func()) {
	t.Helper()

	gpg := crypto.NewNativeGPG()

	alice, err := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create alice entity: %v", err)
	}
	gpg.AddEntity(alice)

	bob, err := openpgp.NewEntity("Bob", "Test User", "bob@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create bob entity: %v", err)
	}
	gpg.AddEntity(bob)

	return gpg, func() {
		crypto.SetProvider(nil)
	}
}
