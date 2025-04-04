package repository

import (
	"context"
	"sync"
	"time"

	"github.com/wangfeng/mcp-gateway2/pkg/models"
)

// InMemoryHTTPInterfaceRepository implements HTTPInterfaceRepository using an in-memory store
type InMemoryHTTPInterfaceRepository struct {
	mu         sync.RWMutex
	interfaces map[string]*models.HTTPInterface
	versions   map[string]map[int]*models.HTTPInterface
	idCounter  int
}

// NewInMemoryHTTPInterfaceRepository creates a new in-memory HTTP interface repository
func NewInMemoryHTTPInterfaceRepository() *InMemoryHTTPInterfaceRepository {
	return &InMemoryHTTPInterfaceRepository{
		interfaces: make(map[string]*models.HTTPInterface),
		versions:   make(map[string]map[int]*models.HTTPInterface),
		idCounter:  0,
	}
}

// Create adds a new HTTP interface to the repository
func (r *InMemoryHTTPInterfaceRepository) Create(ctx context.Context, httpInterface *models.HTTPInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.idCounter++
	httpInterface.ID = generateID("http", r.idCounter)
	httpInterface.CreatedAt = time.Now()
	httpInterface.UpdatedAt = time.Now()
	httpInterface.Version = 1

	r.interfaces[httpInterface.ID] = httpInterface

	// Store version
	if _, ok := r.versions[httpInterface.ID]; !ok {
		r.versions[httpInterface.ID] = make(map[int]*models.HTTPInterface)
	}
	r.versions[httpInterface.ID][httpInterface.Version] = cloneHTTPInterface(httpInterface)

	return nil
}

// GetByID retrieves an HTTP interface by ID
func (r *InMemoryHTTPInterfaceRepository) GetByID(ctx context.Context, id string) (*models.HTTPInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	httpInterface, ok := r.interfaces[id]
	if !ok {
		return nil, ErrNotFound
	}

	return cloneHTTPInterface(httpInterface), nil
}

// GetAll retrieves all HTTP interfaces
func (r *InMemoryHTTPInterfaceRepository) GetAll(ctx context.Context) ([]models.HTTPInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	interfaces := make([]models.HTTPInterface, 0, len(r.interfaces))
	for _, httpInterface := range r.interfaces {
		interfaces = append(interfaces, *cloneHTTPInterface(httpInterface))
	}

	return interfaces, nil
}

// Update updates an HTTP interface
func (r *InMemoryHTTPInterfaceRepository) Update(ctx context.Context, httpInterface *models.HTTPInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.interfaces[httpInterface.ID]
	if !ok {
		return ErrNotFound
	}

	// Increment version
	httpInterface.Version = existing.Version + 1
	httpInterface.UpdatedAt = time.Now()
	httpInterface.CreatedAt = existing.CreatedAt

	r.interfaces[httpInterface.ID] = httpInterface

	// Store version
	if _, ok := r.versions[httpInterface.ID]; !ok {
		r.versions[httpInterface.ID] = make(map[int]*models.HTTPInterface)
	}
	r.versions[httpInterface.ID][httpInterface.Version] = cloneHTTPInterface(httpInterface)

	return nil
}

// Delete removes an HTTP interface
func (r *InMemoryHTTPInterfaceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.interfaces[id]; !ok {
		return ErrNotFound
	}

	delete(r.interfaces, id)
	delete(r.versions, id)

	return nil
}

// GetVersions retrieves all version numbers for an HTTP interface
func (r *InMemoryHTTPInterfaceRepository) GetVersions(ctx context.Context, id string) ([]int, error) {
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

// GetByVersion retrieves a specific version of an HTTP interface
func (r *InMemoryHTTPInterfaceRepository) GetByVersion(ctx context.Context, id string, version int) (*models.HTTPInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.versions[id]; !ok {
		return nil, ErrNotFound
	}

	httpInterface, ok := r.versions[id][version]
	if !ok {
		return nil, ErrNotFound
	}

	return cloneHTTPInterface(httpInterface), nil
}

// Helper function to clone an HTTP interface
func cloneHTTPInterface(httpInterface *models.HTTPInterface) *models.HTTPInterface {
	clone := *httpInterface

	// Clone headers
	if len(httpInterface.Headers) > 0 {
		clone.Headers = make([]models.Header, len(httpInterface.Headers))
		copy(clone.Headers, httpInterface.Headers)
	}

	// Clone parameters
	if len(httpInterface.Parameters) > 0 {
		clone.Parameters = make([]models.Param, len(httpInterface.Parameters))
		copy(clone.Parameters, httpInterface.Parameters)
	}

	// Clone request body
	if httpInterface.RequestBody != nil {
		requestBody := *httpInterface.RequestBody
		clone.RequestBody = &requestBody
	}

	// Clone responses
	if len(httpInterface.Responses) > 0 {
		clone.Responses = make([]models.Response, len(httpInterface.Responses))
		for i, response := range httpInterface.Responses {
			cloneResponse := response
			if response.Body != nil {
				body := *response.Body
				cloneResponse.Body = &body
			}
			clone.Responses[i] = cloneResponse
		}
	}

	return &clone
}
