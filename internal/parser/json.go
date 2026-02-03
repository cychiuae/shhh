package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type JSONParser struct{}

func (p *JSONParser) FileType() string {
	return "json"
}

func (p *JSONParser) EncryptValues(content []byte, encrypt EncryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	encrypted, err := p.processValue(data, encrypt, true, 0)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(encrypted); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *JSONParser) DecryptValues(content []byte, decrypt DecryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	decrypted, err := p.processValue(data, decrypt, false, 0)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(decrypted); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *JSONParser) processValue(value interface{}, transform func(string) (string, error), encrypting bool, depth int) (interface{}, error) {
	if depth > MaxNestingDepth {
		return nil, fmt.Errorf("maximum nesting depth exceeded")
	}

	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if key == "_shhh" {
				result[key] = val
				continue
			}
			processed, err := p.processValue(val, transform, encrypting, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = processed
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			processed, err := p.processValue(val, transform, encrypting, depth+1)
			if err != nil {
				return nil, err
			}
			result[i] = processed
		}
		return result, nil

	case string:
		if encrypting {
			if !IsEncrypted(v) && v != "" {
				encrypted, err := transform(v)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt value: %w", err)
				}
				return encrypted, nil
			}
		} else {
			if IsEncrypted(v) {
				decrypted, err := transform(v)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt value: %w", err)
				}
				return decrypted, nil
			}
		}
		return v, nil

	default:
		return v, nil
	}
}

func AddJSONMetadata(content []byte, metadata map[string]interface{}) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	data["_shhh"] = metadata

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GetJSONMetadata(content []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	shhh, ok := data["_shhh"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	return shhh, nil
}

func RemoveJSONMetadata(content []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	delete(data, "_shhh")

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
