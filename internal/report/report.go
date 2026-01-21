package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"restorable.io/restorable-cli/internal/schema"
	"restorable.io/restorable-cli/internal/verify"
)

// ReportVersion is the current report format version.
const ReportVersion = "1"

// Report represents a verification report.
type Report struct {
	Version      string              `json:"version"`
	ID           string              `json:"id"`
	Timestamp    time.Time           `json:"timestamp"`
	ProjectID    string              `json:"project_id"`
	ProjectName  string              `json:"project_name"`
	MachineID    string              `json:"machine_id"`
	BackupSource string              `json:"backup_source"`
	Database     DatabaseInfo        `json:"database"`
	Schema       *schema.Schema      `json:"schema,omitempty"`
	Metrics      *schema.Metrics     `json:"metrics,omitempty"`
	Checks       []verify.CheckResult `json:"checks"`
	Summary      Summary             `json:"summary"`
	Signature    string              `json:"signature,omitempty"`
}

// DatabaseInfo contains database-related metadata.
type DatabaseInfo struct {
	Type         string `json:"type"`
	MajorVersion int    `json:"major_version"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
}

// Summary provides an overview of the verification result.
type Summary struct {
	Success          bool   `json:"success"`
	TotalChecks      int    `json:"total_checks"`
	PassedChecks     int    `json:"passed_checks"`
	FailedChecks     int    `json:"failed_checks"`
	CriticalFailures int    `json:"critical_failures"`
	WarningFailures  int    `json:"warning_failures"`
	RestoreDuration  string `json:"restore_duration"`
}

// ReportBuilder helps construct reports.
type ReportBuilder struct {
	report *Report
}

// NewReportBuilder creates a new report builder.
func NewReportBuilder() *ReportBuilder {
	return &ReportBuilder{
		report: &Report{
			Version:   ReportVersion,
			Timestamp: time.Now().UTC(),
		},
	}
}

func (b *ReportBuilder) WithID(id string) *ReportBuilder {
	b.report.ID = id
	return b
}

func (b *ReportBuilder) WithProject(id, name string) *ReportBuilder {
	b.report.ProjectID = id
	b.report.ProjectName = name
	return b
}

func (b *ReportBuilder) WithMachineID(machineID string) *ReportBuilder {
	b.report.MachineID = machineID
	return b
}

func (b *ReportBuilder) WithBackupSource(source string) *ReportBuilder {
	b.report.BackupSource = source
	return b
}

func (b *ReportBuilder) WithDatabase(dbType string, majorVersion int) *ReportBuilder {
	b.report.Database = DatabaseInfo{
		Type:         dbType,
		MajorVersion: majorVersion,
	}
	return b
}

func (b *ReportBuilder) WithSchema(s *schema.Schema) *ReportBuilder {
	b.report.Schema = s
	return b
}

func (b *ReportBuilder) WithMetrics(m *schema.Metrics) *ReportBuilder {
	b.report.Metrics = m
	if m != nil {
		b.report.Database.SizeBytes = m.DBSizeBytes
	}
	return b
}

func (b *ReportBuilder) WithChecks(checks []verify.CheckResult) *ReportBuilder {
	b.report.Checks = checks
	return b
}

// Build finalizes the report and computes the summary.
func (b *ReportBuilder) Build() *Report {
	b.computeSummary()
	return b.report
}

func (b *ReportBuilder) computeSummary() {
	total := len(b.report.Checks)
	var passed, failed, critical, warning int

	for _, c := range b.report.Checks {
		if c.Passed {
			passed++
		} else {
			failed++
			switch c.Level {
			case verify.LevelCritical:
				critical++
			case verify.LevelWarning:
				warning++
			}
		}
	}

	b.report.Summary = Summary{
		Success:          critical == 0,
		TotalChecks:      total,
		PassedChecks:     passed,
		FailedChecks:     failed,
		CriticalFailures: critical,
		WarningFailures:  warning,
	}

	if b.report.Metrics != nil {
		b.report.Summary.RestoreDuration = b.report.Metrics.RestoreDuration.String()
	}
}

// WriteJSON writes the report to a JSON file.
func WriteJSON(report *Report, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create report directory: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.json", report.Timestamp.Format("20060102_150405"), report.ID)
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write report file: %w", err)
	}

	return path, nil
}

// LoadReport loads a report from a JSON file.
func LoadReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}

	return &report, nil
}

// ListReports returns all reports in the given directory, sorted by timestamp (newest first).
func ListReports(dir string) ([]*ReportSummary, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read reports directory: %w", err)
	}

	var reports []*ReportSummary
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		report, err := LoadReport(path)
		if err != nil {
			continue // Skip invalid reports
		}

		reports = append(reports, &ReportSummary{
			ID:        report.ID,
			Timestamp: report.Timestamp,
			ProjectID: report.ProjectID,
			Success:   report.Summary.Success,
			Path:      path,
		})
	}

	// Sort by timestamp, newest first
	for i := 0; i < len(reports)-1; i++ {
		for j := i + 1; j < len(reports); j++ {
			if reports[j].Timestamp.After(reports[i].Timestamp) {
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}

	return reports, nil
}

// ReportSummary is a lightweight summary for listing reports.
type ReportSummary struct {
	ID        string
	Timestamp time.Time
	ProjectID string
	Success   bool
	Path      string
}
