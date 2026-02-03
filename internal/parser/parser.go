package parser

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	EncPrefix      = "ENC[v1:"
	EncSuffix      = "]"
	MaxNestingDepth = 100
	MaxFileSize     = 50 * 1024 * 1024 // 50MB
)

var encPattern = regexp.MustCompile(`^ENC\[v1:([A-Za-z0-9+/=\s]+)\]$`)

type EncryptFunc func(plaintext string) (string, error)
type DecryptFunc func(ciphertext string) (string, error)

type Parser interface {
	EncryptValues(content []byte, encrypt EncryptFunc) ([]byte, error)
	DecryptValues(content []byte, decrypt DecryptFunc) ([]byte, error)
	FileType() string
}

func EncodeValue(encryptedData []byte) string {
	return EncPrefix + string(encryptedData) + EncSuffix
}

func DecodeValue(encoded string) ([]byte, bool) {
	matches := encPattern.FindStringSubmatch(encoded)
	if len(matches) != 2 {
		return nil, false
	}
	cleaned := strings.ReplaceAll(matches[1], "\n", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return []byte(cleaned), true
}

func IsEncrypted(value string) bool {
	return encPattern.MatchString(value)
}

func ValidateContentSize(content []byte) error {
	if len(content) > MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", len(content), MaxFileSize)
	}
	return nil
}
