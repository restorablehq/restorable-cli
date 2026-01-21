package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

const defaultCommandTimeout = 10 * time.Minute

// CommandSource implements BackupSource by executing a shell command.
type CommandSource struct {
	Exec    string
	Timeout time.Duration
}

// commandReadCloser wraps a bytes.Reader to implement io.ReadCloser.
type commandReadCloser struct {
	*bytes.Reader
}

func (c *commandReadCloser) Close() error {
	return nil
}

// Acquire executes the command and returns its stdout as a ReadCloser.
func (s *CommandSource) Acquire(ctx context.Context) (io.ReadCloser, error) {
	timeout := s.Timeout
	if timeout == 0 {
		timeout = defaultCommandTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", s.Exec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command timed out after %v: %s", timeout, s.Exec)
		}
		return nil, fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
	}

	return &commandReadCloser{Reader: bytes.NewReader(stdout.Bytes())}, nil
}

// Identifier returns the command for traceability.
func (s *CommandSource) Identifier() string {
	return fmt.Sprintf("command:%s", s.Exec)
}
