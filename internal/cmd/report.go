package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"restorable.io/restorable-cli/internal/config"
	"restorable.io/restorable-cli/internal/report"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Manage verification reports",
	Long:  `List, view, and verify verification reports.`,
}

var reportListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all verification reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		reports, err := report.ListReports(cfg.CLI.ReportDir)
		if err != nil {
			return fmt.Errorf("failed to list reports: %w", err)
		}

		if len(reports) == 0 {
			fmt.Println("No reports found.")
			return nil
		}

		fmt.Printf("%-36s  %-20s  %-20s  %s\n", "ID", "Timestamp", "Project", "Status")
		fmt.Println(strings.Repeat("-", 100))

		for _, r := range reports {
			status := "✓ Success"
			if !r.Success {
				status = "✗ Failed"
			}
			fmt.Printf("%-36s  %-20s  %-20s  %s\n",
				r.ID,
				r.Timestamp.Format("2006-01-02 15:04:05"),
				r.ProjectID,
				status,
			)
		}

		return nil
	},
}

var reportShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display a verification report",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reportID := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		rpt, path, err := findReport(cfg.CLI.ReportDir, reportID)
		if err != nil {
			return err
		}

		showJSON, _ := cmd.Flags().GetBool("json")
		if showJSON {
			data, err := json.MarshalIndent(rpt, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		// Display human-readable report
		fmt.Printf("Report: %s\n", rpt.ID)
		fmt.Printf("Path: %s\n", path)
		fmt.Printf("Timestamp: %s\n", rpt.Timestamp.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("Project: %s (%s)\n", rpt.ProjectName, rpt.ProjectID)
		fmt.Printf("Machine: %s\n", rpt.MachineID)
		fmt.Printf("Backup Source: %s\n", rpt.BackupSource)
		fmt.Println()

		// Database info
		fmt.Printf("Database: %s %d\n", rpt.Database.Type, rpt.Database.MajorVersion)
		if rpt.Database.SizeBytes > 0 {
			fmt.Printf("Database Size: %s\n", formatBytes(rpt.Database.SizeBytes))
		}
		fmt.Println()

		// Summary
		fmt.Println("Summary:")
		if rpt.Summary.Success {
			fmt.Println("  Status: ✓ Success")
		} else {
			fmt.Println("  Status: ✗ Failed")
		}
		fmt.Printf("  Checks: %d/%d passed\n", rpt.Summary.PassedChecks, rpt.Summary.TotalChecks)
		if rpt.Summary.CriticalFailures > 0 {
			fmt.Printf("  Critical Failures: %d\n", rpt.Summary.CriticalFailures)
		}
		if rpt.Summary.WarningFailures > 0 {
			fmt.Printf("  Warnings: %d\n", rpt.Summary.WarningFailures)
		}
		if rpt.Summary.RestoreDuration != "" {
			fmt.Printf("  Restore Duration: %s\n", rpt.Summary.RestoreDuration)
		}
		fmt.Println()

		// Checks
		fmt.Println("Checks:")
		for _, c := range rpt.Checks {
			status := "✓"
			if !c.Passed {
				status = "✗"
			}
			fmt.Printf("  %s [%s] %s: %s\n", status, c.Level, c.Name, c.Message)
		}
		fmt.Println()

		// Signature
		if rpt.Signature != "" {
			fmt.Printf("Signature: %s...\n", rpt.Signature[:min(32, len(rpt.Signature))])
		} else {
			fmt.Println("Signature: (not signed)")
		}

		return nil
	},
}

var reportVerifyCmd = &cobra.Command{
	Use:   "verify <id>",
	Short: "Verify a report's signature",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reportID := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		rpt, _, err := findReport(cfg.CLI.ReportDir, reportID)
		if err != nil {
			return err
		}

		// Load public key
		pubKeyPath := strings.TrimSuffix(cfg.Signing.PrivateKeyPath, ".key") + ".pub"
		pubKey, err := report.LoadPublicKey(pubKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load public key: %w", err)
		}

		valid, err := report.Verify(rpt, pubKey)
		if err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}

		if valid {
			fmt.Println("✓ Signature is valid")
		} else {
			fmt.Println("✗ Signature is INVALID")
			os.Exit(1)
		}

		return nil
	},
}

func findReport(dir string, id string) (*report.Report, string, error) {
	reports, err := report.ListReports(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list reports: %w", err)
	}

	// Try exact match first
	for _, r := range reports {
		if r.ID == id {
			rpt, err := report.LoadReport(r.Path)
			return rpt, r.Path, err
		}
	}

	// Try prefix match
	var matches []*report.ReportSummary
	for _, r := range reports {
		if strings.HasPrefix(r.ID, id) {
			matches = append(matches, r)
		}
	}

	if len(matches) == 0 {
		// Try filename match
		pattern := filepath.Join(dir, "*"+id+"*.json")
		files, _ := filepath.Glob(pattern)
		if len(files) == 1 {
			rpt, err := report.LoadReport(files[0])
			return rpt, files[0], err
		}
		return nil, "", fmt.Errorf("report not found: %s", id)
	}

	if len(matches) > 1 {
		return nil, "", fmt.Errorf("ambiguous report ID %q matches %d reports", id, len(matches))
	}

	rpt, err := report.LoadReport(matches[0].Path)
	return rpt, matches[0].Path, err
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.AddCommand(reportListCmd)
	reportCmd.AddCommand(reportShowCmd)
	reportCmd.AddCommand(reportVerifyCmd)

	reportShowCmd.Flags().Bool("json", false, "Output report as JSON")
}
