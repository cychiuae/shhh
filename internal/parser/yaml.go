package parser

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type YAMLParser struct{}

func (p *YAMLParser) FileType() string {
	return "yaml"
}

func (p *YAMLParser) EncryptValues(content []byte, encrypt EncryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := p.processNode(&root, encrypt, true, 0); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}
	encoder.Close()

	return buf.Bytes(), nil
}

func (p *YAMLParser) DecryptValues(content []byte, decrypt DecryptFunc) ([]byte, error) {
	if err := ValidateContentSize(content); err != nil {
		return nil, err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := p.processNode(&root, decrypt, false, 0); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}
	encoder.Close()

	return buf.Bytes(), nil
}

func (p *YAMLParser) processNode(node *yaml.Node, transform func(string) (string, error), encrypting bool, depth int) error {
	if depth > MaxNestingDepth {
		return fmt.Errorf("maximum nesting depth exceeded")
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if err := p.processNode(child, transform, encrypting, depth+1); err != nil {
				return err
			}
		}

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.Value == "_shhh" {
				continue
			}

			if err := p.processNode(valueNode, transform, encrypting, depth+1); err != nil {
				return err
			}
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := p.processNode(child, transform, encrypting, depth+1); err != nil {
				return err
			}
		}

	case yaml.ScalarNode:
		if encrypting {
			if !IsEncrypted(node.Value) && node.Value != "" {
				encrypted, err := transform(node.Value)
				if err != nil {
					return fmt.Errorf("failed to encrypt value: %w", err)
				}
				node.Value = encrypted
				node.Tag = "!!str"
				node.Style = yaml.LiteralStyle
			}
		} else {
			if IsEncrypted(node.Value) {
				decrypted, err := transform(node.Value)
				if err != nil {
					return fmt.Errorf("failed to decrypt value: %w", err)
				}
				node.Value = decrypted
				node.Style = inferStyle(decrypted)
			}
		}

	case yaml.AliasNode:
		if node.Alias != nil {
			if err := p.processNode(node.Alias, transform, encrypting, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func inferStyle(value string) yaml.Style {
	if strings.Contains(value, "\n") {
		return yaml.LiteralStyle
	}
	return 0
}

func AddShhhMetadata(content []byte, metadata map[string]interface{}) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, err
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return content, nil
	}

	docNode := root.Content[0]
	if docNode.Kind != yaml.MappingNode {
		return content, nil
	}

	metaNode := &yaml.Node{Kind: yaml.MappingNode}
	for k, v := range metadata {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", v)}
		metaNode.Content = append(metaNode.Content, keyNode, valueNode)
	}

	shhhKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "_shhh"}
	docNode.Content = append(docNode.Content, shhhKey, metaNode)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return nil, err
	}
	encoder.Close()

	return buf.Bytes(), nil
}

func GetShhhMetadata(content []byte) (map[string]string, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	shhh, ok := data["_shhh"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	result := make(map[string]string)
	for k, v := range shhh {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result, nil
}

func RemoveShhhMetadata(content []byte) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, err
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return content, nil
	}

	docNode := root.Content[0]
	if docNode.Kind != yaml.MappingNode {
		return content, nil
	}

	var newContent []*yaml.Node
	for i := 0; i < len(docNode.Content); i += 2 {
		if docNode.Content[i].Value != "_shhh" {
			newContent = append(newContent, docNode.Content[i], docNode.Content[i+1])
		}
	}
	docNode.Content = newContent

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return nil, err
	}
	encoder.Close()

	return buf.Bytes(), nil
}
