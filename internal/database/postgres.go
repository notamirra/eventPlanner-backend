package database

import (
    "context"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "sort"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool creates a pgx connection pool with sane defaults.
func NewPostgresPool(databaseURL string) (*pgxpool.Pool, error) {
    config, err := pgxpool.ParseConfig(databaseURL)
    if err != nil {
        return nil, err
    }
    config.MaxConns = 10
    config.MinConns = 1
    config.MaxConnIdleTime = 5 * time.Minute
    config.HealthCheckPeriod = 30 * time.Second

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, err
    }

    // Ping to verify connection
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, err
    }

    // Run migrations automatically
    if err := runMigrations(pool); err != nil {
        return nil, fmt.Errorf("migration error: %w", err)
    }

    return pool, nil
}

// ---------------------------------------------------
// Migration Loader
// ---------------------------------------------------

func runMigrations(pool *pgxpool.Pool) error {
    migrationsPath := "./internal/database/migrations"

    files, err := ioutil.ReadDir(migrationsPath)
    if err != nil {
        return fmt.Errorf("cannot read migrations folder: %w", err)
    }

    // Sort alphabetically to ensure order
    sort.Slice(files, func(i, j int) bool {
        return files[i].Name() < files[j].Name()
    })

    ctx := context.Background()

    for _, file := range files {
        if filepath.Ext(file.Name()) != ".sql" {
            continue // ignore non-sql files
        }

        sqlPath := filepath.Join(migrationsPath, file.Name())

        sqlBytes, err := os.ReadFile(sqlPath)
        if err != nil {
            return fmt.Errorf("failed reading %s: %w", file.Name(), err)
        }

        fmt.Println("Running migration:", file.Name())

        _, err = pool.Exec(ctx, string(sqlBytes))
        if err != nil {
            return fmt.Errorf("migration %s failed: %w", file.Name(), err)
        }
    }

    fmt.Println("All migrations executed successfully.")
    return nil
}
