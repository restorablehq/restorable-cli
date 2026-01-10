package cmd

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "restorable",
    Short: "Restore verification for database backups",
    Long: `Restorable verifies that your database backups can actually be restored.
It restores backups in isolation and produces signed verification reports.`,
}

func Execute() error {
    return rootCmd.Execute()
}

