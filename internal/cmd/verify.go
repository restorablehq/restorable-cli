package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"restorable.io/restorable-cli/internal/config"
	"restorable.io/restorable-cli/internal/restore"
)

var verbose bool

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verifies a backup can be restored",
	Long: `Runs the end-to-end verification process for a backup artifact.

This command performs the following steps:
1. Acquires the backup artifact from the configured source.
2. Decrypts the artifact.
3. Restores it into a temporary, isolated database instance.
4. Performs integrity checks against the restored database.
5. Generates and signs a verification report.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		fmt.Println("Running verification...")

		// 1. Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		fmt.Println("✓ Configuration loaded.")

		// 2. Acquire backup artifact
		var backupStream io.ReadCloser
		fmt.Printf("Acquiring backup from source: %s\n", cfg.Backup.Source)

		switch cfg.Backup.Source {
		case "local":
			if cfg.Backup.Local == nil || cfg.Backup.Local.Path == "" {
				return fmt.Errorf("backup source is 'local' but path is not configured")
			}
			file, err := os.Open(cfg.Backup.Local.Path)
			if err != nil {
				return fmt.Errorf("failed to open local backup file at %s: %w", cfg.Backup.Local.Path, err)
			}
			backupStream = file
			defer backupStream.Close()
		default:
			return fmt.Errorf("unsupported backup source type: %s. Only 'local' is implemented", cfg.Backup.Source)
		}
		fmt.Println("✓ Backup artifact acquired.")

		// 3. Decrypt (if configured)
		if cfg.Encryption != nil {
			fmt.Println("✗ Decryption is configured but not yet implemented.")
			return fmt.Errorf("decryption not implemented")
		} else {
			fmt.Println("✓ Backup is not encrypted, skipping decryption.")
		}

		// 4. Start ephemeral DB container and restore backup
		if cfg.Database.Type == "postgres" {
			restorer := restore.NewPostgresRestorer(cfg, verbose)
			fmt.Println("Starting ephemeral DB container and running restore...")
			if err := restorer.Restore(ctx, backupStream); err != nil {
				return fmt.Errorf("restore process failed: %w", err)
			}
		} else {
			return fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
		}

		// Placeholder for subsequent steps
		fmt.Println("TODO: Extract schema + metrics")
		fmt.Println("TODO: Evaluate checks")
		fmt.Println("TODO: Emit report")

		fmt.Println("\nVerification check completed (partially implemented).")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}
