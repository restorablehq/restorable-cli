package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"restorable.io/restorable-cli/internal/config"
	"restorable.io/restorable-cli/internal/signing"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap config and keys for a new project",
	Long: `Initializes a new Restorable project in the current directory.

This command creates a '.restorable' directory containing a default 'config.yaml'
and a new Ed25519 keypair for signing verification reports. It will prompt
for basic project information to get you started.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Bootstrapping a new Restorable project...")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not get user home directory: %w", err)
		}
		baseDir := filepath.Join(homeDir, ".restorable")

		// Create directories
		if err := os.MkdirAll(filepath.Join(baseDir, "keys"), 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", baseDir, err)
		}

		// Check for existing config
		configPath := filepath.Join(baseDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("a config file already exists at %s", configPath)
		}

		reader := bufio.NewReader(os.Stdin)

		// Interactive prompts
		projectName, err := promptString(reader, "Project name")
		if err != nil {
			return err
		}
		dbType, err := promptWithDefault(reader, "Database type", "postgres")
		if err != nil {
			return err
		}
		dbVersion, err := promptIntWithDefault(reader, "Database major version", 15)
		if err != nil {
			return err
		}

		// Backup source configuration
		backupSource, err := promptWithDefault(reader, "Backup source type (local/s3/command)", "local")
		if err != nil {
			return err
		}
		var backupCfg config.Backup
		backupCfg.Source = backupSource
		backupCfg.RetentionDays = 30 // Default

		switch backupSource {
		case "local":
			path, err := promptString(reader, "Path to backup artifact")
			if err != nil {
				return err
			}
			backupCfg.Local = &config.Local{Path: path}
		case "s3":
			prefix, err := promptString(reader, "S3 key prefix (optional)")
			if err != nil {
				return err
			}
			backupCfg.S3 = &config.S3{
				Endpoint:     "https://s3.eu-central-1.example",
				Bucket:       "restorable-backups",
				Region:       "eu-central-1",
				AccessKeyEnv: "RESTORABLE_S3_KEY",
				SecretKeyEnv: "RESTORABLE_S3_SECRET",
				Prefix:       prefix,
			}
		case "command":
			exec, err := promptString(reader, "Command to fetch backup artifact")
			if err != nil {
				return err
			}
			backupCfg.Command = &config.Command{Exec: exec}
		default:
			return fmt.Errorf("unsupported backup source: %s", backupSource)
		}

		// Encryption configuration
		var encryptionCfg *config.Encryption
		useEncryption, err := promptWithDefault(reader, "Is the backup encrypted? (yes/no)", "yes")
		if err != nil {
			return err
		}
		if strings.ToLower(useEncryption) == "yes" {
			defaultKeyPath := filepath.Join(baseDir, "keys", "backup.key")
			keyPath, err := promptWithDefault(reader, "Path to encryption private key", defaultKeyPath)
			if err != nil {
				return err
			}
			encryptionCfg = &config.Encryption{
				Method:         "age",
				PrivateKeyPath: keyPath,
			}
		}

		// Generate signing keys
		pubKey, privKey, err := signing.GenerateSigningKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate signing key pair: %w", err)
		}

		privKeyPath := filepath.Join(baseDir, "keys", "signing.key")
		pubKeyPath := filepath.Join(baseDir, "keys", "signing.pub")

		// Build default config
		projectID := strings.ToLower(strings.ReplaceAll(projectName, " ", "_"))
		cfg := config.Config{
			Version: 1,
			Project: config.Project{
				ID:   projectID,
				Name: projectName,
			},
			CLI: config.CLI{
				MachineID: "db-verify-01",
				ReportDir: filepath.Join(baseDir, "reports"),
				TempDir:   "/tmp/restorable",
			},
			Backup:     backupCfg,
			Encryption: encryptionCfg,
			Database: config.Database{
				Type:         dbType,
				MajorVersion: dbVersion,
				Restore: config.Restore{
					DockerImage: fmt.Sprintf("%s:%d", dbType, dbVersion),
					User:        "postgres",
					PasswordEnv: "RESTORABLE_DB_PASSWORD",
					DBName:      "restorable_verify",
					Port:        5432,
				},
			},
			Verification: config.Verification{
				Schema: config.SchemaVerification{Enabled: true},
				RowCounts: config.RowCounts{
					Enabled:              true,
					WarnThresholdPercent: 5,
				},
			},
			Docker: config.Docker{
				Network:        "bridge",
				PullPolicy:     "if-not-present",
				TimeoutMinutes: 30,
			},
			Signing: config.Signing{
				PrivateKeyPath: privKeyPath,
			},
		}

		// Marshal and write config
		yamlData, err := yaml.Marshal(&cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config to YAML: %w", err)
		}

		if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		fmt.Printf("✓ Wrote config to %s\n", configPath)

		// Write keys
		if err := os.WriteFile(privKeyPath, privKey, 0600); err != nil {
			return fmt.Errorf("failed to write private key: %w", err)
		}
		if err := os.WriteFile(pubKeyPath, pubKey, 0644); err != nil {
			return fmt.Errorf("failed to write public key: %w", err)
		}
		fmt.Printf("✓ Wrote signing keys to %s and %s\n", privKeyPath, pubKeyPath)
		fmt.Println("\nProject initialized. Please review config.yaml and provide secrets via environment variables.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// promptString asks the user for input without a default value.
func promptString(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// promptWithDefault asks the user for input, providing a default if input is empty.
func promptWithDefault(reader *bufio.Reader, label, defaultValue string) (string, error) {
	fmt.Printf("%s (%s): ", label, defaultValue)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// promptIntWithDefault is a convenience wrapper for integer prompts.
func promptIntWithDefault(reader *bufio.Reader, label string, defaultValue int) (int, error) {
	valStr, err := promptWithDefault(reader, label, strconv.Itoa(defaultValue))
	if err != nil {
		return 0, err
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number provided: %q", valStr)
	}
	return val, nil
}
