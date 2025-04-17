package repository

import (
	"context"

	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// HTTPInterfaceRepository defines the interface for HTTP interface operations
type HTTPInterfaceRepository interface {
	Create(ctx context.Context, httpInterface *models.HTTPInterface) error
	GetByID(ctx context.Context, id string) (*models.HTTPInterface, error)
	GetAll(ctx context.Context) ([]models.HTTPInterface, error)
	Update(ctx context.Context, httpInterface *models.HTTPInterface) error
	Delete(ctx context.Context, id string) error
	GetVersions(ctx context.Context, id string) ([]int, error)
	GetByVersion(ctx context.Context, id string, version int) (*models.HTTPInterface, error)
}

// MCPServerRepository defines the interface for MCP Server operations
type MCPServerRepository interface {
	Create(ctx context.Context, mcpServer *models.MCPServer) error
	GetByID(ctx context.Context, id string) (*models.MCPServer, error)
	GetByName(ctx context.Context, name string) (*models.MCPServer, error)
	GetAll(ctx context.Context) ([]models.MCPServer, error)
	Update(ctx context.Context, mcpServer *models.MCPServer) error
	Delete(ctx context.Context, id string) error
	GetVersions(ctx context.Context, id string) ([]int, error)
	GetByVersion(ctx context.Context, id string, version int) (*models.MCPServer, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}

// RouterRepository defines the interface for Router operations
type RouterRepository interface {
	Create(ctx context.Context, router *models.Router) error
	GetByID(ctx context.Context, id string) (*models.Router, error)
	GetAll(ctx context.Context) ([]models.Router, error)
	Update(ctx context.Context, router *models.Router) error
	Delete(ctx context.Context, id string) error
	GetVersions(ctx context.Context, id string) ([]int, error)
	GetByVersion(ctx context.Context, id string, version int) (*models.Router, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}
