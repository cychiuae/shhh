package config

import (
	"encoding/json"
	"os"

	"github.com/cychiuae/shhh/internal/store"
)

const CurrentVersion = "1"

type Config struct {
	Version      string `json:"version"`
	GPGCopy      bool   `json:"gpg_copy"`
	DefaultVault string `json:"default_vault"`
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
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save(s *store.Store) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return store.WriteFile(s.ConfigPath(), data)
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
