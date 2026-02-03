package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type ENVParser struct{}

func (p *ENVParser) FileType() string {
	return "env"
}

func (p *ENVParser) EncryptValues(content []byte, encrypt EncryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		processed, err := p.processLine(line, encrypt, true)
		if err != nil {
			return nil, err
		}
		buf.WriteString(processed)
		buf.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *ENVParser) DecryptValues(content []byte, decrypt DecryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		processed, err := p.processLine(line, decrypt, false)
		if err != nil {
			return nil, err
		}
		buf.WriteString(processed)
		buf.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *ENVParser) processLine(line string, transform func(string) (string, error), encrypting bool) (string, error) {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return line, nil
	}

	if strings.HasPrefix(trimmed, "_SHHH_") {
		return line, nil
	}

	eqIndex := strings.Index(line, "=")
	if eqIndex == -1 {
		return line, nil
	}

	key := line[:eqIndex]
	value := line[eqIndex+1:]

	unquotedValue, wasQuoted, quoteChar := unquoteValue(value)

	if encrypting {
		if !IsEncrypted(unquotedValue) && unquotedValue != "" {
			encrypted, err := transform(unquotedValue)
			if err != nil {
				return "", fmt.Errorf("failed to encrypt value for %s: %w", strings.TrimSpace(key), err)
			}
			return key + "=" + quoteValue(encrypted, wasQuoted, quoteChar), nil
		}
	} else {
		if IsEncrypted(unquotedValue) {
			decrypted, err := transform(unquotedValue)
			if err != nil {
				return "", fmt.Errorf("failed to decrypt value for %s: %w", strings.TrimSpace(key), err)
			}
			return key + "=" + quoteValue(decrypted, needsQuoting(decrypted), '"'), nil
		}
	}

	return line, nil
}

func unquoteValue(value string) (string, bool, byte) {
	value = strings.TrimSpace(value)

	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1], true, value[0]
		}
	}

	return value, false, 0
}

func quoteValue(value string, quote bool, quoteChar byte) string {
	if !quote && !needsQuoting(value) {
		return value
	}

	if quoteChar == 0 {
		quoteChar = '"'
	}

	q := string(quoteChar)
	return q + value + q
}

func needsQuoting(value string) bool {
	if value == "" {
		return true
	}

	for _, c := range value {
		if c == ' ' || c == '\t' || c == '"' || c == '\'' ||
			c == '#' || c == '$' || c == '\\' || c == '\n' {
			return true
		}
	}

	return false
}

func AddENVMetadata(content []byte, metadata map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(content)
	buf.WriteString("\n# shhh metadata\n")

	for k, v := range metadata {
		buf.WriteString(fmt.Sprintf("_SHHH_%s=%v\n", strings.ToUpper(k), v))
	}

	return buf.Bytes(), nil
}

func GetENVMetadata(content []byte) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "_SHHH_") {
			continue
		}

		eqIndex := strings.Index(line, "=")
		if eqIndex == -1 {
			continue
		}

		key := strings.TrimPrefix(line[:eqIndex], "_SHHH_")
		value := line[eqIndex+1:]
		result[strings.ToLower(key)] = value
	}

	return result, scanner.Err()
}

func RemoveENVMetadata(content []byte) ([]byte, error) {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inMetadata := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "# shhh metadata" {
			inMetadata = true
			continue
		}

		if inMetadata && strings.HasPrefix(strings.TrimSpace(line), "_SHHH_") {
			continue
		}

		if inMetadata && strings.TrimSpace(line) == "" {
			continue
		}

		inMetadata = false
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	var buf bytes.Buffer
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}
