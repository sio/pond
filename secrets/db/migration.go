package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed schema/*.sql
var schema embed.FS

func applyAllMigrations(db *sql.DB) error {
	entries, err := schema.ReadDir("schema")
	if err != nil {
		return fmt.Errorf("failed to read embedded filesystem: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	latest := getLatestMigration(db)
	var start int
	for index, migration := range entries {
		if migration.Name() == latest {
			start = index + 1
			break
		}
	}
	for _, migration := range entries[start:] {
		name := migration.Name()
		code, err := schema.ReadFile(fmt.Sprintf("schema/%s", name))
		if err != nil {
			return fmt.Errorf("failed to read embedded file: %w", err)
		}
		err = applyMigration(db, name, string(code))
		if err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
	}
	return nil
}

func applyMigration(db *sql.DB, name, code string) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.Exec(code)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO migration(schema) values (?);", name)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func getLatestMigration(db *sql.DB) string {
	row := db.QueryRow(
		"SELECT schema FROM migration ORDER BY timestamp DESC, schema DESC LIMIT 1",
	)
	var version string
	err := row.Scan(&version)
	if err != nil {
		version = fmt.Sprintf("failed to detect latest migration: %v", err)
	}
	return version
}
