package db

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB, migrationsDir string) error {
    entries, err := os.ReadDir(migrationsDir)
    if err != nil {
        return err
    }
    for _, entry := range entries {
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
            continue
        }
        path := filepath.Join(migrationsDir, entry.Name())
        sqlBytes, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        if _, err := db.Exec(string(sqlBytes)); err != nil {
            return fmt.Errorf("migration %s failed: %w", entry.Name(), err)
        }
    }
    return nil
}
