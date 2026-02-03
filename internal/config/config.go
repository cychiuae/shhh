package config

import (
	"bytes"
	"os"

	"github.com/cychiuae/shhh/internal/store"
	"gopkg.in/yaml.v3"
)

const CurrentVersion = "1"

type Config struct {
	Version      string `yaml:"version"`
	GPGCopy      bool   `yaml:"gpg_copy"`
	DefaultVault string `yaml:"default_vault"`
}

func NewConfig() *Config {
	return &Config{
		Version:      CurrentVersion,
		GPGCopy:      false,
		DefaultVault: store.DefaultVault,
	}
}

func Load(s *store.Store) (*Config, error) {
	data, err := os.ReadFile(s.ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return NewConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save(s *store.Store) error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		return err
	}
	encoder.Close()
	return store.WriteFile(s.ConfigPath(), buf.Bytes())
}

func (c *Config) Get(key string) (string, bool) {
	switch key {
	case "version":
		return c.Version, true
	case "gpg_copy":
		if c.GPGCopy {
			return "true", true
		}
		return "false", true
	case "default_vault":
		return c.DefaultVault, true
	default:
		return "", false
	}
}

func (c *Config) Set(key, value string) bool {
	switch key {
	case "gpg_copy":
		c.GPGCopy = value == "true" || value == "1" || value == "yes"
		return true
	case "default_vault":
		c.DefaultVault = value
		return true
	default:
		return false
	}
}

func (c *Config) List() map[string]string {
	gpgCopy := "false"
	if c.GPGCopy {
		gpgCopy = "true"
	}
	return map[string]string{
		"version":       c.Version,
		"gpg_copy":      gpgCopy,
		"default_vault": c.DefaultVault,
	}
}
