package models

import (
	"time"
)

// HTTPInterface represents an API configuration
type HTTPInterface struct {
	ID          string     `json:"id"`
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Method      string     `json:"method" binding:"required,oneof=GET POST PUT DELETE PATCH"`
	Path        string     `json:"path" binding:"required"`
	Headers     []Header   `json:"headers"`
	Parameters  []Param    `json:"parameters"`
	RequestBody *Body      `json:"requestBody,omitempty"`
	Responses   []Response `json:"responses"`
	Version     int        `json:"version"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Header represents an HTTP header
type Header struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type" binding:"required,oneof=string integer number boolean array object"`
}

// Param represents a request parameter (query or path)
type Param struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	In          string `json:"in" binding:"required,oneof=query path header"`
	Required    bool   `json:"required"`
	Type        string `json:"type" binding:"required,oneof=string integer number boolean array object"`
	Schema      string `json:"schema,omitempty"`
}

// Body represents a request or response body
type Body struct {
	ContentType string `json:"contentType" binding:"required"`
	Schema      string `json:"schema" binding:"required"`
	Example     string `json:"example,omitempty"`
}

// Response represents an API response
type Response struct {
	StatusCode  int    `json:"statusCode" binding:"required"`
	Description string `json:"description"`
	Body        *Body  `json:"body,omitempty"`
}

// ConvertToOpenAPI converts the HTTP interface to OpenAPI format
func (h *HTTPInterface) ConvertToOpenAPI() map[string]interface{} {
	// Implementation will be added later
	return map[string]interface{}{}
}

// ConvertToMCPServerYAML converts the HTTP interface to MCP Server YAML format
func (h *HTTPInterface) ConvertToMCPServerYAML() string {
	// Implementation will be added later
	return ""
}
