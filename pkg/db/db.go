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
	connStr := fmt.Sprintf("%s?access_mode=READ_ONLY", dbPath)
	database, err := sql.Open("duckdb", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb at %s: %w", dbPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to ping duckdb database: %w", err)
	}

	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(10)
	database.SetConnMaxLifetime(30 * time.Minute)

	log.Printf("Successfully connected to DuckDB database: %s", dbPath)
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
