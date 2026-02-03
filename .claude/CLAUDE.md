# CLAUDE.md

## Project Overview

**shhh** is a GitOps-friendly CLI tool for managing encrypted secrets in Git repositories. It encrypts sensitive values within configuration files (YAML, JSON, INI, ENV) using GPG, preserving file structure while keeping secrets secure.

## Key Features

- **Value-level encryption**: Encrypts only secret values, preserving file structure and keys
- **Full-file encryption**: Option to encrypt entire files when needed
- **Multi-recipient GPG encryption**: Encrypt secrets for multiple users
- **Vault-based organization**: Group secrets and users into logical vaults
- **Per-file recipient overrides**: Restrict files to specific users
- **Multiple format support**: YAML, JSON, INI, ENV (detected by file extension)

## Directory Structure

```
├── main.go                 # Entry point
├── cmd/                    # CLI commands (Cobra-based)
│   ├── root.go             # Root command and version info
│   ├── init.go             # Initialize shhh in project
│   ├── config.go           # Configuration management
│   ├── vault.go            # Vault operations (create, remove, list)
│   ├── user.go             # User management (add, remove, list, check)
│   ├── register.go         # Register files for encryption
│   ├── file.go             # Per-file settings (recipients, mode)
│   ├── encrypt.go          # Encrypt files
│   ├── decrypt.go          # Decrypt files
│   ├── edit.go             # Edit encrypted files in $EDITOR
│   ├── reencrypt.go        # Re-encrypt with updated recipients
│   ├── status.go           # Show encryption status
│   └── list.go             # List registered files
├── internal/               # Core business logic
│   ├── config/             # Configuration handling
│   │   ├── config.go       # Main config struct and operations
│   │   ├── user.go         # User data structures
│   │   ├── file.go         # File registration data
│   │   └── vault.go        # Vault data structures
│   ├── crypto/             # Encryption/Decryption
│   │   ├── encrypt.go      # Encryption logic
│   │   ├── gpg.go          # GPG provider interface
│   │   ├── gpg_native.go   # Native go-crypto implementation
│   │   └── gpg_cli.go      # GPG CLI fallback
│   ├── parser/             # File format handling
│   │   ├── parser.go       # Parser interface
│   │   ├── yaml.go         # YAML parser
│   │   ├── json.go         # JSON parser
│   │   ├── ini.go          # INI parser
│   │   ├── env.go          # ENV parser
│   │   └── detect.go       # Format detection by extension
│   ├── store/              # File system management
│   │   └── store.go        # Store paths, initialization, file I/O
│   └── gitignore/          # Git ignore management
│       └── gitignore.go
├── test/                   # Test suite
│   ├── security/           # Security-focused tests
│   ├── integration/        # Integration tests
│   └── fixtures/           # Test data
├── go.mod                  # Go module definition
└── Makefile                # Build automation
```

## Storage Directory (.shhh/)

```
.shhh/
├── config.yaml             # Project-wide configuration (version, gpg_copy, default_vault)
├── vaults/
│   └── <vault-name>/
│       └── vault.yaml      # Combined users and files for this vault
└── pubkeys/
    └── <email>.asc         # Cached public keys
```

## Tech Stack

- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **Encryption**: ProtonMail go-crypto (GPG)
- **Parsers**: gopkg.in/yaml.v3, gopkg.in/ini.v1

## Common Commands

```bash
# Build
make build              # Build binary
make test               # Run all tests
make test-security      # Security tests
make lint               # Run linters

# Usage
shhh init               # Initialize in project
shhh user add <email>   # Add user
shhh register <file>    # Register file for encryption
shhh encrypt <file>     # Encrypt file
shhh decrypt <file>     # Decrypt file
shhh edit <file>        # Edit encrypted file
shhh status             # Show encryption status

# File recipient management
shhh file set-recipients <file> <email>...     # Set specific recipients
shhh file add-recipients <file> <email>...     # Add recipients
shhh file remove-recipients <file> <email>...  # Remove recipients
shhh file clear-recipients <file>              # Clear (use all vault users)
shhh file show <file>                          # Show file settings
```

## Encryption Format

**Values mode** (default): Preserves structure, encrypts values
```yaml
password: ENC[v1:BASE64_GPG_DATA]
_shhh:
  version: "1"
  vault: default
  mode: values
  recipients: [alice@example.com]
```

**Full mode**: Encrypts entire file content with metadata header

## Architecture Notes

- **Store**: Manages `.shhh/` directory and file paths
- **Crypto**: GPG encryption with native go-crypto or CLI fallback
- **Parser**: Detects format by file extension (.yaml, .yml, .json, .ini, .cfg, .conf, .env) and processes config files; unsupported extensions fall back to full-file encryption
- **Config**: Manages vaults, users, and file registrations
- **Security**: Strict file permissions (0600), automatic .gitignore management
