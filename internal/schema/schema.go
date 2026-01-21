package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Schema represents the database schema structure.
type Schema struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Tables    []Table   `json:"tables"`
}

// Table represents a database table's metadata.
type Table struct {
	Name        string   `json:"name"`
	Schema      string   `json:"schema"`
	ColumnCount int      `json:"column_count"`
	Columns     []Column `json:"columns,omitempty"`
}

// Column represents a database column's metadata.
type Column struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
}

// Metrics represents database metrics collected after restore.
type Metrics struct {
	Timestamp       time.Time       `json:"timestamp"`
	RestoreDuration time.Duration   `json:"restore_duration_ns"`
	DBSizeBytes     int64           `json:"db_size_bytes"`
	TableMetrics    []TableMetrics  `json:"table_metrics"`
}

// TableMetrics represents metrics for a single table.
type TableMetrics struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RowCount int64  `json:"row_count"`
}

// TableNames returns a list of fully qualified table names (schema.table).
func (s *Schema) TableNames() []string {
	names := make([]string, len(s.Tables))
	for i, t := range s.Tables {
		names[i] = fmt.Sprintf("%s.%s", t.Schema, t.Name)
	}
	return names
}

// BaselineStore handles persisting and loading baseline schemas.
type BaselineStore struct {
	basePath string
}

// NewBaselineStore creates a store for baseline schemas.
func NewBaselineStore() (*BaselineStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}
	basePath := filepath.Join(homeDir, ".restorable", "schemas")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create schemas directory: %w", err)
	}
	return &BaselineStore{basePath: basePath}, nil
}

// Save persists a schema as the baseline for a project.
func (s *BaselineStore) Save(projectID string, schema *Schema) error {
	path := filepath.Join(s.basePath, projectID+".json")
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}
	return nil
}

// Load retrieves the baseline schema for a project.
// Returns nil, nil if no baseline exists.
func (s *BaselineStore) Load(projectID string) (*Schema, error) {
	path := filepath.Join(s.basePath, projectID+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	return &schema, nil
}

// Exists checks if a baseline schema exists for a project.
func (s *BaselineStore) Exists(projectID string) bool {
	path := filepath.Join(s.basePath, projectID+".json")
	_, err := os.Stat(path)
	return err == nil
}
