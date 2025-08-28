package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Message struct to represent data from Postgres
type Message struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

func main() {
	// Get environment variables with defaults
	postgresHost := getEnv("POSTGRES_HOST", "postgres-service")
	postgresPort := getEnv("POSTGRES_PORT", "5555")
	postgresUser := getEnv("POSTGRES_USER", "postgres")
	postgresPassword := getEnv("POSTGRES_PASSWORD", "password")
	postgresDB := getEnv("POSTGRES_DB", "testdb")
	redisHost := getEnv("REDIS_HOST", "redis-service")
	redisPort := getEnv("REDIS_PORT", "6300")

	// Connect to Postgres
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		postgresHost, postgresPort, postgresUser, postgresPassword, postgresDB)
	
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected to Postgres!")

	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "",
		DB:       0,
	})

	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong)
	fmt.Println("Successfully connected to Redis!")

	// Set up HTTP server
	http.HandleFunc("/redis", func(w http.ResponseWriter, r *http.Request) {
		// Fetch data from Redis (example: get value for key "message")
		val, err := client.Get(context.Background(), "message").Result()
		if err == redis.Nil {
			http.Error(w, "Key not found in Redis", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, fmt.Sprintf("Redis error: %v", err), http.StatusInternalServerError)
			return
		}

		// Prepare JSON response
		response := map[string]string{"key": "message", "value": val}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/postgres", func(w http.ResponseWriter, r *http.Request) {
		// Query Postgres (example: select from messages table)
		rows, err := db.Query("SELECT id, content FROM messages")
		if err != nil {
			http.Error(w, fmt.Sprintf("Postgres query error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Collect results
		var messages []Message
		for rows.Next() {
			var msg Message
			if err := rows.Scan(&msg.ID, &msg.Content); err != nil {
				http.Error(w, fmt.Sprintf("Postgres scan error: %v", err), http.StatusInternalServerError)
				return
			}
			messages = append(messages, msg)
		}

		// Check for errors from iterating over rows
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("Postgres rows error: %v", err), http.StatusInternalServerError)
			return
		}

		// Prepare JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(messages); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
	})

	// Start the server
	fmt.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}