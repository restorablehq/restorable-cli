package verify

import (
	"context"
	"fmt"
	"strings"

	"restorable.io/restorable-cli/internal/schema"
)

// TablesExistChecker verifies that expected tables exist in the restored database.
type TablesExistChecker struct{}

func NewTablesExistChecker() *TablesExistChecker {
	return &TablesExistChecker{}
}

func (c *TablesExistChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "tables_exist",
		Level: LevelCritical,
	}

	// No baseline means this is the first run - auto-pass
	if baseline == nil {
		result.Passed = true
		result.Message = "No baseline schema available (first verification run)"
		return result
	}

	// Build set of current tables
	currentTables := make(map[string]bool)
	for _, t := range current.Tables {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
		currentTables[key] = true
	}

	// Check which baseline tables are missing
	var missingTables []string
	for _, t := range baseline.Tables {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
		if !currentTables[key] {
			missingTables = append(missingTables, key)
		}
	}

	if len(missingTables) > 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("Missing %d tables: %s", len(missingTables), strings.Join(missingTables, ", "))
	} else {
		result.Passed = true
		result.Message = fmt.Sprintf("All %d expected tables present", len(baseline.Tables))
	}

	return result
}

// TableCountChecker verifies that the number of tables matches the baseline.
type TableCountChecker struct{}

func NewTableCountChecker() *TableCountChecker {
	return &TableCountChecker{}
}

func (c *TableCountChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "table_count",
		Level: LevelWarning,
	}

	if baseline == nil {
		result.Passed = true
		result.Message = fmt.Sprintf("Found %d tables (no baseline for comparison)", len(current.Tables))
		return result
	}

	diff := len(current.Tables) - len(baseline.Tables)
	if diff == 0 {
		result.Passed = true
		result.Message = fmt.Sprintf("Table count matches baseline: %d tables", len(current.Tables))
	} else if diff > 0 {
		result.Passed = true // New tables are typically not a failure
		result.Message = fmt.Sprintf("Table count increased: %d tables (+%d from baseline)", len(current.Tables), diff)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("Table count decreased: %d tables (%d from baseline)", len(current.Tables), diff)
	}

	return result
}

// NewTablesChecker reports new tables that weren't in the baseline.
type NewTablesChecker struct{}

func NewNewTablesChecker() *NewTablesChecker {
	return &NewTablesChecker{}
}

func (c *NewTablesChecker) Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult {
	result := CheckResult{
		Name:  "new_tables",
		Level: LevelInfo,
	}

	if baseline == nil {
		result.Passed = true
		result.Message = "No baseline schema available"
		return result
	}

	// Build set of baseline tables
	baselineTables := make(map[string]bool)
	for _, t := range baseline.Tables {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
		baselineTables[key] = true
	}

	// Find new tables
	var newTables []string
	for _, t := range current.Tables {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Name)
		if !baselineTables[key] {
			newTables = append(newTables, key)
		}
	}

	result.Passed = true // New tables are informational, not a failure
	if len(newTables) > 0 {
		result.Message = fmt.Sprintf("Found %d new tables: %s", len(newTables), strings.Join(newTables, ", "))
	} else {
		result.Message = "No new tables detected"
	}

	return result
}
