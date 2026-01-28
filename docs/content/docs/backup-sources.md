---
title: Backup Sources
type: docs
weight: 3
prev: docs/configuration
next: docs/verification-checks
---

Restorable CLI supports multiple backup source types to fit different infrastructure setups. This guide covers configuration and best practices for each source type.

## Overview

| Source Type | Use Case | Authentication |
|-------------|----------|----------------|
| `local` | Backups on local filesystem | File permissions |
| `s3` | AWS S3 or S3-compatible storage | Access key/secret |
| `command` | Custom retrieval (SSH, scripts) | Depends on command |

## Local Source

Use local source when backups are stored on the same machine or mounted filesystem.

### Configuration

```yaml
backup:
  source: "local"
  local:
    path: "/var/backups/postgres/latest.dump"
```

### Configuration Options

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `path` | string | Yes | Absolute path to backup file |

### Examples

#### Single Backup File

```yaml
backup:
  source: "local"
  local:
    path: "/var/backups/postgres/latest.dump"
```

#### Mounted NFS Share

```yaml
backup:
  source: "local"
  local:
    path: "/mnt/backup-server/databases/production.dump"
```

#### Encrypted Backup

```yaml
backup:
  source: "local"
  local:
    path: "/var/backups/postgres/latest.dump.age"

encryption:
  method: "age"
  private_key_path: "~/.restorable/keys/backup.key"
```

### Best Practices

- Use absolute paths to avoid working directory issues
- Ensure the backup file has appropriate read permissions
- For encrypted backups, use the `.age` extension by convention
- Consider using symlinks for "latest" backup pointers

---

## S3 Source

Use S3 source for backups stored in AWS S3 or S3-compatible storage (MinIO, DigitalOcean Spaces, Backblaze B2, etc.).

### Configuration

```yaml
backup:
  source: "s3"
  s3:
    endpoint: "https://s3.eu-central-1.amazonaws.com"
    bucket: "company-backups"
    region: "eu-central-1"
    access_key_env: "RESTORABLE_S3_KEY"
    secret_key_env: "RESTORABLE_S3_SECRET"
    prefix: "postgres/production/"
```

### Configuration Options

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `endpoint` | string | Yes | S3-compatible endpoint URL |
| `bucket` | string | Yes | Bucket name |
| `region` | string | Yes | AWS region or compatible |
| `access_key_env` | string | Yes | Environment variable name for access key |
| `secret_key_env` | string | Yes | Environment variable name for secret key |
| `prefix` | string | Yes | S3 key or prefix path |

### Prefix Behavior

The `prefix` field has two behaviors:

1. **Exact key** (no trailing `/`): Fetches the specific object
   ```yaml
   prefix: "postgres/production/backup.dump"
   # Fetches: s3://bucket/postgres/production/backup.dump
   ```

2. **Prefix with trailing `/`**: Lists objects and fetches the most recent
   ```yaml
   prefix: "postgres/production/"
   # Lists all objects under prefix, downloads newest by LastModified
   ```

### Examples

#### AWS S3

```yaml
backup:
  source: "s3"
  s3:
    endpoint: "https://s3.us-east-1.amazonaws.com"
    bucket: "my-backups"
    region: "us-east-1"
    access_key_env: "AWS_ACCESS_KEY_ID"
    secret_key_env: "AWS_SECRET_ACCESS_KEY"
    prefix: "databases/prod/"
```

```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

#### MinIO

```yaml
backup:
  source: "s3"
  s3:
    endpoint: "https://minio.internal.company.com"
    bucket: "backups"
    region: "us-east-1"  # MinIO accepts any region
    access_key_env: "MINIO_ACCESS_KEY"
    secret_key_env: "MINIO_SECRET_KEY"
    prefix: "postgres/"
```

#### DigitalOcean Spaces

```yaml
backup:
  source: "s3"
  s3:
    endpoint: "https://fra1.digitaloceanspaces.com"
    bucket: "company-backups"
    region: "fra1"
    access_key_env: "DO_SPACES_KEY"
    secret_key_env: "DO_SPACES_SECRET"
    prefix: "db-backups/latest.dump"
```

#### Backblaze B2

```yaml
backup:
  source: "s3"
  s3:
    endpoint: "https://s3.us-west-002.backblazeb2.com"
    bucket: "my-bucket"
    region: "us-west-002"
    access_key_env: "B2_KEY_ID"
    secret_key_env: "B2_APPLICATION_KEY"
    prefix: "backups/"
```

### IAM Policy (AWS)

Minimum required permissions for the S3 source:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-backup-bucket",
        "arn:aws:s3:::my-backup-bucket/*"
      ]
    }
  ]
}
```

### Best Practices

- Use dedicated credentials with minimal permissions
- Store credentials in environment variables, not config files
- Use prefix with trailing `/` to automatically get latest backup
- Enable versioning on your S3 bucket for backup history
- Consider using VPC endpoints for private access

---

## Command Source

Use command source for custom backup retrieval via shell commands. Useful for SSH, custom scripts, or complex workflows.

### Configuration

```yaml
backup:
  source: "command"
  command:
    exec: "ssh backup@db-host 'cat /var/backups/latest.dump'"
```

### Configuration Options

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `exec` | string | Yes | Shell command to execute |

### How It Works

1. Command is executed via `/bin/sh -c`
2. **stdout** is captured as the backup stream
3. **stderr** is logged for debugging
4. Command must exit with code 0
5. Default timeout: 10 minutes

### Examples

#### SSH Remote Fetch

```yaml
backup:
  source: "command"
  command:
    exec: "ssh backup@db-host 'cat /var/backups/postgres/latest.dump'"
```

Ensure SSH key authentication is configured:
```bash
# Set up SSH key (run once)
ssh-copy-id backup@db-host
```

#### SSH with Compression

```yaml
backup:
  source: "command"
  command:
    exec: "ssh backup@db-host 'gzip -c /var/backups/latest.dump' | gunzip"
```

#### Custom Script

```yaml
backup:
  source: "command"
  command:
    exec: "/usr/local/bin/fetch-latest-backup.sh"
```

Example script (`fetch-latest-backup.sh`):
```bash
#!/bin/bash
set -e

# Find latest backup
LATEST=$(ls -t /var/backups/postgres/*.dump 2>/dev/null | head -1)

if [ -z "$LATEST" ]; then
  echo "No backup found" >&2
  exit 1
fi

# Output to stdout
cat "$LATEST"
```

#### rsync + Local Read

```yaml
backup:
  source: "command"
  command:
    exec: "rsync -az backup@db-host:/var/backups/latest.dump /tmp/backup.dump >&2 && cat /tmp/backup.dump"
```

Note: rsync output goes to stderr (`>&2`), backup content to stdout.

#### AWS CLI

```yaml
backup:
  source: "command"
  command:
    exec: "aws s3 cp s3://my-bucket/backups/latest.dump -"
```

#### kubectl (Kubernetes)

```yaml
backup:
  source: "command"
  command:
    exec: "kubectl exec -n database postgres-0 -- pg_dump -Fc mydb"
```

### Environment Variables

Commands inherit the shell environment, so you can use environment variables:

```yaml
backup:
  source: "command"
  command:
    exec: "ssh $BACKUP_HOST 'cat /var/backups/latest.dump'"
```

```bash
export BACKUP_HOST=backup@192.168.1.100
restorable verify
```

### Best Practices

- Always use `set -e` in scripts to fail on errors
- Send progress/debug output to stderr, only backup data to stdout
- Test commands manually before configuring
- Use absolute paths in scripts
- Handle missing backups gracefully with clear error messages
- Consider compression for large backups over slow networks

---

## Choosing a Source Type

| Scenario | Recommended Source |
|----------|-------------------|
| Backups on local disk | `local` |
| Backups in cloud storage | `s3` |
| Backups on remote server | `command` (SSH) |
| Complex retrieval logic | `command` (script) |
| Kubernetes deployments | `command` (kubectl) |
| Multiple fallback sources | `command` (script) |

## Troubleshooting

### Local Source

**Error: "no such file or directory"**
- Verify the path exists and is readable
- Check file permissions: `ls -la /path/to/backup`

### S3 Source

**Error: "access denied"**
- Verify environment variables are set
- Check IAM permissions include `s3:GetObject` and `s3:ListBucket`
- Verify bucket policy allows access

**Error: "no such bucket"**
- Check bucket name and region match
- Verify endpoint URL is correct

### Command Source

**Error: "command failed with exit code X"**
- Run the command manually to see full output
- Check stderr for error messages
- Verify SSH keys are configured for remote hosts

**Timeout errors**
- Large backups may exceed the 10-minute default timeout
- Consider compressing backups or increasing available bandwidth

## Next Steps

- [Encryption](encryption.md) - Configure backup encryption
- [Configuration](configuration.md) - Full configuration reference
