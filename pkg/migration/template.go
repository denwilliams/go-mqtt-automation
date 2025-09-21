package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type DatabaseConfig struct {
	TextType            string
	IntType             string
	BoolType            string
	TimestampType       string
	CurrentTimestamp    string
	AutoIncrementType   string
	AutoIncrementSuffix string
}

var DatabaseConfigs = map[string]DatabaseConfig{
	"sqlite": {
		TextType:            "TEXT",
		IntType:             "INTEGER",
		BoolType:            "BOOLEAN",
		TimestampType:       "TIMESTAMP",
		CurrentTimestamp:    "CURRENT_TIMESTAMP",
		AutoIncrementType:   "INTEGER",
		AutoIncrementSuffix: " AUTOINCREMENT",
	},
	"postgres": {
		TextType:            "TEXT",
		IntType:             "INTEGER",
		BoolType:            "BOOLEAN",
		TimestampType:       "TIMESTAMP",
		CurrentTimestamp:    "CURRENT_TIMESTAMP",
		AutoIncrementType:   "SERIAL",
		AutoIncrementSuffix: "",
	},
	"mysql": {
		TextType:            "TEXT",
		IntType:             "INT",
		BoolType:            "BOOLEAN",
		TimestampType:       "TIMESTAMP",
		CurrentTimestamp:    "CURRENT_TIMESTAMP",
		AutoIncrementType:   "INT",
		AutoIncrementSuffix: " AUTO_INCREMENT",
	},
}

// ProcessMigrationTemplate reads a .sql.template file and generates database-specific SQL
func ProcessMigrationTemplate(templatePath, dbType, outputPath string) error {
	config, exists := DatabaseConfigs[dbType]
	if !exists {
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Read template file
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template
	tmpl, err := template.New("migration").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Execute template
	if err := tmpl.Execute(outputFile, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// GenerateAllMigrations processes all .sql.template files for all supported databases
func GenerateAllMigrations(migrationsDir string) error {
	templatePattern := filepath.Join(migrationsDir, "*.sql.template")
	templateFiles, err := filepath.Glob(templatePattern)
	if err != nil {
		return fmt.Errorf("failed to find template files: %w", err)
	}

	for _, templateFile := range templateFiles {
		baseName := strings.TrimSuffix(filepath.Base(templateFile), ".template")

		for dbType := range DatabaseConfigs {
			outputDir := filepath.Join(migrationsDir, dbType)
			outputPath := filepath.Join(outputDir, baseName)

			if err := ProcessMigrationTemplate(templateFile, dbType, outputPath); err != nil {
				return fmt.Errorf("failed to process template %s for %s: %w", templateFile, dbType, err)
			}
		}
	}

	return nil
}
