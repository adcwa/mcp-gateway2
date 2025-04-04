package models

import (
	"time"
)

// MCPServer represents an MCP Server configuration
type MCPServer struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	AllowTools  []string  `json:"allowTools"`
	Tools       []Tool    `json:"tools"`
	Version     int       `json:"version"`
	Status      string    `json:"status" binding:"oneof=draft active inactive"`
	WasmPath    string    `json:"wasmPath,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Tool represents a tool in MCP Server
type Tool struct {
	Name             string           `json:"name" binding:"required"`
	Description      string           `json:"description"`
	RequestTemplate  RequestTemplate  `json:"requestTemplate"`
	ResponseTemplate ResponseTemplate `json:"responseTemplate"`
}

// RequestTemplate represents a request template in MCP Server
type RequestTemplate struct {
	Method  string            `json:"method" binding:"required,oneof=GET POST PUT DELETE PATCH"`
	URL     string            `json:"url" binding:"required"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// ResponseTemplate represents a response template in MCP Server
type ResponseTemplate struct {
	Body string `json:"body"`
}

// ToYAML converts the MCP Server to YAML format
func (m *MCPServer) ToYAML() string {
	// Implementation will be added later
	return ""
}

// FromHTTPInterfaces converts a list of HTTP interfaces to an MCP Server
func NewMCPServerFromHTTPInterfaces(name string, description string, interfaces []HTTPInterface) *MCPServer {
	server := &MCPServer{
		Name:        name,
		Description: description,
		AllowTools:  []string{},
		Tools:       []Tool{},
		Version:     1,
		Status:      "draft",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	for _, httpInterface := range interfaces {
		tool := Tool{
			Name:        httpInterface.Name,
			Description: httpInterface.Description,
			RequestTemplate: RequestTemplate{
				Method: httpInterface.Method,
				URL:    httpInterface.Path,
			},
			ResponseTemplate: ResponseTemplate{
				Body: "", // Will be populated based on response schema
			},
		}

		// Add the tool name to allowed tools
		server.AllowTools = append(server.AllowTools, tool.Name)

		// Add the tool to the server
		server.Tools = append(server.Tools, tool)
	}

	return server
}
