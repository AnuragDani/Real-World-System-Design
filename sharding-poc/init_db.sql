-- Initialize database schema for shards
CREATE TABLE IF NOT EXISTS users (
    user_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
-- Even userIDs will go to shard 0 (hash % 2 == 0)
INSERT INTO users (user_id, name, email) VALUES
(2, 'Alice Johnson', 'alice@example.com'),
(4, 'Bob Smith', 'bob@example.com'),
(6, 'Carol White', 'carol@example.com'),
(8, 'David Brown', 'david@example.com'),
(10, 'Eve Davis', 'eve@example.com');

-- Odd userIDs will go to shard 1 (hash % 2 == 1)
-- Note: These inserts will only work on shard_1
-- For shard_0, they'll fail due to duplicate key, which is expected
INSERT INTO users (user_id, name, email) VALUES
(1, 'Frank Miller', 'frank@example.com'),
(3, 'Grace Lee', 'grace@example.com'),
(5, 'Henry Wilson', 'henry@example.com'),
(7, 'Iris Moore', 'iris@example.com'),
(9, 'Jack Taylor', 'jack@example.com')
ON CONFLICT DO NOTHING;