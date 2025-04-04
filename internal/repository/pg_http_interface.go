package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// PgHTTPInterfaceRepository is a PostgreSQL implementation of HTTPInterfaceRepository
type PgHTTPInterfaceRepository struct {
	db *sql.DB
}

// NewPgHTTPInterfaceRepository creates a new PostgreSQL-based HTTP interface repository
func NewPgHTTPInterfaceRepository(db *sql.DB) *PgHTTPInterfaceRepository {
	return &PgHTTPInterfaceRepository{
		db: db,
	}
}

// Initialize creates the necessary tables if they don't exist
func (r *PgHTTPInterfaceRepository) Initialize(ctx context.Context) error {
	// Create http_interfaces table
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS http_interfaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			headers JSONB,
			parameters JSONB,
			request_body JSONB,
			responses JSONB,
			version INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	return err
}

// GetAll returns all HTTP interfaces
func (r *PgHTTPInterfaceRepository) GetAll(ctx context.Context) ([]models.HTTPInterface, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, method, path, headers, parameters, request_body, responses, version, created_at, updated_at
		FROM http_interfaces
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interfaces []models.HTTPInterface
	for rows.Next() {
		var iface models.HTTPInterface
		var headersJSON, paramsJSON, responsesJSON []byte
		var requestBodyJSON sql.NullString

		// Scan rows into variables
		err := rows.Scan(
			&iface.ID,
			&iface.Name,
			&iface.Description,
			&iface.Method,
			&iface.Path,
			&headersJSON,
			&paramsJSON,
			&requestBodyJSON,
			&responsesJSON,
			&iface.Version,
			&iface.CreatedAt,
			&iface.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal headers
		if err := json.Unmarshal(headersJSON, &iface.Headers); err != nil {
			return nil, err
		}

		// Unmarshal parameters
		if err := json.Unmarshal(paramsJSON, &iface.Parameters); err != nil {
			return nil, err
		}

		// Unmarshal request body (if exists)
		if requestBodyJSON.Valid {
			var requestBody models.Body
			if err := json.Unmarshal([]byte(requestBodyJSON.String), &requestBody); err != nil {
				return nil, err
			}
			iface.RequestBody = &requestBody
		}

		// Unmarshal responses
		if err := json.Unmarshal(responsesJSON, &iface.Responses); err != nil {
			return nil, err
		}

		interfaces = append(interfaces, iface)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return interfaces, nil
}

// GetByID returns a specific HTTP interface by ID
func (r *PgHTTPInterfaceRepository) GetByID(ctx context.Context, id string) (*models.HTTPInterface, error) {
	var iface models.HTTPInterface
	var headersJSON, paramsJSON, responsesJSON []byte
	var requestBodyJSON sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, method, path, headers, parameters, request_body, responses, version, created_at, updated_at
		FROM http_interfaces
		WHERE id = $1
	`, id).Scan(
		&iface.ID,
		&iface.Name,
		&iface.Description,
		&iface.Method,
		&iface.Path,
		&headersJSON,
		&paramsJSON,
		&requestBodyJSON,
		&responsesJSON,
		&iface.Version,
		&iface.CreatedAt,
		&iface.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	// Unmarshal headers
	if err := json.Unmarshal(headersJSON, &iface.Headers); err != nil {
		return nil, err
	}

	// Unmarshal parameters
	if err := json.Unmarshal(paramsJSON, &iface.Parameters); err != nil {
		return nil, err
	}

	// Unmarshal request body (if exists)
	if requestBodyJSON.Valid {
		var requestBody models.Body
		if err := json.Unmarshal([]byte(requestBodyJSON.String), &requestBody); err != nil {
			return nil, err
		}
		iface.RequestBody = &requestBody
	}

	// Unmarshal responses
	if err := json.Unmarshal(responsesJSON, &iface.Responses); err != nil {
		return nil, err
	}

	return &iface, nil
}

// Create creates a new HTTP interface
func (r *PgHTTPInterfaceRepository) Create(ctx context.Context, httpInterface *models.HTTPInterface) error {
	// Generate ID if not provided
	if httpInterface.ID == "" {
		httpInterface.ID = fmt.Sprintf("http-%s", uuid.New().String())
	}

	// Set version and timestamps
	httpInterface.Version = 1
	now := time.Now()
	httpInterface.CreatedAt = now
	httpInterface.UpdatedAt = now

	// Serialize complex types to JSON
	headersJSON, err := json.Marshal(httpInterface.Headers)
	if err != nil {
		return err
	}

	paramsJSON, err := json.Marshal(httpInterface.Parameters)
	if err != nil {
		return err
	}

	var requestBodyStr sql.NullString
	if httpInterface.RequestBody != nil {
		requestBodyJSON, err := json.Marshal(httpInterface.RequestBody)
		if err != nil {
			return err
		}
		requestBodyStr = sql.NullString{String: string(requestBodyJSON), Valid: true}
	}

	responsesJSON, err := json.Marshal(httpInterface.Responses)
	if err != nil {
		return err
	}

	// Insert the HTTP interface
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO http_interfaces (
			id, name, description, method, path, headers, parameters, 
			request_body, responses, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		httpInterface.ID,
		httpInterface.Name,
		httpInterface.Description,
		httpInterface.Method,
		httpInterface.Path,
		headersJSON,
		paramsJSON,
		requestBodyStr,
		responsesJSON,
		httpInterface.Version,
		httpInterface.CreatedAt,
		httpInterface.UpdatedAt,
	)

	return err
}

// Update updates an existing HTTP interface
func (r *PgHTTPInterfaceRepository) Update(ctx context.Context, httpInterface *models.HTTPInterface) error {
	// Retrieve the current version
	var currentVersion int
	err := r.db.QueryRowContext(ctx, `
		SELECT version FROM http_interfaces WHERE id = $1
	`, httpInterface.ID).Scan(&currentVersion)

	if err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}

	// Increment version and update timestamp
	httpInterface.Version = currentVersion + 1
	httpInterface.UpdatedAt = time.Now()

	// Serialize complex types to JSON
	headersJSON, err := json.Marshal(httpInterface.Headers)
	if err != nil {
		return err
	}

	paramsJSON, err := json.Marshal(httpInterface.Parameters)
	if err != nil {
		return err
	}

	var requestBodyStr sql.NullString
	if httpInterface.RequestBody != nil {
		requestBodyJSON, err := json.Marshal(httpInterface.RequestBody)
		if err != nil {
			return err
		}
		requestBodyStr = sql.NullString{String: string(requestBodyJSON), Valid: true}
	}

	responsesJSON, err := json.Marshal(httpInterface.Responses)
	if err != nil {
		return err
	}

	// Update the HTTP interface
	result, err := r.db.ExecContext(ctx, `
		UPDATE http_interfaces SET
			name = $1,
			description = $2,
			method = $3,
			path = $4,
			headers = $5,
			parameters = $6,
			request_body = $7,
			responses = $8,
			version = $9,
			updated_at = $10
		WHERE id = $11
	`,
		httpInterface.Name,
		httpInterface.Description,
		httpInterface.Method,
		httpInterface.Path,
		headersJSON,
		paramsJSON,
		requestBodyStr,
		responsesJSON,
		httpInterface.Version,
		httpInterface.UpdatedAt,
		httpInterface.ID,
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

// Delete deletes an HTTP interface by ID
func (r *PgHTTPInterfaceRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM http_interfaces WHERE id = $1
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

// GetVersions returns all versions of a specific HTTP interface
// Note: In this implementation, we only store the current version
// so this will just return a single-element array with the current version number
func (r *PgHTTPInterfaceRepository) GetVersions(ctx context.Context, id string) ([]int, error) {
	var version int
	err := r.db.QueryRowContext(ctx, `
		SELECT version FROM http_interfaces WHERE id = $1
	`, id).Scan(&version)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return []int{version}, nil
}

// GetByVersion returns a specific version of an HTTP interface
// Note: In this implementation, we only store the current version
// so this will just return the current interface if version matches
func (r *PgHTTPInterfaceRepository) GetByVersion(ctx context.Context, id string, version int) (*models.HTTPInterface, error) {
	iface, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if iface.Version != version {
		return nil, ErrNotFound
	}

	return iface, nil
}
