package restore

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"restorable.io/restorable-cli/internal/config"
	"restorable.io/restorable-cli/internal/schema"
)

// PostgresRestorer handles the Docker and pg_restore logic for Postgres.
type PostgresRestorer struct {
	config          *config.Config
	verbose         bool
	container       *postgres.PostgresContainer
	db              *sql.DB
	restoreDuration time.Duration
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
	r.container = pgContainer

	fmt.Println("✓ Database container started.")

	// Create a temporary file on the host for the backup stream
	tmpFile, err := os.CreateTemp("", "restorable-backup-*.dump")
	if err != nil {
		return fmt.Errorf("failed to create temporary backup file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

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

	// Track restore duration
	restoreStart := time.Now()

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
		r.restoreDuration = time.Since(restoreStart)
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

		r.restoreDuration = time.Since(restoreStart)

		if r.verbose && len(psqlLogBytes) > 0 {
			fmt.Println("--- psql output ---")
			fmt.Println(string(psqlLogBytes))
			fmt.Println("-------------------------")
		}
		fmt.Println("✓ Database restore completed successfully with psql.")
	}

	// Establish database connection for queries
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	r.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	return nil
}

// ExtractSchema extracts the schema from the restored database.
func (r *PostgresRestorer) ExtractSchema(ctx context.Context) (*schema.Schema, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not established; call Restore first")
	}

	// Query tables from information_schema
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			table_schema,
			table_name,
			(SELECT COUNT(*) FROM information_schema.columns c
			 WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name) as column_count
		FROM information_schema.tables t
		WHERE table_schema NOT IN ('information_schema', 'pg_catalog')
		  AND table_type = 'BASE TABLE'
		ORDER BY table_schema, table_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var t schema.Table
		if err := rows.Scan(&t.Schema, &t.Name, &t.ColumnCount); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		// Get column details
		columns, err := r.getTableColumns(ctx, t.Schema, t.Name)
		if err != nil {
			return nil, err
		}
		t.Columns = columns

		tables = append(tables, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return &schema.Schema{
		Version:   "1",
		Timestamp: time.Now().UTC(),
		Tables:    tables,
	}, nil
}

func (r *PostgresRestorer) getTableColumns(ctx context.Context, schemaName, tableName string) ([]schema.Column, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var c schema.Column
		var nullable string
		if err := rows.Scan(&c.Name, &c.DataType, &nullable); err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}
		c.Nullable = nullable == "YES"
		columns = append(columns, c)
	}

	return columns, rows.Err()
}

// ExtractMetrics extracts metrics from the restored database.
func (r *PostgresRestorer) ExtractMetrics(ctx context.Context) (*schema.Metrics, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not established; call Restore first")
	}

	metrics := &schema.Metrics{
		Timestamp:       time.Now().UTC(),
		RestoreDuration: r.restoreDuration,
	}

	// Get database size
	var dbSize int64
	err := r.db.QueryRowContext(ctx, `SELECT pg_database_size(current_database())`).Scan(&dbSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}
	metrics.DBSizeBytes = dbSize

	// Get row counts for each table
	rows, err := r.db.QueryContext(ctx, `
		SELECT schemaname, relname, n_live_tup
		FROM pg_stat_user_tables
		ORDER BY schemaname, relname
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query table stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tm schema.TableMetrics
		if err := rows.Scan(&tm.Schema, &tm.Name, &tm.RowCount); err != nil {
			return nil, fmt.Errorf("failed to scan table metrics row: %w", err)
		}
		metrics.TableMetrics = append(metrics.TableMetrics, tm)
	}

	// pg_stat_user_tables may not have accurate counts after restore
	// Run ANALYZE and re-query for more accurate counts if needed
	if len(metrics.TableMetrics) == 0 || r.allZeroRowCounts(metrics.TableMetrics) {
		metrics.TableMetrics, err = r.getAccurateRowCounts(ctx)
		if err != nil {
			return nil, err
		}
	}

	return metrics, rows.Err()
}

func (r *PostgresRestorer) allZeroRowCounts(metrics []schema.TableMetrics) bool {
	for _, m := range metrics {
		if m.RowCount > 0 {
			return false
		}
	}
	return true
}

func (r *PostgresRestorer) getAccurateRowCounts(ctx context.Context) ([]schema.TableMetrics, error) {
	// First get list of tables
	rows, err := r.db.QueryContext(ctx, `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema NOT IN ('information_schema', 'pg_catalog')
		  AND table_type = 'BASE TABLE'
		ORDER BY table_schema, table_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables for row counts: %w", err)
	}
	defer rows.Close()

	var metrics []schema.TableMetrics
	type tableDef struct {
		schema string
		name   string
	}
	var tables []tableDef

	for rows.Next() {
		var t tableDef
		if err := rows.Scan(&t.schema, &t.name); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Count rows in each table
	for _, t := range tables {
		var count int64
		query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s"`, t.schema, t.name)
		if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("failed to count rows in %s.%s: %w", t.schema, t.name, err)
		}
		metrics = append(metrics, schema.TableMetrics{
			Schema:   t.schema,
			Name:     t.name,
			RowCount: count,
		})
	}

	return metrics, nil
}

// Cleanup terminates the ephemeral database container.
func (r *PostgresRestorer) Cleanup(ctx context.Context) error {
	if r.db != nil {
		r.db.Close()
		r.db = nil
	}
	if r.container != nil {
		if err := r.container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
		r.container = nil
	}
	return nil
}
