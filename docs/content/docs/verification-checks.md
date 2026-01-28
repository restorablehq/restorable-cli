---
title: Verification
type: docs
weight: 4
prev: docs/backup-sources
next: docs/reports
---

Restorable CLI runs a series of verification checks after restoring your backup. This document explains each check, its purpose, and how to interpret results.

## Check Levels

Each check has a severity level that affects the exit code:

| Level | Description | Exit Code Impact |
|-------|-------------|------------------|
| `critical` | Blocking failure - backup is unusable | Exit code 2 |
| `warning` | Concerning issue - backup works but has problems | Exit code 1 |
| `info` | Informational - no action required | No impact |

## Exit Code Summary

| Code | Meaning |
|------|---------|
| 0 | All checks passed |
| 1 | Only warnings (backup usable but review recommended) |
| 2 | Critical failure (backup cannot be relied upon) |

## Available Checks

### tables_exist

**Level:** Critical

**Purpose:** Verifies all tables from the baseline schema exist in the restored database.

**Behavior:**
- First run (no baseline): Auto-passes
- Subsequent runs: Compares against baseline

**Pass Condition:** All baseline tables present in restored database.

**Failure Example:**
```
✗ [critical] tables_exist: Missing tables: users, orders, payments
```

**Common Causes:**
- Backup is from wrong database
- Backup was created before tables existed
- Backup corruption
- Partial backup (only some schemas included)

**Resolution:**
- Verify backup source is correct
- Check backup creation process includes all tables
- Review pg_dump options (ensure no schema filters)

---

### table_count

**Level:** Warning

**Purpose:** Checks if the total number of tables matches the baseline.

**Behavior:**
- First run (no baseline): Auto-passes
- Table increases: Passes (new tables are OK)
- Table decreases: Warning

**Pass Condition:** Current table count >= baseline table count.

**Failure Example:**
```
⚠ [warning] table_count: Table count decreased from 15 to 12
```

**Common Causes:**
- Tables were intentionally removed (expected)
- Backup doesn't include all schemas
- Different database selected

**Resolution:**
- If intentional: Update baseline by running successful verification
- If unintentional: Check backup configuration

---

### new_tables

**Level:** Info

**Purpose:** Reports any tables present in the restored database that weren't in the baseline.

**Behavior:**
- Always informational (never fails)
- Helps track schema evolution

**Output Example:**
```
ℹ [info] new_tables: New tables found: audit_logs, feature_flags
```

**Use Case:**
- Track schema changes over time
- Verify expected new tables appear
- Detect unexpected schema additions

---

### non_empty_tables

**Level:** Warning

**Purpose:** Ensures at least some tables contain data.

**Behavior:**
- Counts tables with row count > 0
- Compares against configured minimum

**Pass Condition:** Number of non-empty tables >= minimum threshold.

**Configuration:**
```yaml
verification:
  row_counts:
    enabled: true
```

**Failure Example:**
```
⚠ [warning] non_empty_tables: Only 2 tables have data, expected at least 5
```

**Common Causes:**
- Backup taken from empty/test database
- Backup process excluded data
- Schema-only backup

---

### total_row_count

**Level:** Warning

**Purpose:** Verifies total database row count meets minimum threshold.

**Behavior:**
- Sums row counts across all tables
- Compares against threshold

**Pass Condition:** Total rows >= minimum threshold.

**Failure Example:**
```
⚠ [warning] total_row_count: Total rows (150) below threshold (1000)
```

**Common Causes:**
- Backup from test/staging environment
- Data truncation during backup
- Recent data deletion in production

---

### restore_duration

**Level:** Info

**Purpose:** Tracks how long the restore process took.

**Behavior:**
- Always passes (informational)
- Can optionally warn if duration exceeds threshold

**Output Example:**
```
ℹ [info] restore_duration: Restore completed in 2m 35s
```

**Use Case:**
- Monitor backup size growth over time
- Detect performance issues
- Plan RTO (Recovery Time Objective)

---

## Baseline System

### What is a Baseline?

The baseline is a snapshot of your database schema from a previous successful verification. It's used to detect changes and regressions.

### Baseline Location

Baselines are stored in `~/.restorable/schemas/{project_id}.json`.

### First Run Behavior

On first verification (no baseline exists):
- Schema checks auto-pass
- Current schema becomes the new baseline
- Subsequent runs compare against this baseline

### Baseline Updates

The baseline is updated when:
- First successful verification runs
- A verification completes without critical failures

### Resetting the Baseline

To reset and establish a new baseline:

```bash
# Remove existing baseline
rm ~/.restorable/schemas/{project_id}.json

# Run verification (creates new baseline)
restorable verify
```

---

## Check Configuration

### Enabling/Disabling Checks

```yaml
verification:
  schema:
    enabled: true  # Enable schema checks (tables_exist, table_count, new_tables)
  row_counts:
    enabled: true  # Enable row count checks
```

### Row Count Threshold

```yaml
verification:
  row_counts:
    warn_threshold_percent: 5  # Warn if row count drops >5%
```

---

## Interpreting Results

### All Checks Pass (Exit 0)

```
✓ tables_exist: All 12 baseline tables exist
✓ table_count: Table count matches baseline (12)
ℹ new_tables: No new tables detected
✓ non_empty_tables: 10 tables have data
✓ total_row_count: 15234 total rows

Summary: SUCCESS
```

**Meaning:** Backup is verified and matches expectations.

### Warnings Only (Exit 1)

```
✓ tables_exist: All 12 baseline tables exist
⚠ table_count: Table count decreased from 15 to 12
ℹ new_tables: No new tables detected
✓ non_empty_tables: 10 tables have data

Summary: PARTIAL SUCCESS (warnings)
```

**Meaning:** Backup restored successfully but has concerning changes. Review the warnings.

### Critical Failure (Exit 2)

```
✗ tables_exist: Missing tables: users, payments

Summary: FAILURE (critical)
```

**Meaning:** Backup is unreliable. Investigate immediately.

---

## Best Practices

### 1. Monitor Trends

Track check results over time to identify:
- Gradual row count changes
- Schema evolution
- Restore duration trends

### 2. Set Appropriate Thresholds

Configure thresholds based on your data:
- High-volume databases: Higher row count minimums
- Stable schemas: Stricter table count checks
- Growing databases: Allow table increases

### 3. Investigate Warnings

Don't ignore warnings. They often indicate:
- Configuration drift
- Backup process issues
- Unexpected data changes

### 4. Regular Baseline Updates

After intentional schema changes:
1. Verify the change is expected
2. Run verification to update baseline
3. Confirm subsequent runs pass

---

## Custom Checks

Currently, Restorable includes a fixed set of checks. Future versions may support:
- Custom SQL validation queries
- Application-specific integrity checks
- External validation webhooks

---

## Troubleshooting

### Check Always Fails

1. Verify backup source is correct
2. Check baseline is from same database
3. Try resetting baseline (see above)

### Inconsistent Results

1. Check if backup source changes between runs
2. Verify container is cleaned up properly
3. Check for race conditions in backup creation

### False Positives

1. Review threshold configuration
2. Check if schema legitimately changed
3. Update baseline if changes are expected

## Next Steps

- [Reports](reports.md) - Understanding verification reports
- [Configuration](configuration.md) - Adjust check settings
- [Troubleshooting](troubleshooting.md) - Common issues
