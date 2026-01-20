package restore

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"restorable.io/restorable-cli/internal/config"
)

// PostgresRestorer handles the Docker and pg_restore logic for Postgres.
type PostgresRestorer struct {
	config  *config.Config
	verbose bool
}

// NewPostgresRestorer creates a new restorer instance.
func NewPostgresRestorer(cfg *config.Config, verbose bool) *PostgresRestorer {
	return &PostgresRestorer{config: cfg, verbose: verbose}
}

// Restore performs the end-to-end restore process in an ephemeral container.
func (r *PostgresRestorer) Restore(ctx context.Context, backupStream io.Reader) error {
	dbPassword, ok := os.LookupEnv(r.config.Database.Restore.PasswordEnv)
	if !ok {
		return fmt.Errorf("database password environment variable %s not set", r.config.Database.Restore.PasswordEnv)
	}

	waitStrategy := wait.ForLog("database system is ready to accept connections").
		WithOccurrence(2).
		WithStartupTimeout(5 * time.Minute)

	pgContainer, err := postgres.Run(ctx,
		r.config.Database.Restore.DockerImage,
		postgres.WithDatabase(r.config.Database.Restore.DBName),
		postgres.WithUsername(r.config.Database.Restore.User),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(waitStrategy),
	)
	if err != nil {
		return fmt.Errorf("could not start postgres container: %w", err)
	}
	// Use a background context for termination to ensure it runs even if the parent context is cancelled.
	defer pgContainer.Terminate(context.Background())

        fmt.Println("✓ Database container started.")

        // Create a temporary file on the host for the backup stream
        tmpFile, err := os.CreateTemp("", "restorable-backup-*.dump")
        if err != nil {
            return fmt.Errorf("failed to create temporary backup file: %w", err)
        }
        defer os.Remove(tmpFile.Name()) // Clean up

        // Write the stream to the temporary file
        _, err = io.Copy(tmpFile, backupStream)
        if err != nil {
            return fmt.Errorf("failed to write backup to temporary file: %w", err)
        }
        tmpFile.Close()

        // Copy the temporary file to the container
        containerBackupPath := "/tmp/backup.dump"
        err = pgContainer.CopyFileToContainer(ctx, tmpFile.Name(), containerBackupPath, 0644)
        if err != nil {
            return fmt.Errorf("failed to copy backup file into container: %w", err)
        }

	// --- Attempt 1: pg_restore (for custom format) ---
	fmt.Println("Attempting restore with pg_restore...")
	pgRestoreCmd := []string{
		"pg_restore",
		"--username", r.config.Database.Restore.User,
		"--dbname", r.config.Database.Restore.DBName,
		"--no-password",
		"--verbose",
                "--no-owner",
		containerBackupPath,
	}

	pgRestoreExitCode, pgRestoreLogs, err := pgContainer.Exec(ctx, pgRestoreCmd)
	if err != nil {
		return fmt.Errorf("failed to execute pg_restore: %w", err)
	}

	pgRestoreLogBytes, _ := io.ReadAll(pgRestoreLogs)

	if pgRestoreExitCode == 0 {
		if r.verbose && len(pgRestoreLogBytes) > 0 {
			fmt.Println("--- pg_restore output ---")
			fmt.Println(string(pgRestoreLogBytes))
			fmt.Println("-------------------------")
		}
		fmt.Println("✓ Database restore completed successfully with pg_restore.")
	} else {
		// --- Attempt 2: psql (for plain text format) ---
		fmt.Println("pg_restore failed, attempting restore with psql...")
		if r.verbose {
			fmt.Println("--- pg_restore failure logs ---")
			fmt.Println(string(pgRestoreLogBytes))
			fmt.Println("-----------------------------")
		}

		psqlCmd := []string{
			"psql",
			"--username", r.config.Database.Restore.User,
			"--dbname", r.config.Database.Restore.DBName,
			"--no-password",
			"--file", containerBackupPath,
		}

		psqlExitCode, psqlLogs, err := pgContainer.Exec(ctx, psqlCmd)
		if err != nil {
			return fmt.Errorf("failed to execute psql: %w", err)
		}

		psqlLogBytes, _ := io.ReadAll(psqlLogs)

		if psqlExitCode != 0 {
			return fmt.Errorf("all restore methods failed.\n\npg_restore (exit %d):\n%s\n\npsql (exit %d):\n%s",
				pgRestoreExitCode, string(pgRestoreLogBytes),
				psqlExitCode, string(psqlLogBytes))
		}

		if r.verbose && len(psqlLogBytes) > 0 {
			fmt.Println("--- psql output ---")
			fmt.Println(string(psqlLogBytes))
			fmt.Println("-------------------------")
		}
		fmt.Println("✓ Database restore completed successfully with psql.")
	}

	// Placeholder for next steps
	fmt.Println("TODO: Extract schema from restored database.")

	return nil
}
