package security

import (
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/cychiuae/shhh/internal/crypto"
)

func TestNonRecipientCannotDecrypt(t *testing.T) {
	alice, err := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create alice entity: %v", err)
	}

	bob, err := openpgp.NewEntity("Bob", "Test User", "bob@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create bob entity: %v", err)
	}

	charlie, err := openpgp.NewEntity("Charlie", "Test User", "charlie@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create charlie entity: %v", err)
	}

	aliceGPG := crypto.NewNativeGPG()
	aliceGPG.AddEntity(alice)
	aliceGPG.AddEntity(bob)

	crypto.SetProvider(aliceGPG)

	plaintext := "alice-and-bob-secret"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com", "bob@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	charlieGPG := crypto.NewNativeGPG()
	charlieGPG.AddEntity(charlie)
	crypto.SetProvider(charlieGPG)

	_, err = crypto.DecryptValue(encrypted)
	if err == nil {
		t.Error("charlie should not be able to decrypt alice and bob's secret")
	}

	crypto.SetProvider(nil)
}

func TestRecipientCanDecrypt(t *testing.T) {
	alice, err := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create alice entity: %v", err)
	}

	bob, err := openpgp.NewEntity("Bob", "Test User", "bob@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create bob entity: %v", err)
	}

	encryptGPG := crypto.NewNativeGPG()
	encryptGPG.AddEntity(alice)
	encryptGPG.AddEntity(bob)

	crypto.SetProvider(encryptGPG)

	plaintext := "shared-secret"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com", "bob@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	aliceGPG := crypto.NewNativeGPG()
	aliceGPG.AddEntity(alice)
	crypto.SetProvider(aliceGPG)

	decrypted, err := crypto.DecryptValue(encrypted)
	if err != nil {
		t.Fatalf("alice should be able to decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("alice decrypted %q, expected %q", decrypted, plaintext)
	}

	bobGPG := crypto.NewNativeGPG()
	bobGPG.AddEntity(bob)
	crypto.SetProvider(bobGPG)

	decrypted, err = crypto.DecryptValue(encrypted)
	if err != nil {
		t.Fatalf("bob should be able to decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("bob decrypted %q, expected %q", decrypted, plaintext)
	}

	crypto.SetProvider(nil)
}

func TestEncryptionRequiresRecipients(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	_, err := crypto.EncryptValue("secret", []string{})
	if err == nil {
		t.Error("encryption should fail without recipients")
	}
}

func TestEncryptionFailsForUnknownRecipient(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	_, err := crypto.EncryptValue("secret", []string{"unknown@test.com"})
	if err == nil {
		t.Error("encryption should fail for unknown recipient")
	}
}

func TestSingleRecipientExcludesOthers(t *testing.T) {
	alice, err := openpgp.NewEntity("Alice", "Test User", "alice@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create alice entity: %v", err)
	}

	bob, err := openpgp.NewEntity("Bob", "Test User", "bob@test.com", nil)
	if err != nil {
		t.Fatalf("failed to create bob entity: %v", err)
	}

	gpg := crypto.NewNativeGPG()
	gpg.AddEntity(alice)
	gpg.AddEntity(bob)
	crypto.SetProvider(gpg)

	plaintext := "alice-only-secret"
	encrypted, err := crypto.EncryptValue(plaintext, []string{"alice@test.com"})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	aliceGPG := crypto.NewNativeGPG()
	aliceGPG.AddEntity(alice)
	crypto.SetProvider(aliceGPG)

	decrypted, err := crypto.DecryptValue(encrypted)
	if err != nil {
		t.Fatalf("alice should be able to decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("alice decrypted %q, expected %q", decrypted, plaintext)
	}

	bobGPG := crypto.NewNativeGPG()
	bobGPG.AddEntity(bob)
	crypto.SetProvider(bobGPG)

	_, err = crypto.DecryptValue(encrypted)
	if err == nil {
		t.Error("bob should not be able to decrypt alice-only secret")
	}

	crypto.SetProvider(nil)
}
