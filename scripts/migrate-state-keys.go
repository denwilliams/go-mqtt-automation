package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// This script migrates state keys from the old format (topic:name) to the new format
// (external:name, internal:name, child:name, system:name)

type topicConfig struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	StrategyID string `json:"strategy_id"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate-state-keys.go <database-path>")
		fmt.Println("Example: go run migrate-state-keys.go automation.db")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Load all configured topics
	configuredTopics := make(map[string]topicConfig)
	rows, err := db.Query("SELECT name, type, strategy_id FROM topics")
	if err != nil {
		log.Fatalf("Failed to query topics: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cfg topicConfig
		var strategyID sql.NullString
		if scanErr := rows.Scan(&cfg.Name, &cfg.Type, &strategyID); scanErr != nil {
			log.Printf("Failed to scan topic row: %v", scanErr)
			continue
		}
		if strategyID.Valid {
			cfg.StrategyID = strategyID.String
		}
		configuredTopics[cfg.Name] = cfg
	}

	log.Printf("Loaded %d configured topics", len(configuredTopics))

	// Load all state keys
	stateRows, err := db.Query("SELECT key, value FROM state WHERE key LIKE 'topic:%'")
	if err != nil {
		log.Fatalf("Failed to query state: %v", err)
	}
	defer stateRows.Close()

	updates := make(map[string]string) // old key -> new key
	for stateRows.Next() {
		var key string
		var value string
		if scanErr := stateRows.Scan(&key, &value); scanErr != nil {
			log.Printf("Failed to scan state row: %v", scanErr)
			continue
		}

		if !strings.HasPrefix(key, "topic:") {
			continue
		}

		topicName := strings.TrimPrefix(key, "topic:")
		var newKey string

		// Determine new key based on topic type
		if cfg, exists := configuredTopics[topicName]; exists {
			// This is a configured topic
			switch cfg.Type {
			case "system":
				newKey = "system:" + topicName
			case "internal":
				if cfg.StrategyID == "" {
					// Internal topic with no strategy = child topic
					newKey = "child:" + topicName
				} else {
					newKey = "internal:" + topicName
				}
			default:
				// Default to internal if type is unclear
				newKey = "internal:" + topicName
			}
		} else {
			// Not a configured topic - check if it's a child topic
			isChild := false
			for configuredName := range configuredTopics {
				if strings.HasPrefix(topicName, configuredName+"/") {
					isChild = true
					break
				}
			}

			if isChild {
				newKey = "child:" + topicName
			} else {
				newKey = "external:" + topicName
			}
		}

		if newKey != key {
			updates[key] = newKey
		}
	}

	log.Printf("Found %d state keys to migrate", len(updates))

	// Perform updates in a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	migratedCount := 0
	for oldKey, newKey := range updates {
		// Update the key
		result, err := tx.Exec("UPDATE state SET key = ? WHERE key = ?", newKey, oldKey)
		if err != nil {
			tx.Rollback()
			log.Fatalf("Failed to update state key %s -> %s: %v", oldKey, newKey, err)
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			log.Printf("Migrated: %s -> %s", oldKey, newKey)
			migratedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Printf("Successfully migrated %d state keys", migratedCount)
}
