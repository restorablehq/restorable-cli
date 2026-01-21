package verify

import (
	"context"

	"restorable.io/restorable-cli/internal/schema"
)

// Level indicates the severity of a check.
type Level string

const (
	LevelCritical Level = "critical" // Failures are blocking
	LevelWarning  Level = "warning"  // Failures are concerning but not blocking
	LevelInfo     Level = "info"     // Informational only
)

// CheckResult represents the outcome of a verification check.
type CheckResult struct {
	Name    string `json:"name"`
	Level   Level  `json:"level"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Checker defines the interface for verification checks.
type Checker interface {
	// Check performs the verification and returns the result.
	Check(ctx context.Context, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) CheckResult
}

// RunChecks executes a list of checkers and returns all results.
func RunChecks(ctx context.Context, checkers []Checker, current *schema.Schema, baseline *schema.Schema, metrics *schema.Metrics) []CheckResult {
	results := make([]CheckResult, 0, len(checkers))
	for _, c := range checkers {
		results = append(results, c.Check(ctx, current, baseline, metrics))
	}
	return results
}

// HasCriticalFailure returns true if any critical check failed.
func HasCriticalFailure(results []CheckResult) bool {
	for _, r := range results {
		if r.Level == LevelCritical && !r.Passed {
			return true
		}
	}
	return false
}

// CountFailures returns the number of failed checks by level.
func CountFailures(results []CheckResult) (critical, warning, info int) {
	for _, r := range results {
		if !r.Passed {
			switch r.Level {
			case LevelCritical:
				critical++
			case LevelWarning:
				warning++
			case LevelInfo:
				info++
			}
		}
	}
	return
}
