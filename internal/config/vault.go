package config

import (
	"bytes"
	"os"
	"time"

	"github.com/cychiuae/shhh/internal/store"
	"gopkg.in/yaml.v3"
)

type User struct {
	Email       string     `yaml:"email"`
	KeyID       string     `yaml:"key_id"`
	Fingerprint string     `yaml:"fingerprint"`
	ExpiresAt   *time.Time `yaml:"expires_at,omitempty"`
	AddedAt     time.Time  `yaml:"added_at"`
}

type RegisteredFile struct {
	Path         string    `yaml:"path"`
	Mode         string    `yaml:"mode"`
	GPGCopy      *bool     `yaml:"gpg_copy,omitempty"`
	Recipients   []string  `yaml:"recipients,omitempty"`
	RegisteredAt time.Time `yaml:"registered_at"`
}

type Vault struct {
	Users []User           `yaml:"users"`
	Files []RegisteredFile `yaml:"files"`
}

func NewVault() *Vault {
	return &Vault{
		Users: []User{},
		Files: []RegisteredFile{},
	}
}

func LoadVault(s *store.Store, vaultName string) (*Vault, error) {
	data, err := os.ReadFile(s.VaultConfigPath(vaultName))
	if err != nil {
		if os.IsNotExist(err) {
			return NewVault(), nil
		}
		return nil, err
	}

	var v Vault
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	if v.Users == nil {
		v.Users = []User{}
	}
	if v.Files == nil {
		v.Files = []RegisteredFile{}
	}

	return &v, nil
}

func (v *Vault) Save(s *store.Store, vaultName string) error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(v); err != nil {
		return err
	}
	encoder.Close()
	return store.WriteFile(s.VaultConfigPath(vaultName), buf.Bytes())
}

// User methods

func (v *Vault) AddUser(user User) {
	for i, u := range v.Users {
		if u.Email == user.Email {
			v.Users[i] = user
			return
		}
	}
	v.Users = append(v.Users, user)
}

func (v *Vault) RemoveUser(email string) bool {
	for i, u := range v.Users {
		if u.Email == email {
			v.Users = append(v.Users[:i], v.Users[i+1:]...)
			return true
		}
	}
	return false
}

func (v *Vault) GetUser(email string) *User {
	for i := range v.Users {
		if v.Users[i].Email == email {
			return &v.Users[i]
		}
	}
	return nil
}

func (v *Vault) HasUser(email string) bool {
	return v.GetUser(email) != nil
}

func (v *Vault) Emails() []string {
	emails := make([]string, len(v.Users))
	for i, u := range v.Users {
		emails[i] = u.Email
	}
	return emails
}

// File methods

func (v *Vault) RegisterFile(file RegisteredFile) {
	for i, f := range v.Files {
		if f.Path == file.Path {
			v.Files[i] = file
			return
		}
	}
	v.Files = append(v.Files, file)
}

func (v *Vault) UnregisterFile(path string) bool {
	for i, f := range v.Files {
		if f.Path == path {
			v.Files = append(v.Files[:i], v.Files[i+1:]...)
			return true
		}
	}
	return false
}

func (v *Vault) GetFile(path string) *RegisteredFile {
	for i := range v.Files {
		if v.Files[i].Path == path {
			return &v.Files[i]
		}
	}
	return nil
}

func (v *Vault) HasFile(path string) bool {
	return v.GetFile(path) != nil
}

func (v *Vault) UpdateFile(path string, fn func(*RegisteredFile)) bool {
	for i := range v.Files {
		if v.Files[i].Path == path {
			fn(&v.Files[i])
			return true
		}
	}
	return false
}
