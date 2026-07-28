package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

type DuckDB struct {
	DB     *sql.DB
	DBPath string
	mu     sync.RWMutex
}

func NewDuckDB(dbPath string) (*DuckDB, error) {
	database, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory duckdb: %w", err)
	}

	if dbPath != "" && dbPath != ":memory:" {
		log.Printf("🚀 Loading DuckDB dataset into RAM from %s...", dbPath)
		start := time.Now()

		attachSQL := fmt.Sprintf("ATTACH '%s' AS disk_db (READ_ONLY);", dbPath)
		if _, err := database.Exec(attachSQL); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to attach disk database at %s: %w", dbPath, err)
		}

		loadSQL := `
			CREATE TABLE IF NOT EXISTS voters AS SELECT * FROM disk_db.voters;
			CREATE TABLE IF NOT EXISTS polling_stations AS SELECT * FROM disk_db.polling_stations;
			DETACH disk_db;
		`
		if _, err := database.Exec(loadSQL); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to load dataset into RAM: %w", err)
		}

		log.Printf("✨ DuckDB dataset (1.04M voters) successfully loaded into RAM in %v", time.Since(start))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to ping duckdb database: %w", err)
	}

	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(10)
	database.SetConnMaxLifetime(30 * time.Minute)

	return &DuckDB{
		DB:     database,
		DBPath: dbPath,
	}, nil
}

func (d *DuckDB) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

func (d *DuckDB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.DB != nil {
		return d.DB.Close()
	}
	return nil
}
