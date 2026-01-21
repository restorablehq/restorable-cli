# Getting Started

This guide walks you through your first backup verification with Restorable CLI.

## Overview

The verification process consists of three main steps:

1. **Initialize** - Set up your project configuration
2. **Verify** - Run the backup verification
3. **Review** - Examine the verification report

## Step 1: Initialize Your Project

Run the interactive setup wizard:

```bash
restorable init
```

You'll be prompted for:

### Project Name
A human-readable name for your project (e.g., "Production Database").

### Database Type
Currently supported: `postgres`

### Database Major Version
The PostgreSQL major version (e.g., `15`, `16`). This determines which Docker image is used.

### Backup Source Type

Choose how Restorable accesses your backup:

- **local** - Backup file on local filesystem
- **s3** - Backup stored in S3-compatible storage
- **command** - Custom command to fetch backup (e.g., SSH)

### Encryption

If your backups are encrypted with [age](https://github.com/FiloSottile/age):

1. Enter the path to your age private key
2. Or skip if backups are not encrypted

### Output

After completion, you'll have:

```
~/.restorable/
├── config.yaml           # Your configuration
└── keys/
    ├── signing.key       # Private key for signing reports
    └── signing.pub       # Public key for verification
```

## Step 2: Set Environment Variables

Before running verification, set required environment variables:

```bash
# Always required: Database password for the restore container
export RESTORABLE_DB_PASSWORD=yourpassword

# If using S3 backup source:
export RESTORABLE_S3_KEY=your-access-key
export RESTORABLE_S3_SECRET=your-secret-key
```

## Step 3: Run Verification

Execute the verification:

```bash
restorable verify
```

### What Happens During Verification

1. **Load Configuration** - Reads `~/.restorable/config.yaml`
2. **Acquire Backup** - Fetches backup from configured source
3. **Decrypt** - Decrypts if encryption is configured
4. **Start Container** - Launches ephemeral PostgreSQL container
5. **Restore** - Runs `pg_restore` (or `psql` for plain SQL)
6. **Extract Schema** - Captures table structure and columns
7. **Extract Metrics** - Captures database size and row counts
8. **Run Checks** - Validates against baseline (if exists)
9. **Generate Report** - Creates signed JSON report
10. **Cleanup** - Terminates container

### Verbose Mode

For detailed output including restore logs:

```bash
restorable verify -v
# or
restorable verify --verbose
```

### Example Output

```
Loading configuration...
Acquiring backup from local:/backups/db.dump
Starting PostgreSQL 15 container...
Restoring backup...
Extracting schema...
Extracting metrics...
Running verification checks...

✓ tables_exist: All 12 baseline tables exist
✓ table_count: Table count matches baseline (12)
ℹ new_tables: No new tables detected
✓ non_empty_tables: 10 tables have data
✓ total_row_count: 15,234 total rows

Verification complete!
Report saved: ~/.restorable/reports/2024-01-15T10-30-00Z_abc123.json

Summary:
  Status: SUCCESS
  Checks: 5 passed, 0 failed
  Duration: 45s
  Database size: 256 MB
```

## Step 4: Review Reports

### List All Reports

```bash
restorable report list
```

Output:
```
ID                                    TIMESTAMP             PROJECT              STATUS
abc123                                2024-01-15 10:30:00   Production Database  SUCCESS
def456                                2024-01-14 10:30:00   Production Database  SUCCESS
```

### View Report Details

```bash
restorable report show abc123
```

This displays:
- Report metadata
- Database information
- All check results
- Summary statistics

### Verify Report Signature

Confirm a report hasn't been tampered with:

```bash
restorable report verify abc123
```

Output:
```
✓ Signature valid
```

## Understanding Check Results

### Check Levels

| Level | Meaning | Impact |
|-------|---------|--------|
| `critical` | Blocking failure | Exit code 2 |
| `warning` | Concerning but not blocking | Exit code 1 |
| `info` | Informational | No impact |

### Common Checks

- **tables_exist** (critical) - All baseline tables present
- **table_count** (warning) - Table count matches baseline
- **new_tables** (info) - Reports newly added tables
- **non_empty_tables** (warning) - Tables contain data
- **total_row_count** (warning) - Total rows above threshold

## First Run vs Subsequent Runs

### First Run (No Baseline)

On your first verification:
- Schema checks auto-pass (no baseline to compare)
- Current schema becomes the new baseline
- Baseline saved to `~/.restorable/schemas/`

### Subsequent Runs

On later verifications:
- Schema compared against baseline
- Missing tables trigger critical failure
- Row count drops trigger warnings
- Baseline updated after successful runs

## Automation

### Cron Job

Run daily verification at 2 AM:

```bash
# Edit crontab
crontab -e

# Add line:
0 2 * * * RESTORABLE_DB_PASSWORD=yourpassword /usr/local/bin/restorable verify
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Verify Backup
  env:
    RESTORABLE_DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
  run: |
    restorable verify
    if [ $? -eq 2 ]; then
      echo "Critical verification failure!"
      exit 1
    fi
```

## Next Steps

- [Configuration Reference](configuration.md) - Customize all settings
- [Backup Sources](backup-sources.md) - Configure S3 or custom commands
- [Verification Checks](verification-checks.md) - Understand all checks
- [Reports](reports.md) - Work with verification reports
