package crypto

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

type NativeGPG struct {
	keyring openpgp.EntityList
}

func NewNativeGPG() *NativeGPG {
	gpg := &NativeGPG{}
	gpg.loadKeyring()
	return gpg
}

func (g *NativeGPG) loadKeyring() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	gnupgHome := os.Getenv("GNUPGHOME")
	if gnupgHome == "" {
		gnupgHome = filepath.Join(home, ".gnupg")
	}

	pubringPath := filepath.Join(gnupgHome, "pubring.kbx")
	if _, err := os.Stat(pubringPath); os.IsNotExist(err) {
		pubringPath = filepath.Join(gnupgHome, "pubring.gpg")
	}

	pubFile, err := os.Open(pubringPath)
	if err != nil {
		return
	}
	defer pubFile.Close()

	keyring, _ := openpgp.ReadKeyRing(pubFile)
	if keyring != nil {
		g.keyring = keyring
	}

	secringPath := filepath.Join(gnupgHome, "secring.gpg")
	secFile, err := os.Open(secringPath)
	if err == nil {
		defer secFile.Close()
		secring, _ := openpgp.ReadKeyRing(secFile)
		if secring != nil {
			g.keyring = append(g.keyring, secring...)
		}
	}

	privateKeysDir := filepath.Join(gnupgHome, "private-keys-v1.d")
	if info, err := os.Stat(privateKeysDir); err == nil && info.IsDir() {
		// Modern GnuPG uses keybox format; we may not be able to read all keys
		// Fall back to CLI for these cases
	}
}

func (g *NativeGPG) LookupKey(email string) (*KeyInfo, error) {
	email = strings.ToLower(email)

	for _, entity := range g.keyring {
		for _, ident := range entity.Identities {
			if ident.UserId != nil && strings.ToLower(ident.UserId.Email) == email {
				return g.entityToKeyInfo(entity, email)
			}
		}
	}

	return nil, ErrKeyNotFound
}

func (g *NativeGPG) entityToKeyInfo(entity *openpgp.Entity, email string) (*KeyInfo, error) {
	pk := entity.PrimaryKey
	keyID := fmt.Sprintf("%X", pk.KeyId)
	fingerprint := fmt.Sprintf("%X", pk.Fingerprint)

	var expiresAt *time.Time
	isExpired := false

	for _, ident := range entity.Identities {
		if ident.SelfSignature != nil && ident.SelfSignature.KeyLifetimeSecs != nil {
			expiry := pk.CreationTime.Add(time.Duration(*ident.SelfSignature.KeyLifetimeSecs) * time.Second)
			expiresAt = &expiry
			if expiry.Before(time.Now()) {
				isExpired = true
			}
			break
		}
	}

	var pubKeyBuf bytes.Buffer
	armorWriter, err := armor.Encode(&pubKeyBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armor writer: %w", err)
	}
	if err := entity.Serialize(armorWriter); err != nil {
		armorWriter.Close()
		return nil, fmt.Errorf("failed to serialize public key: %w", err)
	}
	armorWriter.Close()

	return &KeyInfo{
		Email:       email,
		KeyID:       keyID,
		Fingerprint: fingerprint,
		ExpiresAt:   expiresAt,
		CreatedAt:   pk.CreationTime,
		IsExpired:   isExpired,
		PublicKey:   pubKeyBuf.Bytes(),
	}, nil
}

func (g *NativeGPG) GetPublicKey(email string) ([]byte, error) {
	info, err := g.LookupKey(email)
	if err != nil {
		return nil, err
	}
	return info.PublicKey, nil
}

func (g *NativeGPG) Encrypt(data []byte, recipients []string) ([]byte, error) {
	var entities []*openpgp.Entity

	for _, email := range recipients {
		email = strings.ToLower(email)
		found := false

		for _, entity := range g.keyring {
			for _, ident := range entity.Identities {
				if ident.UserId != nil && strings.ToLower(ident.UserId.Email) == email {
					entities = append(entities, entity)
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("key not found for recipient: %s", email)
		}
	}

	if len(entities) == 0 {
		return nil, errors.New("no valid recipients")
	}

	var buf bytes.Buffer
	armorWriter, err := armor.Encode(&buf, "PGP MESSAGE", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armor writer: %w", err)
	}

	config := &packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.CompressionZLIB,
	}

	plainWriter, err := openpgp.Encrypt(armorWriter, entities, nil, nil, config)
	if err != nil {
		armorWriter.Close()
		return nil, fmt.Errorf("failed to create encrypt writer: %w", err)
	}

	if _, err := plainWriter.Write(data); err != nil {
		plainWriter.Close()
		armorWriter.Close()
		return nil, fmt.Errorf("failed to write encrypted data: %w", err)
	}

	if err := plainWriter.Close(); err != nil {
		armorWriter.Close()
		return nil, fmt.Errorf("failed to close plain writer: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close armor writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *NativeGPG) Decrypt(data []byte) ([]byte, error) {
	block, err := armor.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode armor: %w", err)
	}

	var privateKeys openpgp.EntityList
	for _, entity := range g.keyring {
		if entity.PrivateKey != nil {
			privateKeys = append(privateKeys, entity)
		}
	}

	if len(privateKeys) == 0 {
		return nil, ErrNoPrivateKey
	}

	md, err := openpgp.ReadMessage(block.Body, privateKeys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted message: %w", err)
	}

	plaintext, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read plaintext: %w", err)
	}

	return plaintext, nil
}

func (g *NativeGPG) ImportPublicKey(armoredKey []byte) (*KeyInfo, error) {
	block, err := armor.Decode(bytes.NewReader(armoredKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decode armor: %w", err)
	}

	entities, err := openpgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	if len(entities) == 0 {
		return nil, ErrInvalidKey
	}

	entity := entities[0]
	g.keyring = append(g.keyring, entity)

	var email string
	for _, ident := range entity.Identities {
		if ident.UserId != nil && ident.UserId.Email != "" {
			email = ident.UserId.Email
			break
		}
	}

	return g.entityToKeyInfo(entity, email)
}

func (g *NativeGPG) AddEntity(entity *openpgp.Entity) {
	g.keyring = append(g.keyring, entity)
}

func (g *NativeGPG) GetKeyring() openpgp.EntityList {
	return g.keyring
}
