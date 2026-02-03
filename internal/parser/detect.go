package parser

import (
	"bytes"
	"path/filepath"
	"strings"
)

type FileFormat string

const (
	FormatYAML    FileFormat = "yaml"
	FormatJSON    FileFormat = "json"
	FormatINI     FileFormat = "ini"
	FormatENV     FileFormat = "env"
	FormatUnknown FileFormat = "unknown"
)

func DetectFormat(filename string, content []byte) FileFormat {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	case ".json":
		return FormatJSON
	case ".ini", ".cfg", ".conf":
		return FormatINI
	case ".env":
		return FormatENV
	}

	return detectByContent(content)
}

func detectByContent(content []byte) FileFormat {
	content = bytes.TrimSpace(content)

	if len(content) == 0 {
		return FormatUnknown
	}

	if bytes.HasPrefix(content, []byte("---")) {
		return FormatYAML
	}

	// Check for INI sections first (before JSON check)
	// INI files start with [section] but are not valid JSON
	if content[0] == '[' {
		lines := bytes.Split(content, []byte("\n"))
		firstLine := bytes.TrimSpace(lines[0])
		// If first line is [word] without quotes/commas, likely INI
		if bytes.HasSuffix(firstLine, []byte("]")) && !bytes.Contains(firstLine, []byte(",")) && !bytes.Contains(firstLine, []byte("\"")) {
			return FormatINI
		}
	}

	if content[0] == '{' || content[0] == '[' {
		return FormatJSON
	}

	lines := bytes.Split(content, []byte("\n"))
	hasYAMLStructure := false
	hasINISection := false
	hasENVFormat := true

	for _, line := range lines {
		line = bytes.TrimSpace(line)

		if len(line) == 0 || bytes.HasPrefix(line, []byte("#")) {
			continue
		}

		if bytes.HasPrefix(line, []byte("[")) && bytes.HasSuffix(line, []byte("]")) {
			hasINISection = true
		}

		if bytes.Contains(line, []byte(": ")) || bytes.HasSuffix(line, []byte(":")) {
			hasYAMLStructure = true
		}

		if !bytes.Contains(line, []byte("=")) {
			hasENVFormat = false
		}
	}

	if hasINISection {
		return FormatINI
	}

	if hasENVFormat && !hasYAMLStructure {
		return FormatENV
	}

	if hasYAMLStructure {
		return FormatYAML
	}

	return FormatUnknown
}

func GetParser(format FileFormat) Parser {
	switch format {
	case FormatYAML:
		return &YAMLParser{}
	case FormatJSON:
		return &JSONParser{}
	case FormatINI:
		return &INIParser{}
	case FormatENV:
		return &ENVParser{}
	default:
		return nil
	}
}

func GetParserForFile(filename string, content []byte) Parser {
	format := DetectFormat(filename, content)
	return GetParser(format)
}
