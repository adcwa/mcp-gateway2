package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// PgMCPServerRepository is a PostgreSQL implementation of MCPServerRepository
type PgMCPServerRepository struct {
	db *sql.DB
}

// NewPgMCPServerRepository creates a new PostgreSQL-based MCP server repository
func NewPgMCPServerRepository(db *sql.DB) *PgMCPServerRepository {
	return &PgMCPServerRepository{
		db: db,
	}
}

// Initialize creates the necessary tables if they don't exist
func (r *PgMCPServerRepository) Initialize(ctx context.Context) error {
	// Create mcp_servers table
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS mcp_servers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			tools JSONB,
			allow_tools JSONB,
			status TEXT NOT NULL,
			version INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	return err
}

// GetAll returns all MCP servers
func (r *PgMCPServerRepository) GetAll(ctx context.Context) ([]models.MCPServer, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, tools, allow_tools, status, version, created_at, updated_at
		FROM mcp_servers
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.MCPServer
	for rows.Next() {
		var server models.MCPServer
		var toolsJSON, allowToolsJSON []byte

		// Scan rows into variables
		err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Description,
			&toolsJSON,
			&allowToolsJSON,
			&server.Status,
			&server.Version,
			&server.CreatedAt,
			&server.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal tools
		if err := json.Unmarshal(toolsJSON, &server.Tools); err != nil {
			return nil, err
		}

		// Unmarshal allow tools
		if err := json.Unmarshal(allowToolsJSON, &server.AllowTools); err != nil {
			return nil, err
		}

		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

// GetByID returns a specific MCP server by ID
func (r *PgMCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServer, error) {
	var server models.MCPServer
	var toolsJSON, allowToolsJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, tools, allow_tools, status, version, created_at, updated_at
		FROM mcp_servers
		WHERE id = $1
	`, id).Scan(
		&server.ID,
		&server.Name,
		&server.Description,
		&toolsJSON,
		&allowToolsJSON,
		&server.Status,
		&server.Version,
		&server.CreatedAt,
		&server.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	// Unmarshal tools
	if err := json.Unmarshal(toolsJSON, &server.Tools); err != nil {
		return nil, err
	}

	// Unmarshal allow tools
	if err := json.Unmarshal(allowToolsJSON, &server.AllowTools); err != nil {
		return nil, err
	}

	return &server, nil
}

// Create creates a new MCP server
func (r *PgMCPServerRepository) Create(ctx context.Context, server *models.MCPServer) error {
	// Generate ID if not provided
	if server.ID == "" {
		server.ID = fmt.Sprintf("mcp-%s", uuid.New().String())
	}

	// Set version and timestamps
	server.Version = 1
	now := time.Now()
	server.CreatedAt = now
	server.UpdatedAt = now

	// Set status if not provided
	if server.Status == "" {
		server.Status = "draft" // Default status
	}

	// Serialize complex types to JSON
	toolsJSON, err := json.Marshal(server.Tools)
	if err != nil {
		return err
	}

	allowToolsJSON, err := json.Marshal(server.AllowTools)
	if err != nil {
		return err
	}

	// Insert the MCP server
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO mcp_servers (
			id, name, description, tools, allow_tools, status, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		server.ID,
		server.Name,
		server.Description,
		toolsJSON,
		allowToolsJSON,
		server.Status,
		server.Version,
		server.CreatedAt,
		server.UpdatedAt,
	)

	return err
}

// Update updates an existing MCP server
func (r *PgMCPServerRepository) Update(ctx context.Context, server *models.MCPServer) error {
	// Get current version
	var currentVersion int
	err := r.db.QueryRowContext(ctx, `
		SELECT version FROM mcp_servers WHERE id = $1
	`, server.ID).Scan(&currentVersion)

	if err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}

	// Set new version and update timestamp
	server.Version = currentVersion + 1
	server.UpdatedAt = time.Now()

	// Serialize complex types to JSON
	toolsJSON, err := json.Marshal(server.Tools)
	if err != nil {
		return err
	}

	allowToolsJSON, err := json.Marshal(server.AllowTools)
	if err != nil {
		return err
	}

	// Update the MCP server
	result, err := r.db.ExecContext(ctx, `
		UPDATE mcp_servers SET
			name = $1,
			description = $2,
			tools = $3,
			allow_tools = $4,
			status = $5,
			version = $6,
			updated_at = $7
		WHERE id = $8
	`,
		server.Name,
		server.Description,
		toolsJSON,
		allowToolsJSON,
		server.Status,
		server.Version,
		server.UpdatedAt,
		server.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes an MCP server
func (r *PgMCPServerRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM mcp_servers WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetVersions returns all version numbers for an MCP server
func (r *PgMCPServerRepository) GetVersions(ctx context.Context, id string) ([]int, error) {
	// In a real implementation, you'd store past versions
	// For this simplified version, just return the current version
	var version int
	err := r.db.QueryRowContext(ctx, "SELECT version FROM mcp_servers WHERE id = $1", id).Scan(&version)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return []int{version}, nil
}

// GetByVersion retrieves a specific version of an MCP server
func (r *PgMCPServerRepository) GetByVersion(ctx context.Context, id string, version int) (*models.MCPServer, error) {
	// In a real implementation, you'd retrieve the specific version
	// For this simplified version, just return the current version if it matches
	server, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if server.Version != version {
		return nil, ErrNotFound
	}

	return server, nil
}

// UpdateStatus updates the status of an MCP server
func (r *PgMCPServerRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE mcp_servers SET
			status = $1,
			updated_at = $2
		WHERE id = $3
	`, status, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
