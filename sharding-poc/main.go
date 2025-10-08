package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type User struct {
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type ShardManager struct {
	shards []*sql.DB
}

// Initialize shard connections
func NewShardManager() (*ShardManager, error) {
	shard0DSN := os.Getenv("SHARD_0_DSN")
	shard1DSN := os.Getenv("SHARD_1_DSN")

	shard0, err := sql.Open("postgres", shard0DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shard 0: %w", err)
	}

	shard1, err := sql.Open("postgres", shard1DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shard 1: %w", err)
	}

	// Test connections
	if err := shard0.Ping(); err != nil {
		return nil, fmt.Errorf("shard 0 ping failed: %w", err)
	}

	if err := shard1.Ping(); err != nil {
		return nil, fmt.Errorf("shard 1 ping failed: %w", err)
	}

	log.Println("✅ Successfully connected to both shards")

	return &ShardManager{
		shards: []*sql.DB{shard0, shard1},
	}, nil
}

// Hash-based sharding: userID % number_of_shards
func (sm *ShardManager) getShardForUser(userID int) *sql.DB {
	shardIndex := userID % len(sm.shards)
	log.Printf("🔀 Routing userID %d to shard %d", userID, shardIndex)
	return sm.shards[shardIndex]
}

// Get user from appropriate shard
func (sm *ShardManager) getUser(c *gin.Context) {
	userIDStr := c.Param("userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Determine which shard to query based on hash
	shard := sm.getShardForUser(userID)

	var user User
	query := `SELECT user_id, name, email, created_at FROM users WHERE user_id = $1`
	err = shard.QueryRow(query, userID).Scan(&user.UserID, &user.Name, &user.Email, &user.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":       user,
		"shard_used": userID % len(sm.shards),
		"routing":    "application_layer",
	})
}

// Create user in appropriate shard
func (sm *ShardManager) createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shard := sm.getShardForUser(user.UserID)

	query := `INSERT INTO users (user_id, name, email) VALUES ($1, $2, $3) RETURNING created_at`
	err := shard.QueryRow(query, user.UserID, user.Name, user.Email).Scan(&user.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":       user,
		"shard_used": user.UserID % len(sm.shards),
		"routing":    "application_layer",
	})
}

// List all users from all shards (scatter-gather pattern)
func (sm *ShardManager) listAllUsers(c *gin.Context) {
	var allUsers []User

	for i, shard := range sm.shards {
		query := `SELECT user_id, name, email, created_at FROM users ORDER BY user_id`
		rows, err := shard.Query(query)
		if err != nil {
			log.Printf("Error querying shard %d: %v", i, err)
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var user User
			if err := rows.Scan(&user.UserID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
				log.Printf("Error scanning row from shard %d: %v", i, err)
				continue
			}
			allUsers = append(allUsers, user)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users":   allUsers,
		"count":   len(allUsers),
		"routing": "scatter_gather_across_all_shards",
	})
}

func main() {
	// Initialize shard manager
	sm, err := NewShardManager()
	if err != nil {
		log.Fatalf("Failed to initialize shard manager: %v", err)
	}

	// Setup Gin router
	r := gin.Default()

	// Application-layer routing endpoints
	r.GET("/user/:userID", sm.getUser)
	r.POST("/user", sm.createUser)
	r.GET("/users", sm.listAllUsers)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"shards": len(sm.shards),
		})
	})

	// Info endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Sharding POC API Server",
			"endpoints": map[string]string{
				"GET /user/:userID": "Get user by ID (hash-based routing)",
				"POST /user":        "Create new user (hash-based routing)",
				"GET /users":        "List all users (scatter-gather)",
				"GET /health":       "Health check",
			},
			"sharding_strategy": "hash-based (userID % num_shards)",
			"num_shards":        len(sm.shards),
		})
	})

	log.Println("API Server starting on :8080")
	log.Println("Sharding Strategy: Hash-based (userID % 2)")
	log.Println("Endpoints available:")
	log.Println("   - GET  /user/:userID (application-layer routing)")
	log.Println("   - POST /user (application-layer routing)")
	log.Println("   - GET  /users (scatter-gather)")
	log.Println("   - GET  /health")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
