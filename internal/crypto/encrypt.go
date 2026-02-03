package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/cychiuae/shhh/internal/parser"
)

const (
	FullFileHeader = "-----BEGIN SHHH ENCRYPTED FILE-----"
	FullFileFooter = "-----END SHHH ENCRYPTED FILE-----"
)

type EncryptOptions struct {
	Vault      string
	Mode       string
	Recipients []string
}

func EncryptValue(plaintext string, recipients []string) (string, error) {
	if len(recipients) == 0 {
		return "", fmt.Errorf("no recipients specified")
	}

	gpg := GetProvider()
	encrypted, err := gpg.Encrypt([]byte(plaintext), recipients)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)

	return parser.EncPrefix + encoded + parser.EncSuffix, nil
}

func DecryptValue(encoded string) (string, error) {
	if !parser.IsEncrypted(encoded) {
		return encoded, nil
	}

	data, ok := parser.DecodeValue(encoded)
	if !ok {
		return "", fmt.Errorf("invalid encrypted value format")
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	gpg := GetProvider()
	plaintext, err := gpg.Decrypt(decoded)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

func EncryptFileContent(content []byte, filename string, opts EncryptOptions) ([]byte, error) {
	if opts.Mode == "full" {
		return encryptFullFile(content, opts)
	}

	return encryptValuesFile(content, filename, opts)
}

func encryptValuesFile(content []byte, filename string, opts EncryptOptions) ([]byte, error) {
	p := parser.GetParserForFile(filename)
	if p == nil {
		// For unsupported file formats, encrypt the entire content
		return encryptFullFile(content, opts)
	}

	encryptFunc := func(plaintext string) (string, error) {
		return EncryptValue(plaintext, opts.Recipients)
	}

	encrypted, err := p.EncryptValues(content, encryptFunc)
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"version":      "1",
		"vault":        opts.Vault,
		"mode":         opts.Mode,
		"encrypted_at": time.Now().Format(time.RFC3339),
		"recipients":   strings.Join(opts.Recipients, ", "),
	}

	format := parser.DetectFormat(filename)
	switch format {
	case parser.FormatYAML:
		return parser.AddShhhMetadata(encrypted, metadata)
	case parser.FormatJSON:
		return parser.AddJSONMetadata(encrypted, metadata)
	case parser.FormatINI:
		return parser.AddINIMetadata(encrypted, metadata)
	case parser.FormatENV:
		return parser.AddENVMetadata(encrypted, metadata)
	default:
		return encrypted, nil
	}
}

func encryptFullFile(content []byte, opts EncryptOptions) ([]byte, error) {
	gpg := GetProvider()
	encrypted, err := gpg.Encrypt(content, opts.Recipients)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)

	var buf bytes.Buffer
	buf.WriteString(FullFileHeader + "\n")
	buf.WriteString(fmt.Sprintf("Version: 1\n"))
	buf.WriteString(fmt.Sprintf("Vault: %s\n", opts.Vault))
	buf.WriteString(fmt.Sprintf("Mode: full\n"))
	buf.WriteString(fmt.Sprintf("Recipients: %s\n", strings.Join(opts.Recipients, ", ")))
	buf.WriteString(fmt.Sprintf("Encrypted-At: %s\n", time.Now().Format(time.RFC3339)))
	buf.WriteString("\n")

	for i := 0; i < len(encoded); i += 64 {
		end := i + 64
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end] + "\n")
	}

	buf.WriteString(FullFileFooter + "\n")

	return buf.Bytes(), nil
}

func DecryptFileContent(content []byte, filename string) ([]byte, error) {
	if bytes.HasPrefix(content, []byte(FullFileHeader)) {
		return decryptFullFile(content)
	}

	return decryptValuesFile(content, filename)
}

func decryptValuesFile(content []byte, filename string) ([]byte, error) {
	p := parser.GetParserForFile(filename)
	if p == nil {
		return nil, fmt.Errorf("unsupported file format: %s", filename)
	}

	decrypted, err := p.DecryptValues(content, DecryptValue)
	if err != nil {
		return nil, err
	}

	format := parser.DetectFormat(filename)
	switch format {
	case parser.FormatYAML:
		return parser.RemoveShhhMetadata(decrypted)
	case parser.FormatJSON:
		return parser.RemoveJSONMetadata(decrypted)
	case parser.FormatINI:
		return parser.RemoveINIMetadata(decrypted)
	case parser.FormatENV:
		return parser.RemoveENVMetadata(decrypted)
	default:
		return decrypted, nil
	}
}

func decryptFullFile(content []byte) ([]byte, error) {
	lines := strings.Split(string(content), "\n")

	var encodedData strings.Builder
	inBody := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" && !inBody {
			inBody = true
			continue
		}

		if line == FullFileFooter {
			break
		}

		if inBody && line != "" {
			encodedData.WriteString(line)
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(encodedData.String())
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	gpg := GetProvider()
	plaintext, err := gpg.Decrypt(decoded)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

func IsFullyEncrypted(content []byte) bool {
	return bytes.HasPrefix(content, []byte(FullFileHeader))
}

type FileMetadata struct {
	Version     string
	Vault       string
	Mode        string
	Recipients  []string
	EncryptedAt time.Time
}

func GetFileMetadata(content []byte, filename string) (*FileMetadata, error) {
	if bytes.HasPrefix(content, []byte(FullFileHeader)) {
		return parseFullFileMetadata(content)
	}

	format := parser.DetectFormat(filename)

	var meta map[string]string
	var err error

	switch format {
	case parser.FormatYAML:
		meta, err = parser.GetShhhMetadata(content)
	case parser.FormatINI:
		meta, err = parser.GetINIMetadata(content)
	case parser.FormatENV:
		meta, err = parser.GetENVMetadata(content)
	case parser.FormatJSON:
		jsonMeta, jsonErr := parser.GetJSONMetadata(content)
		if jsonErr != nil {
			return nil, jsonErr
		}
		if jsonMeta != nil {
			meta = make(map[string]string)
			for k, v := range jsonMeta {
				meta[k] = fmt.Sprintf("%v", v)
			}
		}
	default:
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if meta == nil {
		return nil, nil
	}

	result := &FileMetadata{
		Version: meta["version"],
		Vault:   meta["vault"],
		Mode:    meta["mode"],
	}

	if recipients, ok := meta["recipients"]; ok && recipients != "" {
		parts := strings.Split(recipients, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result.Recipients = append(result.Recipients, p)
			}
		}
	}

	if encAt, ok := meta["encrypted_at"]; ok {
		if t, err := time.Parse(time.RFC3339, encAt); err == nil {
			result.EncryptedAt = t
		}
	}

	return result, nil
}

func parseFullFileMetadata(content []byte) (*FileMetadata, error) {
	lines := strings.Split(string(content), "\n")
	result := &FileMetadata{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Version:") {
			result.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Vault:") {
			result.Vault = strings.TrimSpace(strings.TrimPrefix(line, "Vault:"))
		} else if strings.HasPrefix(line, "Mode:") {
			result.Mode = strings.TrimSpace(strings.TrimPrefix(line, "Mode:"))
		} else if strings.HasPrefix(line, "Recipients:") {
			recipientsStr := strings.TrimSpace(strings.TrimPrefix(line, "Recipients:"))
			parts := strings.Split(recipientsStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					result.Recipients = append(result.Recipients, p)
				}
			}
		} else if strings.HasPrefix(line, "Encrypted-At:") {
			encAtStr := strings.TrimSpace(strings.TrimPrefix(line, "Encrypted-At:"))
			if t, err := time.Parse(time.RFC3339, encAtStr); err == nil {
				result.EncryptedAt = t
			}
		}
	}

	return result, nil
}
