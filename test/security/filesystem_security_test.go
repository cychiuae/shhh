package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cychiuae/shhh/internal/config"
	"github.com/cychiuae/shhh/internal/gitignore"
	"github.com/cychiuae/shhh/internal/store"
)

func TestStoreDirectoryPermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	s := store.New(tmpDir)
	if err := s.Initialize(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	info, err := os.Stat(s.ShhhPath())
	if err != nil {
		t.Fatalf("failed to stat .shhh: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf(".shhh directory permissions = %o, want 0700", perm)
	}
}

func TestPathTraversalPrevention(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"../../../etc/passwd", true},
		{"../..", true},
		{"./valid/path", false},
		{"valid/path", false},
		{"secrets.yaml", false},
		{"/absolute/path", true},
		{".shhh/config.json", true},
		{"..\\..\\windows\\path", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := config.ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestGitignoreEnsureIgnored(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-gitignore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretFile := "secrets.yaml"
	if err := gitignore.EnsureIgnored(tmpDir, secretFile); err != nil {
		t.Fatalf("failed to ensure ignored: %v", err)
	}

	if !gitignore.IsIgnored(tmpDir, secretFile) {
		t.Error("secrets.yaml should be in .gitignore")
	}

	if err := gitignore.EnsureIgnored(tmpDir, secretFile); err != nil {
		t.Fatalf("failed to ensure ignored second time: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	count := 0
	for _, line := range filepath.SplitList(string(content)) {
		if line == "/"+secretFile {
			count++
		}
	}
}

func TestGitignoreHandlesExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-gitignore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	existingContent := `# Existing gitignore
*.log
node_modules/
.env
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing .gitignore: %v", err)
	}

	if err := gitignore.EnsureIgnored(tmpDir, "secrets.yaml"); err != nil {
		t.Fatalf("failed to ensure ignored: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	contentStr := string(content)
	if !filepath.IsAbs(contentStr) && !gitignore.IsIgnored(tmpDir, "secrets.yaml") {
		t.Error("secrets.yaml should be in .gitignore")
	}
}

func TestVaultNameValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-vault-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	s := store.New(tmpDir)
	if err := s.Initialize(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"valid-vault", false},
		{"vault_123", false},
		{"", true},
		{".", true},
		{"..", true},
		{"vault/subdir", true},
		{"vault\\subdir", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreateVault(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVault(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
			if err == nil {
				s.RemoveVault(tt.name)
			}
		})
	}
}

func TestCannotRemoveDefaultVault(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-vault-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	s := store.New(tmpDir)
	if err := s.Initialize(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	err = s.RemoveVault(store.DefaultVault)
	if err == nil {
		t.Error("should not be able to remove default vault")
	}
}

func TestWriteFilePermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-file-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "subdir", "test.json")
	if err := store.WriteFile(testFile, []byte("test content")); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"valid@example.com", false},
		{"user.name@domain.org", false},
		{"user+tag@domain.com", false},
		{"invalid", true},
		{"@domain.com", true},
		{"user@", true},
		{"", true},
		{"user@domain", true},
		{"user@.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := config.ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestFindRootWalksUp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-root-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	s := store.New(tmpDir)
	if err := s.Initialize(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	subDir := filepath.Join(tmpDir, "sub1", "sub2", "sub3")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	foundRoot, err := store.FindRoot()
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}

	if foundRoot != tmpDir {
		t.Errorf("FindRoot() = %q, want %q", foundRoot, tmpDir)
	}
}

func TestFindRootErrorsWhenNotInitialized(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shhh-root-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	_, err = store.FindRoot()
	if err == nil {
		t.Error("FindRoot should error when not initialized")
	}
}
