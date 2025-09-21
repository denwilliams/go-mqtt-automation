package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/denwilliams/go-mqtt-automation/pkg/migration"
)

func main() {
	var (
		migrationsDir = flag.String("dir", "db/migrations", "Directory containing migration templates")
		dbType        = flag.String("db", "", "Database type (sqlite, postgres, mysql). If empty, generates for all types")
		templateFile  = flag.String("template", "", "Specific template file to process. If empty, processes all templates")
	)
	flag.Parse()

	if *templateFile != "" && *dbType != "" {
		// Process single template for single database
		outputPath := fmt.Sprintf("%s/%s/%s", *migrationsDir, *dbType,
			fmt.Sprintf("%s.sql", (*templateFile)[:len(*templateFile)-12])) // Remove .sql.template

		if err := migration.ProcessMigrationTemplate(*templateFile, *dbType, outputPath); err != nil {
			log.Fatalf("Failed to process template: %v", err)
		}
		fmt.Printf("Generated migration: %s\n", outputPath)
	} else {
		// Process all templates for all databases
		if err := migration.GenerateAllMigrations(*migrationsDir); err != nil {
			log.Fatalf("Failed to generate migrations: %v", err)
		}
		fmt.Println("Generated all database-specific migrations")
	}
}