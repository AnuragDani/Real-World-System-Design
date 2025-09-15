# Connection Pool Benchmark

A Go application that demonstrates the performance difference between using connection pooling vs creating new connections for each database operation.

## Features

- Support for PostgreSQL and MySQL databases
- Configurable connection pool sizes and concurrency levels
- Performance metrics including operations per second and average operation time
- Docker containerization for easy deployment
- Comprehensive benchmarking with error tracking
- Realistic database workload simulation

## Quick Start with Docker

### Option 1: Using Docker Compose (Recommended)

This will start PostgreSQL, MySQL databases and run the benchmark:

```bash
# Clone/download the project files
# Ensure you have: main.go, Dockerfile, docker-compose.yml, go.mod

# Start all services
docker-compose up --build

# To run with different parameters
docker-compose run --rm benchmark ./main -concurrency 100 -operations 2000

# To test MySQL instead
docker-compose run --rm benchmark ./main -db mysql

# High concurrency test
docker-compose run --rm benchmark ./main -concurrency 500 -operations 10000
```

### Option 2: Manual Docker Setup

```bash
# Start PostgreSQL
docker run --name postgres-db -e POSTGRES_DB=testdb -e POSTGRES_USER=testuser -e POSTGRES_PASSWORD=testpass -p 5432:5432 -d postgres:15-alpine

# Build and run the benchmark
docker build -t connection-benchmark .
docker run --rm --link postgres-db connection-benchmark ./main -dsn "postgres://testuser:testpass@postgres-db:5432/testdb?sslmode=disable"
```

## Local Development

### Prerequisites

- Go 1.21 or higher
- PostgreSQL or MySQL server
- Database and user with appropriate permissions

### Setup

```bash
# Initialize Go module
go mod init connection-pool-benchmark
go mod tidy

# Run the application
go run main.go [flags]
```

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-db` | `postgres` | Database type: `mysql` or `postgres` |
| `-dsn` | (auto-generated) | Database connection string |
| `-pool-size` | `10` | Connection pool size |
| `-concurrency` | `200` | Number of concurrent operations |
| `-operations` | `5000` | Total operations to perform |

## Example Usage

```bash
# Basic PostgreSQL test
go run main.go

# MySQL with custom parameters
go run main.go -db mysql -dsn "user:pass@tcp(localhost:3306)/dbname" -concurrency 100 -operations 2000

# High concurrency PostgreSQL test
go run main.go -dsn "postgres://user:pass@localhost:5432/db?sslmode=disable" -concurrency 500 -pool-size 20

# Quick test with Docker
docker-compose run --rm benchmark ./main

# Stress test
docker-compose run --rm benchmark ./main -concurrency 1000 -operations 20000
```

## Connection String Examples

### PostgreSQL
```
postgres://username:password@localhost:5432/database?sslmode=disable
```

### MySQL
```
username:password@tcp(localhost:3306)/database
```

## Expected Output

The benchmark will show:

```
Connection Pool Benchmark
Database: postgres
Pool Size: 10
Concurrency: 200
Operations: 5000

Database connection successful

Running benchmark WITHOUT connection pooling...
Running benchmark WITH connection pooling...

============================================================
BENCHMARK RESULTS
============================================================

Non-pooled connections:
  Duration: 2.543s
  Successful: 5000
  Errors: 0
  Avg per operation: 508.6µs
  Operations/sec: 1965.25

Pooled connections:
  Duration: 890.2ms
  Successful: 5000
  Errors: 0
  Avg per operation: 178.04µs
  Operations/sec: 5618.45

PERFORMANCE COMPARISON:
  Connection pooling is 2.86x faster!
  That's a 185.7% improvement in performance.
============================================================
```

## Understanding the Results

### What the benchmark tests:

1. **Non-pooled**: Creates a new database connection for each operation
   - Higher connection overhead
   - TCP handshake for each operation
   - Authentication for each connection

2. **Pooled**: Reuses connections from a managed pool
   - Connections are established once and reused
   - No repeated handshakes
   - Better resource utilization

### Realistic Workload:

The benchmark simulates realistic database operations:
- Multiple SQL queries per operation
- Transaction usage
- Information schema queries
- Timestamp operations

## Performance Insights

Connection pooling typically shows:

- **2-10x performance improvement** under high concurrency
- **Reduced connection overhead** - no TCP handshake per operation
- **Better resource utilization** - controlled connection limits
- **Lower database server load** - fewer connection/disconnection cycles
- **More consistent latency** - no connection establishment delays

### When pooling provides the most benefit:

- **High concurrency** (100+ concurrent operations)
- **Frequent database operations**
- **Network latency between app and database**
- **Limited database connection limits**
- **Long-running applications**

## Architecture

The benchmark demonstrates two approaches:

1. **Non-pooled**: Creates a new database connection for each operation
2. **Pooled**: Reuses connections from a managed connection pool

### Connection Pool Implementations

- **PostgreSQL**: Uses `pgxpool` (native PostgreSQL connection pool)
- **MySQL**: Uses `database/sql` built-in connection pooling

## Troubleshooting

### Database Connection Issues

1. Verify database is running and accessible
2. Check connection string format
3. Ensure user has proper permissions
4. For Docker: ensure containers can communicate

### Common Docker Issues

```bash
# Check if containers are running
docker-compose ps

# View logs
docker-compose logs postgres
docker-compose logs benchmark

# Restart services
docker-compose restart

# Remove version warning (optional)
# Edit docker-compose.yml and remove the "version: '3.8'" line
```

### Performance Tuning

- Increase pool size for higher concurrency workloads
- Adjust concurrency based on your system capabilities
- Monitor database connection limits
- Use appropriate timeout values for your network conditions

### If pooling appears slower:

This can happen in specific scenarios:
- Very low latency networks (containers on same host)
- Simple queries with minimal database work
- Pool size too small for concurrency level
- Container resource constraints

Try:
```bash
# Increase pool size
docker-compose run --rm benchmark ./main -pool-size 50 -concurrency 500

# Increase workload complexity
docker-compose run --rm benchmark ./main -operations 20000

# Test on different environments (cloud, separate hosts)
```

## Dependencies

- `github.com/jackc/pgx/v5/pgxpool` - PostgreSQL driver and connection pool
- `github.com/lib/pq` - PostgreSQL driver for database/sql
- `github.com/go-sql-driver/mysql` - MySQL driver

## Real-World Applications

This benchmark is useful for:

- **System design interviews** - Demonstrating connection pooling concepts
- **Performance optimization** - Measuring pooling benefits in your environment
- **Architecture decisions** - Choosing appropriate pool sizes
- **Load testing** - Understanding database connection limits
- **Educational purposes** - Learning about database connection management

## License

This project is provided as-is for educational and benchmarking purposes.