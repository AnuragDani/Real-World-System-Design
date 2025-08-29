# Connection pool prototype

## Goal
Compare PostgreSQL with and without a connection pool using CLI only, show sustained connection count and memory footprint.

## Metrics to capture
- Client, attempted connections, established connections, TPS, average latency, p95, p99, error rate.
- Server, active backends count, process RSS or total memory.
- Pooler, client connections, server connections, if used.
