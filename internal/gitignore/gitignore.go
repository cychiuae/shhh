package gitignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EnsureIgnored(rootDir, filePath string) error {
	gitignorePath := filepath.Join(rootDir, ".gitignore")

	lines, err := readGitignore(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	relativePath := filePath
	if filepath.IsAbs(filePath) {
		rel, err := filepath.Rel(rootDir, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		relativePath = rel
	}

	pattern := "/" + relativePath

	if isIgnored(lines, pattern) {
		return nil
	}

	lines = append(lines, pattern)

	if err := writeGitignore(gitignorePath, lines); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

func IsIgnored(rootDir, filePath string) bool {
	gitignorePath := filepath.Join(rootDir, ".gitignore")

	lines, err := readGitignore(gitignorePath)
	if err != nil {
		return false
	}

	relativePath := filePath
	if filepath.IsAbs(filePath) {
		rel, err := filepath.Rel(rootDir, filePath)
		if err != nil {
			return false
		}
		relativePath = rel
	}

	pattern := "/" + relativePath

	return isIgnored(lines, pattern)
}

func readGitignore(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func writeGitignore(path string, lines []string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for i, line := range lines {
		if i > 0 {
			file.WriteString("\n")
		}
		file.WriteString(line)
	}
	file.WriteString("\n")

	return nil
}

func isIgnored(lines []string, pattern string) bool {
	pattern = strings.TrimSpace(pattern)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == pattern {
			return true
		}

		if strings.HasPrefix(line, "/") && strings.HasPrefix(pattern, "/") {
			if line == pattern {
				return true
			}
		} else if !strings.HasPrefix(line, "/") {
			filename := filepath.Base(strings.TrimPrefix(pattern, "/"))
			if line == filename {
				return true
			}
		}
	}

	return false
}

func WarnIfNotIgnored(rootDir, filePath string) string {
	if !IsIgnored(rootDir, filePath) {
		return fmt.Sprintf("Warning: %s is not in .gitignore", filePath)
	}
	return ""
}

func CheckGitignoreExists(rootDir string) bool {
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	_, err := os.Stat(gitignorePath)
	return err == nil
}
