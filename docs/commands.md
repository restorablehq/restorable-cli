# CLI Commands Reference

This document describes all available Restorable CLI commands and their options.

## Global Usage

```bash
restorable [command] [flags]
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `init` | Initialize a new Restorable project |
| `verify` | Run backup verification |
| `report` | Manage verification reports |
| `version` | Print CLI version |

---

## restorable init

Initialize a new Restorable project with configuration and signing keys.

### Usage

```bash
restorable init
```

### Description

The `init` command runs an interactive setup wizard that creates:

- `~/.restorable/config.yaml` - Main configuration file
- `~/.restorable/keys/signing.key` - Ed25519 private key for signing reports
- `~/.restorable/keys/signing.pub` - Ed25519 public key for verification
- `~/.restorable/keys/backup.key` - Age encryption key path (if configured)

### Interactive Prompts

1. **Project name** - Human-readable name for your project
2. **Database type** - Currently only `postgres` is supported
3. **Database major version** - PostgreSQL major version (e.g., 15)
4. **Backup source type** - `local`, `s3`, or `command`
5. **Source-specific settings** - Path, S3 details, or command
6. **Encryption** - Optional age encryption configuration

### Example

```bash
$ restorable init

Welcome to Restorable CLI setup!

Enter project name: Production Database
Enter database type [postgres]: postgres
Enter database major version [15]: 15
Select backup source (local/s3/command): local
Enter backup file path: /var/backups/db.dump
Use encryption? (y/n): n

Configuration saved to ~/.restorable/config.yaml
Signing keys generated in ~/.restorable/keys/

Run 'restorable verify' to start verification.
```

---

## restorable verify

Run end-to-end backup verification.

### Usage

```bash
restorable verify [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output with full restore logs |

### Description

The `verify` command performs a complete backup verification cycle:

1. Loads configuration from `~/.restorable/config.yaml`
2. Acquires backup from configured source
3. Decrypts backup (if encryption configured)
4. Starts ephemeral PostgreSQL container
5. Restores backup using `pg_restore` or `psql`
6. Extracts database schema and metrics
7. Compares against baseline (if exists)
8. Runs verification checks
9. Generates and signs verification report
10. Saves report to `~/.restorable/reports/`
11. Updates baseline schema

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `RESTORABLE_DB_PASSWORD` | Yes | Password for restore container |
| `RESTORABLE_S3_KEY` | If using S3 | AWS access key |
| `RESTORABLE_S3_SECRET` | If using S3 | AWS secret key |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed |
| 1 | Warnings only (partial success) |
| 2 | Critical failures |
| 3 | CLI/configuration error |
| 4 | Environment error (Docker, network, etc.) |

### Example

```bash
# Basic verification
$ restorable verify

Loading configuration...
Acquiring backup from local:/var/backups/db.dump
Starting PostgreSQL 15 container...
Restoring backup...
Running verification checks...

✓ tables_exist: All 12 baseline tables exist
✓ table_count: Table count matches baseline (12)

Verification complete!
Report saved: ~/.restorable/reports/2024-01-15T10-30-00Z_abc123.json

# Verbose mode
$ restorable verify -v

Loading configuration...
  Project: Production Database
  Database: postgres:15
  Backup source: local:/var/backups/db.dump

Acquiring backup...
  File size: 15.2 MB

Starting container...
  Image: postgres:15
  Container ID: abc123def456

Restoring backup...
  pg_restore: setting owner for table "users"
  pg_restore: setting owner for table "orders"
  ...

Verification complete!
```

---

## restorable report

Manage verification reports.

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List all verification reports |
| `show` | Display a specific report |
| `verify` | Verify a report's signature |

---

### restorable report list

List all verification reports.

#### Usage

```bash
restorable report list
```

#### Description

Lists all reports in the report directory, sorted by timestamp (newest first).

#### Output Columns

- **ID** - Report UUID (truncated for display)
- **TIMESTAMP** - Report creation time
- **PROJECT** - Project name
- **STATUS** - SUCCESS or FAILURE

#### Example

```bash
$ restorable report list

ID        TIMESTAMP             PROJECT              STATUS
abc123    2024-01-15 10:30:00   Production Database  SUCCESS
def456    2024-01-14 10:30:00   Production Database  SUCCESS
ghi789    2024-01-13 10:30:00   Production Database  FAILURE
```

---

### restorable report show

Display detailed information about a specific report.

#### Usage

```bash
restorable report show <report-id> [flags]
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `report-id` | Full or partial report ID (prefix matching supported) |

#### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON instead of formatted text |

#### Description

Displays comprehensive report information including:

- Report metadata (ID, timestamp, project, machine)
- Database information (type, version, size)
- Verification summary (status, check counts)
- Individual check results
- Signature information

#### Example

```bash
$ restorable report show abc123

Report: abc123-def4-5678-90ab-cdef12345678
Generated: 2024-01-15 10:30:00 UTC
Project: Production Database
Machine: db-verify-01

Database:
  Type: postgres
  Version: 15
  Size: 256 MB

Summary:
  Status: SUCCESS
  Total Checks: 5
  Passed: 5
  Failed: 0
  Duration: 45s

Checks:
  ✓ [critical] tables_exist: All 12 baseline tables exist
  ✓ [warning]  table_count: Table count matches baseline (12)
  ✓ [info]     new_tables: No new tables detected
  ✓ [warning]  non_empty_tables: 10 tables have data
  ✓ [warning]  total_row_count: 15234 total rows

Signature: Valid (Ed25519)

# JSON output
$ restorable report show abc123 --json
{
  "version": "1",
  "id": "abc123-def4-5678-90ab-cdef12345678",
  ...
}
```

---

### restorable report verify

Verify the cryptographic signature of a report.

#### Usage

```bash
restorable report verify <report-id>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `report-id` | Full or partial report ID |

#### Description

Validates the Ed25519 signature to ensure the report hasn't been tampered with. Uses the public key derived from the configured signing key path.

#### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Signature valid |
| 1 | Signature invalid or verification failed |

#### Example

```bash
$ restorable report verify abc123
✓ Signature valid

$ restorable report verify tampered123
✗ Signature invalid: signature mismatch
```

---

## restorable version

Print the CLI version.

### Usage

```bash
restorable version
```

### Example

```bash
$ restorable version
restorable version 0.1.0
```

---

## Shell Completion

Generate shell completion scripts for your shell.

### Bash

```bash
restorable completion bash > /etc/bash_completion.d/restorable
```

### Zsh

```bash
restorable completion zsh > "${fpath[1]}/_restorable"
```

### Fish

```bash
restorable completion fish > ~/.config/fish/completions/restorable.fish
```

---

## Common Usage Patterns

### Daily Verification Cron

```bash
# Run at 2 AM daily
0 2 * * * RESTORABLE_DB_PASSWORD=pass /usr/local/bin/restorable verify
```

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Run verification
restorable verify

# Check exit code
case $? in
  0) echo "All checks passed" ;;
  1) echo "Warnings detected, review report" ;;
  2) echo "Critical failure!"; exit 1 ;;
  *) echo "Error occurred"; exit 1 ;;
esac

# Get latest report
REPORT_ID=$(restorable report list | head -2 | tail -1 | awk '{print $1}')
restorable report show "$REPORT_ID" --json > report.json
```

### Monitoring Integration

```bash
#!/bin/bash
# Send alert on failure

restorable verify
EXIT_CODE=$?

if [ $EXIT_CODE -eq 2 ]; then
  curl -X POST https://alerts.example.com/webhook \
    -H "Content-Type: application/json" \
    -d '{"alert": "Backup verification failed", "severity": "critical"}'
fi
```

## Next Steps

- [Getting Started](getting-started.md) - First verification walkthrough
- [Configuration](configuration.md) - Full configuration reference
- [Verification Checks](verification-checks.md) - Understanding check results
