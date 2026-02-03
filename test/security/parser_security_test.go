package security

import (
	"strings"
	"testing"

	"github.com/cychiuae/shhh/internal/parser"
)

func TestYAMLNoPlaintextLeakage(t *testing.T) {
	p := &parser.YAMLParser{}
	content := []byte(`
database:
  password: supersecret123
  api_key: very-secret-key
`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENCRYPTED" + parser.EncSuffix, nil
	}

	encrypted, err := p.EncryptValues(content, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if strings.Contains(string(encrypted), "supersecret123") {
		t.Error("encrypted output contains plaintext 'supersecret123'")
	}

	if strings.Contains(string(encrypted), "very-secret-key") {
		t.Error("encrypted output contains plaintext 'very-secret-key'")
	}
}

func TestJSONNoPlaintextLeakage(t *testing.T) {
	p := &parser.JSONParser{}
	content := []byte(`{
  "database": {
    "password": "supersecret123",
    "api_key": "very-secret-key"
  }
}`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENCRYPTED" + parser.EncSuffix, nil
	}

	encrypted, err := p.EncryptValues(content, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if strings.Contains(string(encrypted), "supersecret123") {
		t.Error("encrypted output contains plaintext 'supersecret123'")
	}

	if strings.Contains(string(encrypted), "very-secret-key") {
		t.Error("encrypted output contains plaintext 'very-secret-key'")
	}
}

func TestININoPlaintextLeakage(t *testing.T) {
	p := &parser.INIParser{}
	content := []byte(`[database]
password = supersecret123
api_key = very-secret-key
`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENCRYPTED" + parser.EncSuffix, nil
	}

	encrypted, err := p.EncryptValues(content, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if strings.Contains(string(encrypted), "supersecret123") {
		t.Error("encrypted output contains plaintext 'supersecret123'")
	}

	if strings.Contains(string(encrypted), "very-secret-key") {
		t.Error("encrypted output contains plaintext 'very-secret-key'")
	}
}

func TestENVNoPlaintextLeakage(t *testing.T) {
	p := &parser.ENVParser{}
	content := []byte(`DATABASE_PASSWORD=supersecret123
API_KEY=very-secret-key
`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENCRYPTED" + parser.EncSuffix, nil
	}

	encrypted, err := p.EncryptValues(content, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if strings.Contains(string(encrypted), "supersecret123") {
		t.Error("encrypted output contains plaintext 'supersecret123'")
	}

	if strings.Contains(string(encrypted), "very-secret-key") {
		t.Error("encrypted output contains plaintext 'very-secret-key'")
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	p := &parser.YAMLParser{}
	original := []byte(`database:
  host: localhost
  password: mysecret
`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENC:" + plaintext + parser.EncSuffix, nil
	}

	decryptFunc := func(ciphertext string) (string, error) {
		// Extract the plaintext from our test format
		if !strings.HasPrefix(ciphertext, parser.EncPrefix) {
			return ciphertext, nil
		}
		inner := strings.TrimPrefix(ciphertext, parser.EncPrefix)
		inner = strings.TrimSuffix(inner, parser.EncSuffix)
		return strings.TrimPrefix(inner, "ENC:"), nil
	}

	encrypted, err := p.EncryptValues(original, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := p.DecryptValues(encrypted, decryptFunc)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !strings.Contains(string(decrypted), "localhost") {
		t.Error("decrypted output should contain 'localhost'")
	}

	if !strings.Contains(string(decrypted), "mysecret") {
		t.Error("decrypted output should contain 'mysecret'")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	p := &parser.JSONParser{}
	original := []byte(`{"database":{"host":"localhost","password":"mysecret"}}`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENC:" + plaintext + parser.EncSuffix, nil
	}

	decryptFunc := func(ciphertext string) (string, error) {
		if !strings.HasPrefix(ciphertext, parser.EncPrefix) {
			return ciphertext, nil
		}
		inner := strings.TrimPrefix(ciphertext, parser.EncPrefix)
		inner = strings.TrimSuffix(inner, parser.EncSuffix)
		return strings.TrimPrefix(inner, "ENC:"), nil
	}

	encrypted, err := p.EncryptValues(original, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := p.DecryptValues(encrypted, decryptFunc)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !strings.Contains(string(decrypted), "localhost") {
		t.Error("decrypted output should contain 'localhost'")
	}

	if !strings.Contains(string(decrypted), "mysecret") {
		t.Error("decrypted output should contain 'mysecret'")
	}
}

func TestSpecialCharactersPreserved(t *testing.T) {
	p := &parser.YAMLParser{}

	content := []byte(`secret: "!@#$%^&*()_+-=[]{}|;',./<>?"`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENC:" + plaintext + parser.EncSuffix, nil
	}

	decryptFunc := func(ciphertext string) (string, error) {
		if !strings.HasPrefix(ciphertext, parser.EncPrefix) {
			return ciphertext, nil
		}
		inner := strings.TrimPrefix(ciphertext, parser.EncPrefix)
		inner = strings.TrimSuffix(inner, parser.EncSuffix)
		return strings.TrimPrefix(inner, "ENC:"), nil
	}

	encrypted, err := p.EncryptValues(content, encryptFunc)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := p.DecryptValues(encrypted, decryptFunc)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !strings.Contains(string(decrypted), "!@#$%^&*()") {
		t.Error("special characters should be preserved")
	}
}

func TestMalformedYAMLHandled(t *testing.T) {
	p := &parser.YAMLParser{}
	malformed := []byte(`this is not: valid: yaml: at: all:`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENCRYPTED" + parser.EncSuffix, nil
	}

	_, err := p.EncryptValues(malformed, encryptFunc)
	if err == nil {
		t.Error("should error on malformed YAML")
	}
}

func TestMalformedJSONHandled(t *testing.T) {
	p := &parser.JSONParser{}
	malformed := []byte(`{this is not valid json}`)

	encryptFunc := func(plaintext string) (string, error) {
		return parser.EncPrefix + "ENCRYPTED" + parser.EncSuffix, nil
	}

	_, err := p.EncryptValues(malformed, encryptFunc)
	if err == nil {
		t.Error("should error on malformed JSON")
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{parser.EncPrefix + "abc123" + parser.EncSuffix, true},
		{parser.EncPrefix + "YWJjMTIz" + parser.EncSuffix, true},
		{"plaintext", false},
		{"ENC[abc]", false},
		{"ENC[v2:abc]", false},
		{parser.EncPrefix, false},
		{parser.EncSuffix, false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := parser.IsEncrypted(tt.value)
			if got != tt.want {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestFileSizeLimits(t *testing.T) {
	largeContent := make([]byte, parser.MaxFileSize+1)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	err := parser.ValidateContentSize(largeContent)
	if err == nil {
		t.Error("should error on content exceeding max size")
	}

	smallContent := make([]byte, 1024)
	err = parser.ValidateContentSize(smallContent)
	if err != nil {
		t.Errorf("should accept small content: %v", err)
	}
}

func TestFileTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		want     parser.FileFormat
	}{
		{"test.yaml", parser.FormatYAML},
		{"test.yml", parser.FormatYAML},
		{"test.json", parser.FormatJSON},
		{"test.ini", parser.FormatINI},
		{"test.cfg", parser.FormatINI},
		{"test.conf", parser.FormatINI},
		{"test.env", parser.FormatENV},
		{"test.txt", parser.FormatUnknown},
		{"test.md", parser.FormatUnknown},
		{"test", parser.FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := parser.DetectFormat(tt.filename)
			if got != tt.want {
				t.Errorf("DetectFormat(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
