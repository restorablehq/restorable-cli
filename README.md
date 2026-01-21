# Restorable CLI

**Continuously prove your database backups can actually be restored.**

Restorable CLI is a command-line tool that verifies database backup integrity by performing actual restores in isolated environments. Instead of trusting that your backups work, Restorable proves it.

## Why Restorable?

A backup that hasn't been restored is assumed broken. Production teams often discover backup failures only during actual disasters. Restorable eliminates this risk by:

- **Actually restoring backups** in ephemeral containers
- **Validating schema integrity** against baselines
- **Tracking metrics** like row counts and restore duration
- **Generating signed reports** for compliance audits

## Quick Start

### Prerequisites

- Docker (for running ephemeral database containers)
- A PostgreSQL backup file (`.dump` or `.sql`)

### Installation

```bash
curl -fsSL https://get.restorable.io | sh
```

See [Installation](docs/installation.md) for alternative installation methods.

### Initialize a Project

```bash
restorable init
```

Follow the interactive prompts to configure:
- Project name
- Database type and version
- Backup source (local file, S3, or command)
- Optional encryption settings

### Run Your First Verification

```bash
# Set required environment variable
export RESTORABLE_DB_PASSWORD=yourpassword

# Run verification
restorable verify
```

### View Results

```bash
# List all reports
restorable report list

# Show details of a specific report
restorable report show <report-id>

# Verify report signature
restorable report verify <report-id>
```

## Features

- **Multiple Backup Sources**: Local files, S3-compatible storage, or custom commands (SSH, etc.)
- **Age Encryption**: Decrypt age-encrypted backups automatically
- **Ephemeral Containers**: Uses testcontainers for isolated, reproducible restores
- **Schema Validation**: Detect missing tables, schema drift, and structural changes
- **Metrics Tracking**: Monitor database size, row counts, and restore duration
- **Signed Reports**: Ed25519 signatures for tamper-proof audit trails
- **Compliance Ready**: Generate evidence for ISO 27001, NIS2, and SOC2

## Supported Databases

| Database   | Status |
|------------|--------|
| PostgreSQL | Supported |
| MySQL      | Planned |
| MongoDB    | Planned |

## Configuration

Restorable stores configuration in `~/.restorable/config.yaml`. Key settings include:

```yaml
project:
  name: "my-project"

backup:
  source: local  # local, s3, or command
  local:
    path: /path/to/backup.dump

database:
  type: postgres
  major_version: 15
```

See [Configuration Reference](docs/configuration.md) for all options.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed |
| 1 | Warnings only (partial success) |
| 2 | Critical failures |
| 3 | CLI/configuration error |
| 4 | Environment error |

## Documentation

- [Installation](docs/installation.md)
- [Getting Started](docs/getting-started.md)
- [Configuration Reference](docs/configuration.md)
- [CLI Commands](docs/commands.md)
- [Backup Sources](docs/backup-sources.md)
- [Encryption](docs/encryption.md)
- [Verification Checks](docs/verification-checks.md)
- [Reports](docs/reports.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Architecture](docs/architecture.md)

## Design Principles

1. **Restore > Backup**: A backup is worthless until proven restorable
2. **Deterministic**: Same input always produces the same result
3. **Fail loudly on real risk**: Critical issues stop the pipeline
4. **Zero production access**: Only verifies in isolated environments
5. **Stateless CLI**: Authority lives in signed reports

## License

[MIT License](LICENSE)

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.
