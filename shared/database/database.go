package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ConnectPostgres establishes a connection to PostgreSQL using pgxpool
func ConnectPostgres(dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// This connects to the database pool.
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	// Make sure we have a connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Log that we have connected successfully
	log.Println("Connected to PostgreSQL")

	return pool, nil
}

// ConnectRedis establishes a connection to Redis
func ConnectRedis(addr string, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,     // "localhost:6379"
		Password: password, // no password set
		DB:       0,        // use default DB
	})

	// Make sure we have a connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	// Log that we have connected successfully
	log.Println("Connected to Redis")

	return client, nil
}
