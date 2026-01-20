package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config matches the structure of the config.yaml file.
type Config struct {
	Version      int           `yaml:"version"`
	Project      Project       `yaml:"project"`
	CLI          CLI           `yaml:"cli"`
	Backup       Backup        `yaml:"backup"`
	Encryption   *Encryption   `yaml:"encryption,omitempty"`
	Database     Database      `yaml:"database"`
	Verification Verification  `yaml:"verification"`
	Docker       Docker        `yaml:"docker"`
	Signing      Signing       `yaml:"signing"`
}

type Project struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type CLI struct {
	MachineID string `yaml:"machine_id"`
	ReportDir string `yaml:"report_dir"`
	TempDir   string `yaml:"temp_dir"`
}

type Local struct {
	Path string `yaml:"path"`
}

type Command struct {
	Exec string `yaml:"exec"`
}

type Backup struct {
	Source        string   `yaml:"source"`
	Local         *Local   `yaml:"local,omitempty"`
	S3            *S3      `yaml:"s3,omitempty"`
	Command       *Command `yaml:"command,omitempty"`
	RetentionDays int      `yaml:"retention_days"`
}

type S3 struct {
	Endpoint     string `yaml:"endpoint"`
	Bucket       string `yaml:"bucket"`
	Region       string `yaml:"region"`
	AccessKeyEnv string `yaml:"access_key_env"`
	SecretKeyEnv string `yaml:"secret_key_env"`
	Prefix       string `yaml:"prefix"`
}

type Encryption struct {
	Method         string `yaml:"method"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

type Database struct {
	Type         string  `yaml:"type"`
	MajorVersion int     `yaml:"major_version"`
	Restore      Restore `yaml:"restore"`
}

type Restore struct {
	DockerImage string `yaml:"docker_image"`
	User        string `yaml:"user"`
	PasswordEnv string `yaml:"password_env"`
	DBName      string `yaml:"db_name"`
	Port        int    `yaml:"port"`
}

type Verification struct {
	Schema    SchemaVerification `yaml:"schema"`
	RowCounts RowCounts          `yaml:"row_counts"`
}

type SchemaVerification struct {
	Enabled bool `yaml:"enabled"`
}

type RowCounts struct {
	Enabled              bool `yaml:"enabled"`
	WarnThresholdPercent int  `yaml:"warn_threshold_percent"`
}

type Docker struct {
	Network        string `yaml:"network"`
	PullPolicy     string `yaml:"pull_policy"`
	TimeoutMinutes int    `yaml:"timeout_minutes"`
}

type Signing struct {
	PrivateKeyPath string `yaml:"private_key_path"`
}

// Load finds, reads, and parses the configuration file.
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".restorable", "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s. Please run 'restorable init'", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file at %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}
