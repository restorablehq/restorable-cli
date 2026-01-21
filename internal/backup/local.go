package backup

import (
	"context"
	"fmt"
	"io"
	"os"
)

// LocalSource implements BackupSource for local file paths.
type LocalSource struct {
	Path string
}

// Acquire opens the local file and returns it as a ReadCloser.
func (s *LocalSource) Acquire(ctx context.Context) (io.ReadCloser, error) {
	file, err := os.Open(s.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open local backup file at %s: %w", s.Path, err)
	}
	return file, nil
}

// Identifier returns the local file path for traceability.
func (s *LocalSource) Identifier() string {
	return fmt.Sprintf("local:%s", s.Path)
}
