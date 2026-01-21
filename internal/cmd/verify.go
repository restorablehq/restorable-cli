package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"restorable.io/restorable-cli/internal/backup"
	"restorable.io/restorable-cli/internal/config"
	"restorable.io/restorable-cli/internal/crypto"
	"restorable.io/restorable-cli/internal/report"
	"restorable.io/restorable-cli/internal/restore"
	"restorable.io/restorable-cli/internal/schema"
	"restorable.io/restorable-cli/internal/verify"
)

var verbose bool

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verifies a backup can be restored",
	Long: `Runs the end-to-end verification process for a backup artifact.

This command performs the following steps:
1. Acquires the backup artifact from the configured source.
2. Decrypts the artifact (if configured).
3. Restores it into a temporary, isolated database instance.
4. Extracts schema and metrics from the restored database.
5. Performs integrity checks against the restored database.
6. Generates and signs a verification report.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		fmt.Println("Running verification...")

		// 1. Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		fmt.Println("✓ Configuration loaded.")

		// 2. Acquire backup artifact using BackupSource interface
		source, err := backup.NewSourceFromConfig(&cfg.Backup)
		if err != nil {
			return fmt.Errorf("failed to create backup source: %w", err)
		}

		fmt.Printf("Acquiring backup from source: %s\n", source.Identifier())
		backupStream, err := source.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("failed to acquire backup: %w", err)
		}
		defer backupStream.Close()
		fmt.Println("✓ Backup artifact acquired.")

		// 3. Decrypt (if configured)
		var dataStream io.ReadCloser = backupStream
		if cfg.Encryption != nil {
			fmt.Println("Decrypting backup...")
			decryptor, err := crypto.NewAgeDecryptor(cfg.Encryption.PrivateKeyPath)
			if err != nil {
				return fmt.Errorf("failed to create decryptor: %w", err)
			}
			decryptedStream, err := decryptor.NewDecryptReadCloser(backupStream)
			if err != nil {
				return fmt.Errorf("decryption failed: %w", err)
			}
			dataStream = decryptedStream
			fmt.Println("✓ Backup decrypted.")
		} else {
			fmt.Println("✓ Backup is not encrypted, skipping decryption.")
		}

		// 4. Start ephemeral DB container and restore backup
		var restorer restore.Restorer
		if cfg.Database.Type == "postgres" {
			restorer = restore.NewPostgresRestorer(cfg, verbose)
		} else {
			return fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
		}

		fmt.Println("Starting ephemeral DB container and running restore...")
		if err := restorer.Restore(ctx, dataStream); err != nil {
			return fmt.Errorf("restore process failed: %w", err)
		}
		defer restorer.Cleanup(context.Background())

		// 5. Extract schema and metrics
		fmt.Println("Extracting schema...")
		extractedSchema, err := restorer.ExtractSchema(ctx)
		if err != nil {
			return fmt.Errorf("failed to extract schema: %w", err)
		}
		fmt.Printf("✓ Schema extracted: %d tables found.\n", len(extractedSchema.Tables))

		fmt.Println("Extracting metrics...")
		metrics, err := restorer.ExtractMetrics(ctx)
		if err != nil {
			return fmt.Errorf("failed to extract metrics: %w", err)
		}
		fmt.Println("✓ Metrics extracted.")

		// 6. Load baseline schema (if exists)
		baselineStore, err := schema.NewBaselineStore()
		if err != nil {
			return fmt.Errorf("failed to create baseline store: %w", err)
		}

		baseline, err := baselineStore.Load(cfg.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to load baseline schema: %w", err)
		}

		if baseline == nil {
			fmt.Println("No baseline schema found. This will be stored as the baseline.")
		} else {
			fmt.Printf("✓ Baseline schema loaded (%d tables).\n", len(baseline.Tables))
		}

		// 7. Run verification checks
		fmt.Println("Running verification checks...")
		checkers := buildCheckers(cfg)
		checkResults := verify.RunChecks(ctx, checkers, extractedSchema, baseline, metrics)

		for _, r := range checkResults {
			status := "✓"
			if !r.Passed {
				status = "✗"
			}
			fmt.Printf("  %s [%s] %s: %s\n", status, r.Level, r.Name, r.Message)
		}

		critical, warning, _ := verify.CountFailures(checkResults)
		if critical > 0 {
			fmt.Printf("\n✗ Verification failed with %d critical failure(s).\n", critical)
		} else if warning > 0 {
			fmt.Printf("\n⚠ Verification passed with %d warning(s).\n", warning)
		} else {
			fmt.Println("\n✓ All verification checks passed.")
		}

		// 8. Generate report
		fmt.Println("\nGenerating report...")
		reportID := uuid.New().String()

		rpt := report.NewReportBuilder().
			WithID(reportID).
			WithProject(cfg.Project.ID, cfg.Project.Name).
			WithMachineID(cfg.CLI.MachineID).
			WithBackupSource(source.Identifier()).
			WithDatabase(cfg.Database.Type, cfg.Database.MajorVersion).
			WithSchema(extractedSchema).
			WithMetrics(metrics).
			WithChecks(checkResults).
			Build()

		// 9. Sign report
		privateKey, err := report.LoadPrivateKey(cfg.Signing.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load signing key: %w", err)
		}

		if err := report.Sign(rpt, privateKey); err != nil {
			return fmt.Errorf("failed to sign report: %w", err)
		}
		fmt.Println("✓ Report signed.")

		// 10. Write report
		reportPath, err := report.WriteJSON(rpt, cfg.CLI.ReportDir)
		if err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		fmt.Printf("✓ Report saved to %s\n", reportPath)

		// 11. Save schema as new baseline if this is the first run
		if baseline == nil {
			if err := baselineStore.Save(cfg.Project.ID, extractedSchema); err != nil {
				return fmt.Errorf("failed to save baseline schema: %w", err)
			}
			fmt.Println("✓ Schema saved as baseline for future comparisons.")
		}

		// Final summary
		fmt.Printf("\nVerification completed. Report ID: %s\n", reportID)
		if critical > 0 {
			return fmt.Errorf("verification failed with %d critical failure(s)", critical)
		}

		return nil
	},
}

func buildCheckers(cfg *config.Config) []verify.Checker {
	var checkers []verify.Checker

	// Always run table checks (critical)
	checkers = append(checkers, verify.NewTablesExistChecker())
	checkers = append(checkers, verify.NewTableCountChecker())
	checkers = append(checkers, verify.NewNewTablesChecker())

	// Row count checks (if enabled)
	if cfg.Verification.RowCounts.Enabled {
		checkers = append(checkers, verify.NewRowCountChecker(cfg.Verification.RowCounts.WarnThresholdPercent))
		checkers = append(checkers, verify.NewNonEmptyTablesChecker(1))
		checkers = append(checkers, verify.NewTotalRowCountChecker(1))
	}

	// Always track restore duration
	checkers = append(checkers, verify.NewRestoreDurationChecker(0))

	return checkers
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}
