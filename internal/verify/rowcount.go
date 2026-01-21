package verify

import (
	"context"
	"fmt"

	"restorable.io/restorable-cli/internal/schema"
)

// RowCountChecker verifies that row counts are within acceptable thresholds.
type RowCountChecker struct {
	// WarnThresholdPercent is the percentage decrease that triggers a warning.
	// For example, 20 means warn if row count dropped by more than 20%.
	WarnThresholdPercent int
}

func NewRowCountChecker(warnThreshold int) *RowCountChecker {
	return &RowCountChecker{WarnThresholdPercent: warnThreshold}
}

func (c *RowCountChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "row_counts",
		Level: LevelWarning,
	}

	if baseline == nil || metrics == nil {
		result.Passed = true
		result.Message = "No baseline available for row count comparison"
		return result
	}

	// We need baseline metrics to compare, but we only have baseline schema.
	// For now, this check requires stored baseline metrics which we don't have yet.
	// This checker will be enhanced when baseline metrics storage is implemented.
	result.Passed = true
	result.Message = fmt.Sprintf("Row count check skipped (baseline metrics not available). Current total rows: %d", c.totalRows(metrics))
	return result
}

func (c *RowCountChecker) totalRows(metrics *schema.Metrics) int64 {
	var total int64
	for _, tm := range metrics.TableMetrics {
		total += tm.RowCount
	}
	return total
}

// NonEmptyTablesChecker verifies that tables have at least some data.
type NonEmptyTablesChecker struct {
	// MinimumTables is the minimum number of tables that should have data.
	MinimumTables int
}

func NewNonEmptyTablesChecker(minimumTables int) *NonEmptyTablesChecker {
	return &NonEmptyTablesChecker{MinimumTables: minimumTables}
}

func (c *NonEmptyTablesChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "non_empty_tables",
		Level: LevelWarning,
	}

	if metrics == nil {
		result.Passed = false
		result.Message = "No metrics available to check table data"
		return result
	}

	var tablesWithData int
	var emptyTables []string
	for _, tm := range metrics.TableMetrics {
		if tm.RowCount > 0 {
			tablesWithData++
		} else {
			emptyTables = append(emptyTables, fmt.Sprintf("%s.%s", tm.Schema, tm.Name))
		}
	}

	if tablesWithData >= c.MinimumTables {
		result.Passed = true
		result.Message = fmt.Sprintf("%d/%d tables have data", tablesWithData, len(metrics.TableMetrics))
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("Only %d tables have data (minimum: %d)", tablesWithData, c.MinimumTables)
	}

	return result
}

// TotalRowCountChecker verifies that the database has a minimum total row count.
type TotalRowCountChecker struct {
	MinimumRows int64
}

func NewTotalRowCountChecker(minimumRows int64) *TotalRowCountChecker {
	return &TotalRowCountChecker{MinimumRows: minimumRows}
}

func (c *TotalRowCountChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "total_row_count",
		Level: LevelWarning,
	}

	if metrics == nil {
		result.Passed = false
		result.Message = "No metrics available"
		return result
	}

	var totalRows int64
	for _, tm := range metrics.TableMetrics {
		totalRows += tm.RowCount
	}

	if totalRows >= c.MinimumRows {
		result.Passed = true
		result.Message = fmt.Sprintf("Total row count: %d", totalRows)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("Total row count %d is below minimum %d", totalRows, c.MinimumRows)
	}

	return result
}

// RestoreDurationChecker verifies that the restore completed within an acceptable time.
type RestoreDurationChecker struct {
	// MaxDurationSeconds is the maximum acceptable restore duration.
	MaxDurationSeconds int
}

func NewRestoreDurationChecker(maxSeconds int) *RestoreDurationChecker {
	return &RestoreDurationChecker{MaxDurationSeconds: maxSeconds}
}

func (c *RestoreDurationChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "restore_duration",
		Level: LevelInfo,
	}

	if metrics == nil {
		result.Passed = true
		result.Message = "No metrics available"
		return result
	}

	durationSecs := int(metrics.RestoreDuration.Seconds())
	result.Passed = true
	result.Message = fmt.Sprintf("Restore completed in %d seconds", durationSecs)

	if c.MaxDurationSeconds > 0 && durationSecs > c.MaxDurationSeconds {
		result.Level = LevelWarning
		result.Passed = false
		result.Message = fmt.Sprintf("Restore took %d seconds (maximum: %d)", durationSecs, c.MaxDurationSeconds)
	}

	return result
}
