package crypto

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type CLIGPG struct{}

func NewCLIGPG() *CLIGPG {
	return &CLIGPG{}
}

func (g *CLIGPG) LookupKey(email string) (*KeyInfo, error) {
	cmd := exec.Command("gpg", "--list-keys", "--with-colons", "--with-fingerprint", email)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "No public key") ||
				strings.Contains(string(exitErr.Stderr), "not found") {
				return nil, ErrKeyNotFound
			}
		}
		return nil, fmt.Errorf("gpg command failed: %w", err)
	}

	return g.parseKeyOutput(string(output), email)
}

func (g *CLIGPG) parseKeyOutput(output, email string) (*KeyInfo, error) {
	lines := strings.Split(output, "\n")

	var keyID, fingerprint string
	var expiresAt *time.Time
	var createdAt time.Time
	isExpired := false

	for _, line := range lines {
		fields := strings.Split(line, ":")

		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "pub":
			if len(fields) >= 5 {
				keyID = fields[4]
			}
			if len(fields) >= 6 && fields[5] != "" {
				if ts, err := parseTimestamp(fields[5]); err == nil {
					createdAt = ts
				}
			}
			if len(fields) >= 7 && fields[6] != "" {
				if ts, err := parseTimestamp(fields[6]); err == nil {
					expiresAt = &ts
					if ts.Before(time.Now()) {
						isExpired = true
					}
				}
			}
			if len(fields) >= 2 && fields[1] == "e" {
				isExpired = true
			}
		case "fpr":
			if len(fields) >= 10 && fingerprint == "" {
				fingerprint = fields[9]
			}
		}
	}

	if keyID == "" {
		return nil, ErrKeyNotFound
	}

	return &KeyInfo{
		Email:       email,
		KeyID:       keyID,
		Fingerprint: fingerprint,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
		IsExpired:   isExpired,
	}, nil
}

func parseTimestamp(s string) (time.Time, error) {
	if matched, _ := regexp.MatchString(`^\d+$`, s); matched {
		var ts int64
		fmt.Sscanf(s, "%d", &ts)
		return time.Unix(ts, 0), nil
	}
	return time.Parse("2006-01-02", s)
}

func (g *CLIGPG) GetPublicKey(email string) ([]byte, error) {
	cmd := exec.Command("gpg", "--export", "--armor", email)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to export public key: %w", err)
	}

	if len(output) == 0 {
		return nil, ErrKeyNotFound
	}

	return output, nil
}

func (g *CLIGPG) Encrypt(data []byte, recipients []string) ([]byte, error) {
	args := []string{"--encrypt", "--armor", "--trust-model", "always"}
	for _, r := range recipients {
		args = append(args, "--recipient", r)
	}

	cmd := exec.Command("gpg", args...)
	cmd.Stdin = bytes.NewReader(data)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg encrypt failed: %s", stderr.String())
	}

	return stdout.Bytes(), nil
}

func (g *CLIGPG) Decrypt(data []byte) ([]byte, error) {
	cmd := exec.Command("gpg", "--decrypt", "--quiet", "--batch")
	cmd.Stdin = bytes.NewReader(data)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := stderr.String()
		if strings.Contains(errStr, "No secret key") {
			return nil, ErrNoPrivateKey
		}
		return nil, fmt.Errorf("gpg decrypt failed: %s", errStr)
	}

	return stdout.Bytes(), nil
}

func (g *CLIGPG) ImportPublicKey(armoredKey []byte) (*KeyInfo, error) {
	cmd := exec.Command("gpg", "--import")
	cmd.Stdin = bytes.NewReader(armoredKey)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg import failed: %s", stderr.String())
	}

	emailRegex := regexp.MustCompile(`<([^>]+)>`)
	matches := emailRegex.FindStringSubmatch(stderr.String())
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract email from import output")
	}

	email := matches[1]
	return g.LookupKey(email)
}
