package backup

import (
	"context"
	"fmt"
	"io"

	"restorable.io/restorable-cli/internal/config"
)

// BackupSource defines the interface for acquiring backup artifacts.
type BackupSource interface {
	// Acquire retrieves the backup artifact and returns a stream.
	Acquire(ctx context.Context) (io.ReadCloser, error)
	// Identifier returns a string identifying this backup source for report traceability.
	Identifier() string
}

// NewSourceFromConfig creates the appropriate BackupSource based on configuration.
func NewSourceFromConfig(cfg *config.Backup) (BackupSource, error) {
	switch cfg.Source {
	case "local":
		if cfg.Local == nil || cfg.Local.Path == "" {
			return nil, fmt.Errorf("backup source is 'local' but path is not configured")
		}
		return &LocalSource{Path: cfg.Local.Path}, nil

	case "s3":
		if cfg.S3 == nil {
			return nil, fmt.Errorf("backup source is 's3' but s3 configuration is missing")
		}
		return NewS3Source(cfg.S3)

	case "command":
		if cfg.Command == nil || cfg.Command.Exec == "" {
			return nil, fmt.Errorf("backup source is 'command' but exec is not configured")
		}
		return &CommandSource{Exec: cfg.Command.Exec}, nil

	default:
		return nil, fmt.Errorf("unsupported backup source type: %s", cfg.Source)
	}
}
