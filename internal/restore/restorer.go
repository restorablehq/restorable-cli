package restore

import (
	"context"
	"io"

	"restorable.io/restorable-cli/internal/schema"
)

// Restorer defines the interface for database restore operations.
type Restorer interface {
	// Restore performs the database restore from a backup stream.
	Restore(ctx context.Context, backup io.Reader) error
	// ExtractSchema extracts the schema from the restored database.
	ExtractSchema(ctx context.Context) (*schema.Schema, error)
	// ExtractMetrics extracts metrics from the restored database.
	ExtractMetrics(ctx context.Context) (*schema.Metrics, error)
	// Cleanup terminates the ephemeral database container.
	Cleanup(ctx context.Context) error
}
