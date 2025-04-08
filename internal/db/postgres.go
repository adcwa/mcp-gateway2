package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Config holds the PostgreSQL connection parameters
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// DefaultConfig returns the default database configuration
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "admin",
		Password: "Admin123",
		Database: "mcp-gateway",
	}
}

// GetConfig returns the database configuration from environment variables or defaults
func GetConfig() Config {
	config := DefaultConfig()

	// Override with environment variables if present
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Host = host
	}

	if port := os.Getenv("DB_PORT"); port != "" {
		config.Port = port
	}

	if user := os.Getenv("DB_USER"); user != "" {
		config.User = user
	}

	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Password = password
	}

	if database := os.Getenv("DB_NAME"); database != "" {
		config.Database = database
	}

	return config
}

// ConnectDB establishes a connection to the PostgreSQL database
func ConnectDB() (*sql.DB, error) {
	config := GetConfig()

	// Construct the connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.Database)

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	return db, nil
}
