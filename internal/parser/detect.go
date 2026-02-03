package parser

import (
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

func DetectFormat(filename string) FileFormat {
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
	default:
		return FormatUnknown
	}
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

func GetParserForFile(filename string) Parser {
	format := DetectFormat(filename)
	return GetParser(format)
}
