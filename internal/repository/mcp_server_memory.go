package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

var (
	ErrNotFound = errors.New("not found")
)

// InMemoryMCPServerRepository implements MCPServerRepository using an in-memory store
type InMemoryMCPServerRepository struct {
	mu        sync.RWMutex
	servers   map[string]*models.MCPServer
	versions  map[string]map[int]*models.MCPServer
	idCounter int
}

// NewInMemoryMCPServerRepository creates a new in-memory MCP server repository
func NewInMemoryMCPServerRepository() *InMemoryMCPServerRepository {
	return &InMemoryMCPServerRepository{
		servers:   make(map[string]*models.MCPServer),
		versions:  make(map[string]map[int]*models.MCPServer),
		idCounter: 0,
	}
}

// Create adds a new MCP server to the repository
func (r *InMemoryMCPServerRepository) Create(ctx context.Context, server *models.MCPServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.idCounter++
	server.ID = generateID("mcp", r.idCounter)
	server.CreatedAt = time.Now()
	server.UpdatedAt = time.Now()
	server.Version = 1

	r.servers[server.ID] = server

	// Store version
	if _, ok := r.versions[server.ID]; !ok {
		r.versions[server.ID] = make(map[int]*models.MCPServer)
	}
	r.versions[server.ID][server.Version] = cloneMCPServer(server)

	return nil
}

// GetByID retrieves an MCP server by ID
func (r *InMemoryMCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	server, ok := r.servers[id]
	if !ok {
		return nil, ErrNotFound
	}

	return cloneMCPServer(server), nil
}

// GetAll retrieves all MCP servers
func (r *InMemoryMCPServerRepository) GetAll(ctx context.Context) ([]models.MCPServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]models.MCPServer, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, *cloneMCPServer(server))
	}

	return servers, nil
}

// Update updates an MCP server
func (r *InMemoryMCPServerRepository) Update(ctx context.Context, server *models.MCPServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.servers[server.ID]
	if !ok {
		return ErrNotFound
	}

	// Increment version
	server.Version = existing.Version + 1
	server.UpdatedAt = time.Now()
	server.CreatedAt = existing.CreatedAt

	r.servers[server.ID] = server

	// Store version
	if _, ok := r.versions[server.ID]; !ok {
		r.versions[server.ID] = make(map[int]*models.MCPServer)
	}
	r.versions[server.ID][server.Version] = cloneMCPServer(server)

	return nil
}

// Delete removes an MCP server
func (r *InMemoryMCPServerRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.servers[id]; !ok {
		return ErrNotFound
	}

	delete(r.servers, id)
	delete(r.versions, id)

	return nil
}

// GetVersions retrieves all version numbers for an MCP server
func (r *InMemoryMCPServerRepository) GetVersions(ctx context.Context, id string) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.versions[id]; !ok {
		return nil, ErrNotFound
	}

	versions := make([]int, 0, len(r.versions[id]))
	for v := range r.versions[id] {
		versions = append(versions, v)
	}

	return versions, nil
}

// GetByVersion retrieves a specific version of an MCP server
func (r *InMemoryMCPServerRepository) GetByVersion(ctx context.Context, id string, version int) (*models.MCPServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.versions[id]; !ok {
		return nil, ErrNotFound
	}

	server, ok := r.versions[id][version]
	if !ok {
		return nil, ErrNotFound
	}

	return cloneMCPServer(server), nil
}

// UpdateStatus updates the status of an MCP server
func (r *InMemoryMCPServerRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	server, ok := r.servers[id]
	if !ok {
		return ErrNotFound
	}

	server.Status = status
	server.UpdatedAt = time.Now()

	return nil
}

// Helper function to clone an MCP server
func cloneMCPServer(server *models.MCPServer) *models.MCPServer {
	clone := *server
	clone.AllowTools = make([]string, len(server.AllowTools))
	copy(clone.AllowTools, server.AllowTools)

	clone.Tools = make([]models.Tool, len(server.Tools))
	for i, tool := range server.Tools {
		cloneTool := tool
		if tool.RequestTemplate.Headers != nil {
			cloneTool.RequestTemplate.Headers = make(map[string]string)
			for k, v := range tool.RequestTemplate.Headers {
				cloneTool.RequestTemplate.Headers[k] = v
			}
		}
		clone.Tools[i] = cloneTool
	}

	return &clone
}

// Helper function to generate ID
func generateID(prefix string, counter int) string {
	return prefix + "-" + time.Now().Format("20060102") + "-" + intToString(counter)
}

// Helper function to convert int to string
func intToString(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	return string(buf[pos:])
}
