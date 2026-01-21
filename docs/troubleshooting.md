# Troubleshooting

This guide covers common issues and their solutions when using Restorable CLI.

## Quick Diagnosis

### Check Exit Code

```bash
restorable verify
echo "Exit code: $?"
```

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | No action needed |
| 1 | Warnings | Review report for concerns |
| 2 | Critical failure | Investigate immediately |
| 3 | CLI/config error | Check configuration |
| 4 | Environment error | Check Docker, network |

### Enable Verbose Mode

```bash
restorable verify -v
```

Verbose mode shows:
- Detailed configuration loading
- Backup acquisition progress
- Full restore output
- Check execution details

## Common Issues

### Configuration Issues

#### "config file not found"

**Cause:** Restorable cannot find `~/.restorable/config.yaml`.

**Solution:**
```bash
# Initialize configuration
restorable init
```

#### "invalid configuration"

**Cause:** YAML syntax error or missing required fields.

**Solution:**
```bash
# Validate YAML syntax
cat ~/.restorable/config.yaml | python3 -c "import yaml, sys; yaml.safe_load(sys.stdin)"

# Check required fields exist
grep -E "^(project|backup|database):" ~/.restorable/config.yaml
```

#### "unknown backup source type"

**Cause:** Invalid `backup.source` value.

**Solution:** Use one of: `local`, `s3`, `command`

```yaml
backup:
  source: "local"  # or "s3" or "command"
```

---

### Backup Acquisition Issues

#### "no such file or directory" (local source)

**Cause:** Backup file does not exist at specified path.

**Solution:**
```bash
# Verify file exists
ls -la /path/to/your/backup.dump

# Check path in config
grep -A2 "local:" ~/.restorable/config.yaml
```

#### "access denied" (S3 source)

**Cause:** Invalid or missing S3 credentials.

**Solution:**
```bash
# Verify environment variables are set
echo "S3 Key: ${RESTORABLE_S3_KEY:-(not set)}"
echo "S3 Secret: ${RESTORABLE_S3_SECRET:-(not set)}"

# Test S3 access
aws s3 ls s3://your-bucket/your-prefix/
```

#### "no objects found in prefix" (S3 source)

**Cause:** S3 prefix has no matching objects.

**Solution:**
```bash
# List objects in the prefix
aws s3 ls s3://your-bucket/your-prefix/

# Verify prefix in config (trailing / means list objects)
grep "prefix:" ~/.restorable/config.yaml
```

#### "command failed" (command source)

**Cause:** The shell command returned non-zero exit code.

**Solution:**
```bash
# Test command manually
ssh backup@db-host 'cat /var/backups/latest.dump' > /dev/null
echo "Exit code: $?"

# Check command in config
grep -A2 "command:" ~/.restorable/config.yaml
```

---

### Encryption Issues

#### "failed to decrypt"

**Cause:** Wrong private key or corrupted backup.

**Solution:**
```bash
# Test decryption manually
age -d -i ~/.restorable/keys/backup.key backup.dump.age > /dev/null

# Verify key path in config
grep "private_key_path:" ~/.restorable/config.yaml
```

#### "no identity matched any of the recipients"

**Cause:** Backup was encrypted with a different public key.

**Solution:**
```bash
# Check which public key encrypted the backup
age -d -i ~/.restorable/keys/backup.key backup.dump.age 2>&1 | head -5

# Verify you have the correct private key
head -3 ~/.restorable/keys/backup.key
```

---

### Docker Issues

#### "Cannot connect to Docker daemon"

**Cause:** Docker is not running or not accessible.

**Solution:**
```bash
# Check Docker is running
docker info

# Linux: Start Docker daemon
sudo systemctl start docker

# macOS: Start Docker Desktop from Applications
```

#### "permission denied" for Docker

**Cause:** User not in docker group (Linux).

**Solution:**
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and back in, then verify
docker run hello-world
```

#### "image pull failed"

**Cause:** Cannot pull PostgreSQL image.

**Solution:**
```bash
# Test pulling image manually
docker pull postgres:15

# Check Docker Hub connectivity
curl -s https://hub.docker.com/v2/ | head -1
```

#### "container startup timeout"

**Cause:** Container takes too long to become ready.

**Solution:**
```yaml
# Increase timeout in config
docker:
  timeout_minutes: 60
```

---

### Restore Issues

#### "pg_restore: error"

**Cause:** Backup format incompatible or corrupted.

**Solution:**
```bash
# Check backup file type
file /path/to/backup.dump

# Test restore manually
pg_restore --list /path/to/backup.dump

# If it is a plain SQL file, it should start with:
head -5 /path/to/backup.dump
```

#### "FATAL: password authentication failed"

**Cause:** Missing or incorrect database password.

**Solution:**
```bash
# Verify password environment variable
echo "Password var: ${RESTORABLE_DB_PASSWORD:-(not set)}"

# Check which env var is configured
grep "password_env:" ~/.restorable/config.yaml
```

#### "restore succeeded but database is empty"

**Cause:** Backup is schema-only or from empty database.

**Solution:**
```bash
# Check backup contents
pg_restore --list /path/to/backup.dump | head -20

# Verify data was included in backup
# Look for "TABLE DATA" entries
pg_restore --list /path/to/backup.dump | grep "TABLE DATA"
```

---

### Verification Issues

#### "tables_exist: Missing tables"

**Cause:** Restored database missing expected tables.

**Possible Causes:**
1. Wrong backup file
2. Partial backup (missing schemas)
3. Backup from different database
4. Baseline is outdated

**Solution:**
```bash
# Check which tables are in the backup
pg_restore --list /path/to/backup.dump | grep "TABLE"

# Reset baseline if schema intentionally changed
rm ~/.restorable/schemas/your-project-id.json
restorable verify
```

#### "row_count decreased significantly"

**Cause:** Current backup has fewer rows than baseline.

**Possible Causes:**
1. Data was deleted in production
2. Backup is from test environment
3. Backup process filtered data

**Solution:**
```bash
# Check table sizes in backup
pg_restore --list /path/to/backup.dump | grep "TABLE DATA"

# Adjust threshold if expected
```
```yaml
verification:
  row_counts:
    warn_threshold_percent: 20  # Increase threshold
```

---

### Report Issues

#### "report not found"

**Cause:** Invalid report ID or report deleted.

**Solution:**
```bash
# List available reports
restorable report list

# Check reports directory
ls ~/.restorable/reports/
```

#### "signature invalid"

**Cause:** Report was modified or wrong key.

**Solution:**
```bash
# Verify signing key exists
ls -la ~/.restorable/keys/signing.*

# Check report file is valid JSON
jq . ~/.restorable/reports/your-report.json > /dev/null
```

---

## Environment-Specific Issues

### CI/CD Environments

#### Docker-in-Docker Issues

```yaml
# GitHub Actions example
services:
  docker:
    image: docker:dind
    options: --privileged

steps:
  - name: Run verification
    env:
      RESTORABLE_DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
      DOCKER_HOST: tcp://docker:2375
    run: restorable verify
```

#### Missing Environment Variables

```yaml
# Ensure all required vars are set
env:
  RESTORABLE_DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
  RESTORABLE_S3_KEY: ${{ secrets.S3_KEY }}
  RESTORABLE_S3_SECRET: ${{ secrets.S3_SECRET }}
```

### Kubernetes

#### Pod Security Restrictions

If running in a restricted pod:

```yaml
# Ensure pod can run Docker or use testcontainers cloud
securityContext:
  privileged: true  # Required for Docker-in-Docker
```

#### Resource Limits

```yaml
resources:
  limits:
    memory: "2Gi"  # Increase for large databases
    cpu: "1"
```

---

## Diagnostic Commands

### Full Diagnostic Script

```bash
#!/bin/bash
echo "=== Restorable Diagnostic ==="
echo

echo "Version:"
restorable version
echo

echo "Configuration:"
cat ~/.restorable/config.yaml 2>/dev/null || echo "Config not found"
echo

echo "Environment:"
echo "DB Password: ${RESTORABLE_DB_PASSWORD:+set}"
echo "S3 Key: ${RESTORABLE_S3_KEY:+set}"
echo "S3 Secret: ${RESTORABLE_S3_SECRET:+set}"
echo

echo "Docker:"
docker info 2>&1 | head -5
echo

echo "Keys:"
ls -la ~/.restorable/keys/ 2>/dev/null || echo "Keys not found"
echo

echo "Reports:"
ls ~/.restorable/reports/ 2>/dev/null | wc -l
echo "report files"
echo

echo "Baseline:"
ls ~/.restorable/schemas/ 2>/dev/null || echo "No baseline"
```

### Test Backup Access

```bash
# Test local file
ls -la /path/to/backup.dump

# Test S3
aws s3 ls s3://bucket/prefix/

# Test SSH command
ssh backup@host 'ls -la /var/backups/'
```

### Test Docker

```bash
# Basic Docker test
docker run --rm postgres:15 pg_isready --version

# Test container creation
docker run --rm -e POSTGRES_PASSWORD=test postgres:15 postgres --version
```

## Getting Help

### Log Collection

When reporting issues, include:

```bash
# Collect diagnostic info
restorable verify -v 2>&1 | tee restorable-debug.log
cat ~/.restorable/config.yaml >> restorable-debug.log
docker info >> restorable-debug.log 2>&1
```

### Filing Issues

Include:
1. Restorable version (`restorable version`)
2. Operating system and version
3. Docker version (`docker version`)
4. Anonymized configuration
5. Full error output with verbose mode

## Next Steps

- [Configuration](configuration.md) - Review configuration options
- [Backup Sources](backup-sources.md) - Backup source details
- [Commands](commands.md) - CLI reference
