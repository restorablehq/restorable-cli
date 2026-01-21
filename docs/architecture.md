# Architecture

This document describes the technical architecture of Restorable CLI for developers and contributors.

## Overview

Restorable CLI follows a modular architecture designed around a single responsibility: prove that database backups can be restored.

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Layer (cmd)                         │
│  init │ verify │ report │ version                          │
└─────────────────┬───────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────────┐
│                   Core Packages                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │  backup  │ │  crypto  │ │ restore  │ │  verify  │       │
│  │  source  │ │  (age)   │ │ (docker) │ │  checks  │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                    │
│  │  schema  │ │  report  │ │  config  │                    │
│  │ baseline │ │ signing  │ │  loader  │                    │
│  └──────────┘ └──────────┘ └──────────┘                    │
└─────────────────────────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────────┐
│                External Dependencies                        │
│  Docker │ PostgreSQL │ S3 │ Age │ Ed25519                  │
└─────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
restorable-cli/
├── cmd/
│   └── restorable/
│       └── main.go           # Entry point
├── internal/
│   ├── backup/               # Backup source abstraction
│   │   ├── source.go         # BackupSource interface
│   │   ├── local.go          # Local file source
│   │   ├── s3.go             # S3-compatible source
│   │   └── command.go        # Shell command source
│   ├── cmd/                  # CLI commands
│   │   ├── root.go           # Root command setup
│   │   ├── init.go           # Init command
│   │   ├── verify.go         # Verify command
│   │   ├── report.go         # Report commands
│   │   └── version.go        # Version command
│   ├── config/               # Configuration management
│   │   └── config.go         # Config loading and types
│   ├── crypto/               # Encryption handling
│   │   └── age.go            # Age decryption
│   ├── report/               # Report generation
│   │   ├── report.go         # Report structure and builder
│   │   └── sign.go           # Ed25519 signing
│   ├── restore/              # Database restoration
│   │   ├── restorer.go       # Restorer interface
│   │   └── postgres.go       # PostgreSQL implementation
│   ├── schema/               # Schema management
│   │   └── schema.go         # Schema types and baseline
│   ├── signing/              # Key generation
│   │   └── key.go            # Ed25519 key generation
│   └── verify/               # Verification checks
│       ├── checker.go        # Checker interface
│       ├── tables.go         # Table existence checks
│       └── rowcount.go       # Row count checks
├── go.mod
└── go.sum
```

## Core Components

### Backup Source Abstraction

The backup package provides a unified interface for fetching backups from different sources.

```go
type BackupSource interface {
    // Acquire fetches the backup and returns a reader
    Acquire(ctx context.Context) (io.ReadCloser, error)
    
    // Identifier returns a string identifying the source
    Identifier() string
}
```

**Implementations:**
- `LocalSource` - Reads from filesystem
- `S3Source` - Fetches from S3-compatible storage
- `CommandSource` - Executes shell command, captures stdout

### Restore Engine

The restore package handles database restoration in ephemeral containers.

```go
type Restorer interface {
    // Restore performs the backup restoration
    Restore(ctx context.Context, backup io.Reader) error
    
    // ExtractSchema captures the database schema
    ExtractSchema(ctx context.Context) (*schema.Schema, error)
    
    // ExtractMetrics captures database metrics
    ExtractMetrics(ctx context.Context) (*schema.Metrics, error)
    
    // Cleanup terminates the container
    Cleanup(ctx context.Context) error
}
```

**PostgresRestorer Flow:**
1. Start PostgreSQL container via testcontainers
2. Copy backup file into container
3. Attempt `pg_restore` (custom format)
4. Fallback to `psql` (plain SQL format)
5. Connect via database driver
6. Query information_schema for schema
7. Query pg_stat_user_tables for metrics

### Verification System

The verify package implements a checker pattern for validation.

```go
type Checker interface {
    Check(ctx context.Context, 
          current *schema.Schema, 
          baseline *schema.Schema, 
          metrics *schema.Metrics) CheckResult
}

type CheckResult struct {
    Name    string
    Level   Level  // critical, warning, info
    Passed  bool
    Message string
}
```

**Built-in Checkers:**
- `TablesExistChecker` - Verifies baseline tables exist
- `TableCountChecker` - Compares table counts
- `NewTablesChecker` - Reports new tables
- `RowCountChecker` - Validates row counts
- `NonEmptyTablesChecker` - Ensures tables have data
- `TotalRowCountChecker` - Checks total row count
- `RestoreDurationChecker` - Tracks restore time

### Report Generation

Reports are built using the builder pattern:

```go
report := report.NewBuilder().
    WithID(uuid.New().String()).
    WithProject(cfg.Project.ID, cfg.Project.Name).
    WithMachineID(cfg.CLI.MachineID).
    WithBackupSource(source.Identifier()).
    WithDatabase(dbType, dbVersion, dbSize).
    WithSchema(schema).
    WithMetrics(metrics).
    WithChecks(checks).
    Build()
```

### Cryptographic Signing

Reports are signed using Ed25519:

```go
// Sign a report
err := report.Sign(report, privateKey)

// Verify a report
valid := report.Verify(report, publicKey)
```

## Data Flow

### Verify Command Flow

```
1. Load Config
   └─> config.Load() -> Config

2. Create Backup Source
   └─> backup.NewSource(config) -> BackupSource

3. Acquire Backup
   └─> source.Acquire(ctx) -> io.ReadCloser

4. Decrypt (if configured)
   └─> crypto.Decrypt(reader, key) -> io.ReadCloser

5. Create Restorer
   └─> restore.NewPostgresRestorer(config) -> Restorer

6. Restore Backup
   └─> restorer.Restore(ctx, reader) -> error

7. Extract Schema
   └─> restorer.ExtractSchema(ctx) -> Schema

8. Extract Metrics
   └─> restorer.ExtractMetrics(ctx) -> Metrics

9. Load Baseline
   └─> schema.LoadBaseline(projectID) -> Schema?

10. Run Checks
    └─> verify.RunChecks(current, baseline, metrics) -> []CheckResult

11. Build Report
    └─> report.NewBuilder()...Build() -> Report

12. Sign Report
    └─> report.Sign(report, privateKey) -> error

13. Save Report
    └─> report.WriteJSON(report, dir) -> error

14. Update Baseline
    └─> schema.SaveBaseline(projectID, current) -> error

15. Cleanup
    └─> restorer.Cleanup(ctx) -> error
```

## Design Principles

### 1. Interface Segregation

Each component defines minimal interfaces:
- `BackupSource` - just `Acquire()` and `Identifier()`
- `Restorer` - restore, extract, cleanup
- `Checker` - single `Check()` method

### 2. Dependency Injection

Components receive dependencies via constructors:

```go
func NewPostgresRestorer(cfg *config.Config) (*PostgresRestorer, error)
func NewS3Source(cfg *config.S3Config) (*S3Source, error)
```

### 3. Context Propagation

All long-running operations accept `context.Context`:

```go
func (r *PostgresRestorer) Restore(ctx context.Context, backup io.Reader) error
```

### 4. Error Wrapping

Errors include context for debugging:

```go
return fmt.Errorf("failed to restore backup: %w", err)
```

### 5. Streaming

Backup data is streamed to minimize memory usage:

```go
type BackupSource interface {
    Acquire(ctx context.Context) (io.ReadCloser, error)
}
```

## External Dependencies

### Core Libraries

| Library | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/testcontainers/testcontainers-go` | Ephemeral containers |
| `filippo.io/age` | Modern encryption |
| `github.com/lib/pq` | PostgreSQL driver |
| `github.com/aws/aws-sdk-go-v2` | S3 operations |
| `github.com/google/uuid` | UUID generation |
| `gopkg.in/yaml.v3` | YAML parsing |

### Runtime Dependencies

- **Docker** - Container runtime for testcontainers
- **PostgreSQL** - Database for restoration (via container)

## Extension Points

### Adding a New Backup Source

1. Implement `BackupSource` interface in `internal/backup/`
2. Add source type to config
3. Update source factory in `verify.go`

```go
type MySource struct {
    config *config.MyConfig
}

func (s *MySource) Acquire(ctx context.Context) (io.ReadCloser, error) {
    // Implementation
}

func (s *MySource) Identifier() string {
    return "my-source://..."
}
```

### Adding a New Database Type

1. Implement `Restorer` interface in `internal/restore/`
2. Add database type to config
3. Update restorer factory

### Adding a New Checker

1. Implement `Checker` interface in `internal/verify/`
2. Add to checker list in `RunChecks()`

```go
type MyChecker struct{}

func (c *MyChecker) Check(ctx context.Context, 
    current, baseline *schema.Schema, 
    metrics *schema.Metrics) CheckResult {
    // Implementation
    return CheckResult{
        Name:    "my_check",
        Level:   verify.LevelWarning,
        Passed:  true,
        Message: "Check passed",
    }
}
```

## Security Considerations

### Key Storage

- Signing keys stored in `~/.restorable/keys/`
- File permissions should be 600 (owner read/write only)
- Keys never transmitted or logged

### Credential Handling

- Database passwords via environment variables
- S3 credentials via environment variables
- Never stored in config files

### Container Isolation

- Containers are ephemeral (created and destroyed)
- No persistent volumes mounted
- Network isolated (bridge mode)

### Report Integrity

- Reports signed with Ed25519
- Signature covers entire report content
- Tampering detectable via signature verification

## Testing

### Unit Tests

```bash
go test ./internal/...
```

### Integration Tests

Require Docker:

```bash
go test ./internal/restore/... -tags=integration
```

### Test Coverage

```bash
go test -cover ./...
```

## Build and Release

### Building

```bash
go build -o restorable ./cmd/restorable
```

### Cross-Compilation

```bash
GOOS=linux GOARCH=amd64 go build -o restorable-linux-amd64 ./cmd/restorable
GOOS=darwin GOARCH=arm64 go build -o restorable-darwin-arm64 ./cmd/restorable
```

### Version Embedding

```bash
go build -ldflags "-X main.Version=1.0.0" ./cmd/restorable
```

## Future Considerations

### Planned Features

- MySQL/MariaDB support
- MongoDB support
- Custom SQL validation checks
- Webhook notifications
- Testcontainers Cloud integration

### Architecture Evolution

- Plugin system for backup sources
- Plugin system for database types
- Remote report storage
- Distributed verification
