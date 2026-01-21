# Configuration Reference

Restorable CLI stores its configuration in `~/.restorable/config.yaml`. This document describes all available configuration options.

## Configuration File Location

The configuration file is located at:

```
~/.restorable/config.yaml
```

This location is fixed and cannot be changed. Run `restorable init` to create an initial configuration.

## Complete Configuration Example

```yaml
version: 1

project:
  id: "prod-billing-db"
  name: "Production Billing Database"

cli:
  machine_id: "db-verify-01"
  report_dir: "~/.restorable/reports"
  temp_dir: "/tmp/restorable"

backup:
  source: "s3"
  retention_days: 30
  
  local:
    path: "/var/backups/postgres/latest.dump"
  
  s3:
    endpoint: "https://s3.eu-central-1.amazonaws.com"
    bucket: "company-backups"
    region: "eu-central-1"
    access_key_env: "RESTORABLE_S3_KEY"
    secret_key_env: "RESTORABLE_S3_SECRET"
    prefix: "billing-prod/"
  
  command:
    exec: "ssh backup@db-host 'cat /var/backups/latest.dump'"

encryption:
  method: "age"
  private_key_path: "~/.restorable/keys/backup.key"

database:
  type: "postgres"
  major_version: 15
  restore:
    docker_image: "postgres:15"
    user: "postgres"
    password_env: "RESTORABLE_DB_PASSWORD"
    db_name: "restorable_verify"
    port: 5432

verification:
  schema:
    enabled: true
  row_counts:
    enabled: true
    warn_threshold_percent: 5

docker:
  network: "bridge"
  pull_policy: "if-not-present"
  timeout_minutes: 30

signing:
  private_key_path: "~/.restorable/keys/signing.key"
```

## Configuration Sections

### version

```yaml
version: 1
```

Configuration format version. Always `1` for current releases.

---

### project

Project identification settings.

```yaml
project:
  id: "prod-billing-db"
  name: "Production Billing Database"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `id` | string | Yes | Unique identifier for the project. Used for baseline storage. |
| `name` | string | Yes | Human-readable project name. Appears in reports. |

---

### cli

CLI behavior settings.

```yaml
cli:
  machine_id: "db-verify-01"
  report_dir: "~/.restorable/reports"
  temp_dir: "/tmp/restorable"
```

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `machine_id` | string | No | `"db-verify-01"` | Identifier for this verification instance. |
| `report_dir` | string | No | `~/.restorable/reports` | Directory for storing reports. |
| `temp_dir` | string | No | `/tmp/restorable` | Temporary directory for backup processing. |

---

### backup

Backup source configuration. See [Backup Sources](backup-sources.md) for detailed examples.

```yaml
backup:
  source: "local"
  retention_days: 30
```

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `source` | string | Yes | - | Backup source type: `local`, `s3`, or `command`. |
| `retention_days` | int | No | 30 | Retention policy (informational, not enforced by CLI). |

#### backup.local

Local filesystem backup source.

```yaml
backup:
  source: "local"
  local:
    path: "/var/backups/postgres/latest.dump"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `path` | string | Yes (if source=local) | Absolute path to backup file. |

#### backup.s3

S3-compatible storage backup source.

```yaml
backup:
  source: "s3"
  s3:
    endpoint: "https://s3.eu-central-1.amazonaws.com"
    bucket: "company-backups"
    region: "eu-central-1"
    access_key_env: "RESTORABLE_S3_KEY"
    secret_key_env: "RESTORABLE_S3_SECRET"
    prefix: "billing-prod/latest.dump"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `endpoint` | string | Yes | S3-compatible endpoint URL. |
| `bucket` | string | Yes | Bucket name. |
| `region` | string | Yes | AWS region or compatible. |
| `access_key_env` | string | Yes | Environment variable name for access key. |
| `secret_key_env` | string | Yes | Environment variable name for secret key. |
| `prefix` | string | Yes | S3 key or prefix. If ends with `/`, fetches most recent object. |

#### backup.command

Custom command backup source.

```yaml
backup:
  source: "command"
  command:
    exec: "ssh backup@db-host 'cat /var/backups/latest.dump'"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `exec` | string | Yes (if source=command) | Shell command to execute. Stdout is the backup stream. |

---

### encryption

Backup encryption settings. See [Encryption](encryption.md) for setup instructions.

```yaml
encryption:
  method: "age"
  private_key_path: "~/.restorable/keys/backup.key"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `method` | string | No | Encryption method. Only `"age"` supported. |
| `private_key_path` | string | No | Path to age private key file. |

If `encryption` section is omitted, backups are assumed to be unencrypted.

---

### database

Database configuration for restore operations.

```yaml
database:
  type: "postgres"
  major_version: 15
  restore:
    docker_image: "postgres:15"
    user: "postgres"
    password_env: "RESTORABLE_DB_PASSWORD"
    db_name: "restorable_verify"
    port: 5432
```

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `type` | string | Yes | - | Database type. Only `"postgres"` supported. |
| `major_version` | int | Yes | - | PostgreSQL major version (11-16). |

#### database.restore

Restore container settings.

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `docker_image` | string | No | `postgres:{version}` | Docker image for restore container. |
| `user` | string | No | `"postgres"` | Database user for restore. |
| `password_env` | string | No | `"RESTORABLE_DB_PASSWORD"` | Environment variable for database password. |
| `db_name` | string | No | `"restorable_verify"` | Name of temporary database. |
| `port` | int | No | 5432 | Port inside container. |

---

### verification

Verification check configuration.

```yaml
verification:
  schema:
    enabled: true
  row_counts:
    enabled: true
    warn_threshold_percent: 5
```

#### verification.schema

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `enabled` | bool | No | true | Enable schema verification checks. |

#### verification.row_counts

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `enabled` | bool | No | true | Enable row count verification. |
| `warn_threshold_percent` | int | No | 5 | Warn if row count drops by more than this percentage. |

---

### docker

Docker configuration for containers.

```yaml
docker:
  network: "bridge"
  pull_policy: "if-not-present"
  timeout_minutes: 30
```

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `network` | string | No | `"bridge"` | Docker network mode. |
| `pull_policy` | string | No | `"if-not-present"` | Image pull policy: `always`, `never`, `if-not-present`. |
| `timeout_minutes` | int | No | 30 | Timeout for container operations. |

---

### signing

Report signing configuration.

```yaml
signing:
  private_key_path: "~/.restorable/keys/signing.key"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `private_key_path` | string | Yes | Path to Ed25519 private key for signing reports. |

The public key is derived from the private key path by replacing `.key` with `.pub`.

---

## Environment Variables

The following environment variables are used by Restorable:

| Variable | Required | Description |
|----------|----------|-------------|
| `RESTORABLE_DB_PASSWORD` | Always | Database password for restore container. |
| `RESTORABLE_S3_KEY` | If using S3 | AWS access key (or configured name). |
| `RESTORABLE_S3_SECRET` | If using S3 | AWS secret key (or configured name). |

## Configuration Tips

### Multiple Projects

To manage multiple projects, create separate configuration files and use symlinks:

```bash
# Create project-specific configs
mkdir -p ~/.restorable/projects
cp ~/.restorable/config.yaml ~/.restorable/projects/billing.yaml
cp ~/.restorable/config.yaml ~/.restorable/projects/users.yaml

# Switch between projects
ln -sf ~/.restorable/projects/billing.yaml ~/.restorable/config.yaml
```

### Testing Configuration

After modifying configuration, verify syntax:

```bash
# Run with verbose mode to see configuration loading
restorable verify -v
```

## Next Steps

- [Backup Sources](backup-sources.md) - Detailed backup source setup
- [Encryption](encryption.md) - Configure age encryption
- [Verification Checks](verification-checks.md) - Customize verification
