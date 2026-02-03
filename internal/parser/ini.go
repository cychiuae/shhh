package parser

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

type INIParser struct{}

func (p *INIParser) FileType() string {
	return "ini"
}

func (p *INIParser) EncryptValues(content []byte, encrypt EncryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	cfg, err := ini.Load(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI: %w", err)
	}

	for _, section := range cfg.Sections() {
		if section.Name() == "_shhh" {
			continue
		}

		for _, key := range section.Keys() {
			value := key.String()
			if !IsEncrypted(value) && value != "" {
				encrypted, err := encrypt(value)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt value for %s.%s: %w", section.Name(), key.Name(), err)
				}
				key.SetValue(encrypted)
			}
		}
	}

	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to encode INI: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *INIParser) DecryptValues(content []byte, decrypt DecryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	cfg, err := ini.Load(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI: %w", err)
	}

	for _, section := range cfg.Sections() {
		if section.Name() == "_shhh" {
			continue
		}

		for _, key := range section.Keys() {
			value := key.String()
			if IsEncrypted(value) {
				decrypted, err := decrypt(value)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt value for %s.%s: %w", section.Name(), key.Name(), err)
				}
				key.SetValue(decrypted)
			}
		}
	}

	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to encode INI: %w", err)
	}

	return buf.Bytes(), nil
}

func AddINIMetadata(content []byte, metadata map[string]interface{}) ([]byte, error) {
	cfg, err := ini.Load(content)
	if err != nil {
		return nil, err
	}

	section, err := cfg.NewSection("_shhh")
	if err != nil {
		return nil, err
	}

	for k, v := range metadata {
		section.Key(k).SetValue(fmt.Sprintf("%v", v))
	}

	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GetINIMetadata(content []byte) (map[string]string, error) {
	cfg, err := ini.Load(content)
	if err != nil {
		return nil, err
	}

	section := cfg.Section("_shhh")
	if section == nil {
		return nil, nil
	}

	result := make(map[string]string)
	for _, key := range section.Keys() {
		result[key.Name()] = key.String()
	}

	return result, nil
}

func RemoveINIMetadata(content []byte) ([]byte, error) {
	cfg, err := ini.Load(content)
	if err != nil {
		return nil, err
	}

	cfg.DeleteSection("_shhh")

	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ParseINISection(content []byte, sectionName string) (map[string]string, error) {
	cfg, err := ini.Load(content)
	if err != nil {
		return nil, err
	}

	section := cfg.Section(sectionName)
	result := make(map[string]string)
	for _, key := range section.Keys() {
		result[key.Name()] = key.String()
	}

	return result, nil
}

func EscapeINIValue(value string) string {
	if strings.ContainsAny(value, "=;#\n\r") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}
