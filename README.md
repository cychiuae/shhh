# shhh - Secret Management Tool

A GitOps-friendly CLI tool for managing secrets in Git repositories.

## Features

- **Value-level encryption**: Encrypts only values within YAML/JSON/INI/ENV files, preserving structure
- **Full-file encryption**: Encrypts entire files when needed (or files with unsupported extensions)
- **Multi-recipient**: Encrypt secrets for multiple GPG users
- **Vault-based organization**: Group secrets and users into vaults
- **Per-file access control**: Override vault-wide recipients for specific files
- **GPG integration**: Uses GPG keys for encryption (go-crypto library with gpg CLI fallback)

## Supported File Formats

Format detection is based on file extension only:

| Extension | Format |
|-----------|--------|
| `.yaml`, `.yml` | YAML |
| `.json` | JSON |
| `.ini`, `.cfg`, `.conf` | INI |
| `.env` | ENV |

Files with other extensions are encrypted using full-file mode.

## Installation

```bash
go install github.com/cychiuae/shhh@latest
```

Or build from source:

```bash
git clone https://github.com/cychiuae/shhh.git
cd shhh
go build -o shhh .
```

## Quick Start

```bash
# Initialize shhh in your project
shhh init

# Add users who can decrypt secrets
shhh user add alice@example.com
shhh user add bob@example.com

# Register a file for encryption
echo "password: supersecret123" > secrets.yaml
shhh register secrets.yaml

# Encrypt the file
shhh encrypt secrets.yaml
# Creates secrets.yaml.enc

# Decrypt when needed
shhh decrypt secrets.yaml

# Edit encrypted files directly
shhh edit secrets.yaml
```

## Commands

### Initialization
- `shhh init` - Initialize shhh in the current directory

### Configuration
- `shhh config get <key>` - Get a config value
- `shhh config set <key> <value>` - Set a config value
- `shhh config list` - List all config values

### Vault Management
- `shhh vault create <name>` - Create a new vault
- `shhh vault remove <name>` - Remove a vault
- `shhh vault list` - List all vaults

### User Management
- `shhh user add <email>` - Add a user to a vault
- `shhh user remove <email>` - Remove a user from a vault
- `shhh user list` - List users in a vault
- `shhh user check` - Verify all user keys are valid

### File Registration
- `shhh register <file>` - Register a file for encryption
- `shhh unregister <file>` - Unregister a file
- `shhh list` - List registered files

### File Settings
- `shhh file set-recipients <file> <email>...` - Set specific recipients
- `shhh file clear-recipients <file>` - Clear per-file recipients
- `shhh file set-mode <file> <values|full>` - Set encryption mode
- `shhh file set-gpg-copy <file> <true|false>` - Enable/disable GPG backup
- `shhh file show <file>` - Show file settings

### Encryption
- `shhh encrypt [file]` - Encrypt a file
- `shhh encrypt --vault <name>` - Encrypt all files in a vault
- `shhh encrypt --all` - Encrypt all registered files
- `shhh decrypt [file]` - Decrypt a file
- `shhh decrypt --all` - Decrypt all registered files

### Editing
- `shhh edit <file>` - Edit an encrypted file in $EDITOR
- `shhh reencrypt [file]` - Re-encrypt with current recipients

### Status
- `shhh status` - Show status of all registered files

## Encryption Modes

### Values Mode (default)
Encrypts only the values in structured files, preserving keys and structure:

```yaml
# Original
database:
  password: supersecret123

# Encrypted (.enc)
database:
  password: ENC[v1:BASE64_GPG_DATA]
_shhh:
  version: "1"
  vault: "default"
  mode: "values"
```

### Full Mode
Encrypts the entire file:

```
-----BEGIN SHHH ENCRYPTED FILE-----
Version: 1
Vault: default
Mode: full
Recipients: alice@example.com

BASE64_ENCODED_GPG_ENCRYPTED_CONTENT
-----END SHHH ENCRYPTED FILE-----
```

## Multi-Vault Setup

```bash
# Create a production vault
shhh vault create production

# Add users to production (only trusted admins)
shhh user add admin@example.com --vault production

# Register production secrets
shhh register prod-secrets.yaml --vault production

# Encrypt production secrets
shhh encrypt --vault production
```

## Per-File Recipients

```bash
# Restrict a file to specific users
shhh file set-recipients secrets.yaml alice@example.com

# Re-encrypt with new recipients
shhh reencrypt secrets.yaml

# Clear restrictions (use all vault users)
shhh file clear-recipients secrets.yaml
```

## Directory Structure

```
.shhh/
├── config.json           # Project configuration
├── vaults/
│   └── <vault-name>/
│       ├── users.json    # Users in this vault
│       └── files.json    # Registered files
└── pubkeys/
    └── <email>.asc       # Cached public keys
```

## Security

- Uses GPG multi-recipient encryption
- All sensitive files created with 0600 permissions
- .shhh/ directory created with 0700 permissions
- Plaintext files automatically added to .gitignore
- Key expiration tracking with warnings

## License

MIT
