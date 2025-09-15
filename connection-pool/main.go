/*
Connection Pool Benchmark Demo

This program demonstrates the performance difference between using connection pooling
vs creating new connections for each database operation.

Usage:
  go mod init pool-benchmark
  go mod tidy
  go run main.go [flags]

Flags:
  -db string        Database type: mysql or postgres (default "postgres")
  -dsn string       Database connection string (see examples below)
  -pool-size int    Connection pool size (default 10)
  -concurrency int  Number of concurrent operations (default 50)
  -operations int   Total operations to perform (default 1000)

Example DSNs:
  PostgreSQL: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
  MySQL: "user:password@tcp(localhost:3306)/dbname"

Prerequisites:
  - PostgreSQL or MySQL server running
  - Database and user created with appropriate permissions
*/

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

// Config holds the benchmark configuration
type Config struct {
	DatabaseType string
	DSN          string
	PoolSize     int
	Concurrency  int
	Operations   int
}

// BenchmarkResult holds timing and error information
type BenchmarkResult struct {
	Duration    time.Duration
	Errors      int
	Successful  int
	Description string
}

// ConnectionPool interface for different pool implementations
type ConnectionPool interface {
	Execute(ctx context.Context, query string) error
	Close() error
}

// SQLConnectionPool wraps database/sql connection pool
type SQLConnectionPool struct {
	db *sql.DB
}

func (p *SQLConnectionPool) Execute(ctx context.Context, query string) error {
	_, err := p.db.ExecContext(ctx, query)
	return err
}

func (p *SQLConnectionPool) Close() error {
	return p.db.Close()
}

// PgxConnectionPool wraps pgxpool for PostgreSQL
type PgxConnectionPool struct {
	pool *pgxpool.Pool
}

func (p *PgxConnectionPool) Execute(ctx context.Context, query string) error {
	_, err := p.pool.Exec(ctx, query)
	return err
}

func (p *PgxConnectionPool) Close() error {
	p.pool.Close()
	return nil
}

func main() {
	config := parseFlags()

	fmt.Printf("Connection Pool Benchmark\n")
	fmt.Printf("Database: %s\n", config.DatabaseType)
	fmt.Printf("Pool Size: %d\n", config.PoolSize)
	fmt.Printf("Concurrency: %d\n", config.Concurrency)
	fmt.Printf("Operations: %d\n\n", config.Operations)

	// Test database connectivity first
	if err := testConnection(config); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	fmt.Println("Database connection successful\n")

	// Run benchmarks
	results := []BenchmarkResult{}

	// Benchmark without connection pooling
	fmt.Println("Running benchmark WITHOUT connection pooling...")
	nonPoolResult := benchmarkNonPool(config)
	results = append(results, nonPoolResult)

	// Benchmark with connection pooling
	fmt.Println("Running benchmark WITH connection pooling...")
	poolResult := benchmarkWithPool(config)
	results = append(results, poolResult)

	// Display results
	displayResults(results)
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.DatabaseType, "db", "postgres", "Database type: mysql or postgres")
	flag.StringVar(&config.DSN, "dsn", "", "Database connection string")
	flag.IntVar(&config.PoolSize, "pool-size", 10, "Connection pool size")
	flag.IntVar(&config.Concurrency, "concurrency", 200, "Number of concurrent operations")
	flag.IntVar(&config.Operations, "operations", 5000, "Total operations to perform")

	flag.Parse()

	// Set default DSN if not provided
	if config.DSN == "" {
		switch config.DatabaseType {
		case "postgres":
			config.DSN = "postgres://testuser:testpass@postgres:5432/testdb?sslmode=disable"
		case "mysql":
			config.DSN = "testuser:testpass@tcp(mysql:3306)/testdb"
		default:
			log.Fatalf("Unsupported database type: %s", config.DatabaseType)
		}
	}

	return config
}

func testConnection(config Config) error {
	switch config.DatabaseType {
	case "postgres":
		// Test with pgx
		pool, err := pgxpool.New(context.Background(), config.DSN)
		if err != nil {
			return err
		}
		defer pool.Close()
		return pool.Ping(context.Background())

	case "mysql":
		// Test with database/sql
		db, err := sql.Open("mysql", config.DSN)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()

	default:
		return fmt.Errorf("unsupported database type: %s", config.DatabaseType)
	}
}

func benchmarkNonPool(config Config) BenchmarkResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := 0
	successful := 0

	start := time.Now()

	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, config.Concurrency)

	for i := 0; i < config.Operations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Create new connection for each operation
			if err := executeSingleOperation(config); err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
			} else {
				mu.Lock()
				successful++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	return BenchmarkResult{
		Duration:    duration,
		Errors:      errors,
		Successful:  successful,
		Description: "Non-pooled connections",
	}
}

func executeSingleOperation(config Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch config.DatabaseType {
	case "postgres":
		db, err := sql.Open("postgres", config.DSN)
		if err != nil {
			return err
		}
		defer db.Close()

		// Lightweight operation - just test the connection
		_, err = db.ExecContext(ctx, "SELECT pg_sleep(0.01)")
		return err

	case "mysql":
		db, err := sql.Open("mysql", config.DSN)
		if err != nil {
			return err
		}
		defer db.Close()

		// Lightweight operation
		_, err = db.ExecContext(ctx, "SELECT SLEEP(0.01)")
		return err

	default:
		return fmt.Errorf("unsupported database type: %s", config.DatabaseType)
	}
}

func benchmarkWithPool(config Config) BenchmarkResult {
	// Create connection pool
	pool, err := createConnectionPool(config)
	if err != nil {
		return BenchmarkResult{
			Description: "Pooled connections",
			Errors:      1,
		}
	}
	defer pool.Close()

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := 0
	successful := 0

	start := time.Now()

	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, config.Concurrency)

	for i := 0; i < config.Operations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Use pooled connection
			var query string
			switch config.DatabaseType {
			case "postgres":
				query = "SELECT pg_sleep(0.01)"
			case "mysql":
				query = "SELECT SLEEP(0.01)"
			}

			if err := pool.Execute(ctx, query); err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
			} else {
				mu.Lock()
				successful++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	return BenchmarkResult{
		Duration:    duration,
		Errors:      errors,
		Successful:  successful,
		Description: "Pooled connections",
	}
}

func createConnectionPool(config Config) (ConnectionPool, error) {
	switch config.DatabaseType {
	case "postgres":
		// Use pgxpool for PostgreSQL
		poolConfig, err := pgxpool.ParseConfig(config.DSN)
		if err != nil {
			return nil, err
		}

		poolConfig.MaxConns = int32(config.PoolSize)
		poolConfig.MinConns = int32(config.PoolSize / 4) // Keep some minimum connections

		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			return nil, err
		}

		return &PgxConnectionPool{pool: pool}, nil

	case "mysql":
		// Use database/sql with connection pooling for MySQL
		db, err := sql.Open("mysql", config.DSN)
		if err != nil {
			return nil, err
		}

		db.SetMaxOpenConns(config.PoolSize)
		db.SetMaxIdleConns(config.PoolSize / 2)
		db.SetConnMaxLifetime(5 * time.Minute)

		return &SQLConnectionPool{db: db}, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.DatabaseType)
	}
}

func displayResults(results []BenchmarkResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("=", 60))

	for _, result := range results {
		fmt.Printf("\n%s:\n", result.Description)
		fmt.Printf("  Duration: %v\n", result.Duration)
		fmt.Printf("  Successful: %d\n", result.Successful)
		fmt.Printf("  Errors: %d\n", result.Errors)
		if result.Successful > 0 {
			avgTime := result.Duration / time.Duration(result.Successful)
			fmt.Printf("  Avg per operation: %v\n", avgTime)
			opsPerSec := float64(result.Successful) / result.Duration.Seconds()
			fmt.Printf("  Operations/sec: %.2f\n", opsPerSec)
		}
	}

	// Compare results if we have both
	if len(results) == 2 {
		nonPool := results[0]
		pool := results[1]

		if nonPool.Duration > 0 && pool.Duration > 0 {
			speedup := float64(nonPool.Duration) / float64(pool.Duration)
			fmt.Printf("\nPERFORMANCE COMPARISON:\n")
			fmt.Printf("  Connection pooling is %.2fx faster!\n", speedup)

			if speedup > 1 {
				improvement := ((speedup - 1) * 100)
				fmt.Printf("  That's a %.1f%% improvement in performance.\n", improvement)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}
