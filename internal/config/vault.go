package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/cychiuae/shhh/internal/store"
)

type User struct {
	Email       string     `json:"email"`
	KeyID       string     `json:"key_id"`
	Fingerprint string     `json:"fingerprint"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	AddedAt     time.Time  `json:"added_at"`
}

type VaultUsers struct {
	Users []User `json:"users"`
}

func NewVaultUsers() *VaultUsers {
	return &VaultUsers{
		Users: []User{},
	}
}

func LoadVaultUsers(s *store.Store, vault string) (*VaultUsers, error) {
	path := s.VaultUsersPath(vault)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewVaultUsers(), nil
		}
		return nil, err
	}

	var users VaultUsers
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}

	return &users, nil
}

func (v *VaultUsers) Save(s *store.Store, vault string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteFile(s.VaultUsersPath(vault), data)
}

func (v *VaultUsers) Add(user User) {
	for i, u := range v.Users {
		if u.Email == user.Email {
			v.Users[i] = user
			return
		}
	}
	v.Users = append(v.Users, user)
}

func (v *VaultUsers) Remove(email string) bool {
	for i, u := range v.Users {
		if u.Email == email {
			v.Users = append(v.Users[:i], v.Users[i+1:]...)
			return true
		}
	}
	return false
}

func (v *VaultUsers) Get(email string) *User {
	for i := range v.Users {
		if v.Users[i].Email == email {
			return &v.Users[i]
		}
	}
	return nil
}

func (v *VaultUsers) HasUser(email string) bool {
	return v.Get(email) != nil
}

func (v *VaultUsers) Emails() []string {
	emails := make([]string, len(v.Users))
	for i, u := range v.Users {
		emails[i] = u.Email
	}
	return emails
}

type RegisteredFile struct {
	Path         string    `json:"path"`
	Mode         string    `json:"mode"`
	GPGCopy      bool      `json:"gpg_copy"`
	Recipients   []string  `json:"recipients"`
	RegisteredAt time.Time `json:"registered_at"`
}

type VaultFiles struct {
	Files []RegisteredFile `json:"files"`
}

func NewVaultFiles() *VaultFiles {
	return &VaultFiles{
		Files: []RegisteredFile{},
	}
}

func LoadVaultFiles(s *store.Store, vault string) (*VaultFiles, error) {
	path := s.VaultFilesPath(vault)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewVaultFiles(), nil
		}
		return nil, err
	}

	var files VaultFiles
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}

	return &files, nil
}

func (v *VaultFiles) Save(s *store.Store, vault string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteFile(s.VaultFilesPath(vault), data)
}

func (v *VaultFiles) Register(file RegisteredFile) {
	for i, f := range v.Files {
		if f.Path == file.Path {
			v.Files[i] = file
			return
		}
	}
	v.Files = append(v.Files, file)
}

func (v *VaultFiles) Unregister(path string) bool {
	for i, f := range v.Files {
		if f.Path == path {
			v.Files = append(v.Files[:i], v.Files[i+1:]...)
			return true
		}
	}
	return false
}

func (v *VaultFiles) Get(path string) *RegisteredFile {
	for i := range v.Files {
		if v.Files[i].Path == path {
			return &v.Files[i]
		}
	}
	return nil
}

func (v *VaultFiles) HasFile(path string) bool {
	return v.Get(path) != nil
}

func (v *VaultFiles) Update(path string, fn func(*RegisteredFile)) bool {
	for i := range v.Files {
		if v.Files[i].Path == path {
			fn(&v.Files[i])
			return true
		}
	}
	return false
}
